#!/usr/bin/env python3
"""
实时监控 x-ui 数据库中的流量统计
用于诊断流量更新延迟问题
"""

import sqlite3
import time
import sys
from datetime import datetime

DB_PATH = "/etc/x-ui/x-ui.db"
INTERVAL = 2  # 监控间隔（秒）

def format_bytes(bytes_val):
    """格式化字节数为可读格式"""
    if bytes_val is None:
        return "0 B"
    
    for unit in ['B', 'KB', 'MB', 'GB', 'TB']:
        if bytes_val < 1024.0:
            return f"{bytes_val:.2f} {unit}"
        bytes_val /= 1024.0
    return f"{bytes_val:.2f} PB"

def get_inbound_traffic(conn):
    """获取所有inbound的流量数据"""
    cursor = conn.cursor()
    cursor.execute("""
        SELECT id, tag, up, down, slave_id
        FROM inbounds
        ORDER BY slave_id, id
    """)
    return cursor.fetchall()

def monitor_traffic():
    """主监控循环"""
    print(f"=== X-UI 流量监控工具 ===")
    print(f"数据库: {DB_PATH}")
    print(f"更新间隔: {INTERVAL}秒")
    print(f"开始时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print("=" * 100)
    print()
    
    # 存储上一次的数据
    last_data = {}
    
    try:
        conn = sqlite3.connect(DB_PATH)
        iteration = 0
        
        while True:
            iteration += 1
            current_time = datetime.now().strftime('%H:%M:%S')
            
            print(f"\n[{current_time}] 第 {iteration} 次检查:")
            print("-" * 100)
            
            # 获取当前数据
            current_data = get_inbound_traffic(conn)
            
            # 显示每个inbound的流量
            for row in current_data:
                inbound_id, tag, up, down, slave_id = row
                key = inbound_id
                
                # 计算增量
                if key in last_data:
                    last_up, last_down = last_data[key]
                    up_delta = up - last_up if up and last_up else 0
                    down_delta = down - last_down if down and last_down else 0
                    delta_str = f"  [+{format_bytes(up_delta)} ↑ / +{format_bytes(down_delta)} ↓]"
                    
                    # 高亮有变化的行
                    if up_delta > 0 or down_delta > 0:
                        delta_str = f"  *** {delta_str} ***"
                else:
                    delta_str = "  [新记录]"
                
                # 显示信息
                print(f"  Slave {slave_id} | ID={inbound_id:2d} | {tag:20s} | "
                      f"↑ {format_bytes(up):>12s} | ↓ {format_bytes(down):>12s}{delta_str}")
                
                # 更新last_data
                last_data[key] = (up, down)
            
            # 总计统计
            total_up = sum(row[2] or 0 for row in current_data)
            total_down = sum(row[3] or 0 for row in current_data)
            print("-" * 100)
            print(f"  总计: ↑ {format_bytes(total_up)} | ↓ {format_bytes(total_down)}")
            
            # 等待下一次检查
            time.sleep(INTERVAL)
            
    except KeyboardInterrupt:
        print("\n\n监控已停止")
        conn.close()
        sys.exit(0)
    except sqlite3.Error as e:
        print(f"\n数据库错误: {e}")
        print("请确保:")
        print("1. 数据库文件存在: /etc/x-ui/x-ui.db")
        print("2. 有读取权限 (可能需要 sudo)")
        sys.exit(1)
    except Exception as e:
        print(f"\n未知错误: {e}")
        sys.exit(1)

if __name__ == "__main__":
    print("提示: 按 Ctrl+C 停止监控\n")
    monitor_traffic()
