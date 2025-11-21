# analytics/main.py
import os
import json
import time
import logging
import pymysql
from kafka import KafkaConsumer
from user_agents import parse

# 配置日志格式
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("AnalyticsService")

# 环境变量
KAFKA_SERVER = os.getenv('KAFKA_BOOTSTRAP_SERVERS', 'kafka:9092')
# 默认给个端口，防止 split 报错
DB_HOST_STR = os.getenv('DB_HOST', 'db:3306') 
DB_USER = 'root'
DB_PASSWORD = 'root_password'
DB_NAME = 'tinylink'
TOPIC_NAME = 'link_clicks'

def get_db_connection():
    """连接 MySQL，带有重试和容错逻辑"""
    while True:
        try:
            # 容错处理：如果地址里没有冒号，默认使用 3306
            if ':' in DB_HOST_STR:
                host, port = DB_HOST_STR.split(':')
                port = int(port)
            else:
                host = DB_HOST_STR
                port = 3306

            conn = pymysql.connect(
                host=host,
                port=port,
                user=DB_USER,
                password=DB_PASSWORD,
                database=DB_NAME,
                cursorclass=pymysql.cursors.DictCursor,
                autocommit=True,
                # 设置连接超时，防止卡死
                connect_timeout=10
            )
            logger.info("Successfully connected to MySQL!")
            return conn
        except Exception as e:
            logger.warning(f"MySQL connection failed: {e}. Retrying in 5s...")
            time.sleep(5)

def ensure_table_exists(conn):
    """确保表存在"""
    try:
        # 使用 Ping 确保连接是活的
        conn.ping(reconnect=True)
        sql = """
        CREATE TABLE IF NOT EXISTS click_stats (
            id BIGINT AUTO_INCREMENT PRIMARY KEY,
            short_url VARCHAR(20),
            long_url TEXT,
            ip VARCHAR(45),
            browser VARCHAR(50),
            os VARCHAR(50),
            device VARCHAR(50),
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
        """
        with conn.cursor() as cursor:
            cursor.execute(sql)
        logger.info("Table 'click_stats' checked/created.")
    except Exception as e:
        logger.error(f"Failed to create table: {e}")
        raise e

def start_consumer():
    # 1. 准备数据库
    db_conn = get_db_connection()
    ensure_table_exists(db_conn)

    # 2. 连接 Kafka (配置已优化)
    logger.info(f"Connecting to Kafka at {KAFKA_SERVER}...")
    consumer = None
    while not consumer:
        try:
            consumer = KafkaConsumer(
                TOPIC_NAME,
                bootstrap_servers=KAFKA_SERVER,
                group_id='analytics-group',
                auto_offset_reset='earliest',
                value_deserializer=lambda x: json.loads(x.decode('utf-8')),
                
                # --- 关键参数调整，防止重平衡崩溃 ---
                max_poll_records=10,         # 一次只拿10条，处理完再拿
                max_poll_interval_ms=300000, # 允许处理间隔最长 5分钟
                session_timeout_ms=30000,    # 心跳超时 30秒
                heartbeat_interval_ms=10000  # 每10秒发心跳
            )
            logger.info("Successfully connected to Kafka!")
        except Exception as e:
            logger.warning(f"Kafka connection failed: {e}. Retrying in 5s...")
            time.sleep(5)

    logger.info(f"Listening for messages on '{TOPIC_NAME}'...")
    
    # 3. 消费循环
    for message in consumer:
        try:
            # --- 关键：每次写入前，检查数据库连接是否存活 ---
            # Kafka 重平衡期间可能耗时很久，MySQL 连接容易断开
            db_conn.ping(reconnect=True)

            data = message.value
            logger.info(f"Processing event for short_url: {data.get('short_url')}")

            # 解析 UA
            ua_string = data.get('user_agent', '')
            user_agent = parse(ua_string)
            browser = user_agent.browser.family
            os_info = user_agent.os.family
            device = user_agent.device.family

            # 入库
            sql = """
            INSERT INTO click_stats (short_url, long_url, ip, browser, os, device)
            VALUES (%s, %s, %s, %s, %s, %s)
            """
            with db_conn.cursor() as cursor:
                cursor.execute(sql, (
                    data.get('short_url'),
                    data.get('long_url'),
                    data.get('ip'),
                    browser,
                    os_info,
                    device
                ))
            
        except Exception as e:
            logger.error(f"Error processing message: {e}")

if __name__ == "__main__":
    start_consumer()