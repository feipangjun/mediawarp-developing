#!/usr/bin/env python3
"""
FontInAss 启动脚本
移除 nginx 依赖，直接启动 Python 服务
"""

import os
import sys
import subprocess
import time
from pathlib import Path

def main():
    # 设置 Python 路径
    src_dir = Path(__file__).parent / "src"
    sys.path.insert(0, str(src_dir))
    
    # 启动 FontInAss 服务
    print("启动 FontInAss 服务...")
    
    # 直接导入并运行 main 模块
    try:
        from main import main as fontinass_main
        fontinass_main()
    except ImportError as e:
        print(f"导入 FontInAss 模块失败: {e}")
        print("尝试直接运行 main.py...")
        
        # 如果导入失败，直接运行 main.py
        main_py = src_dir / "main.py"
        if main_py.exists():
            subprocess.run([sys.executable, str(main_py)])
        else:
            print(f"找不到 main.py 文件: {main_py}")
            sys.exit(1)
    except KeyboardInterrupt:
        print("\nFontInAss 服务已停止")
    except Exception as e:
        print(f"FontInAss 服务运行异常: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()