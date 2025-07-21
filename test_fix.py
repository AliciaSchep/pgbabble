#!/usr/bin/env python3
"""
Test script to verify that Ctrl+C cancellation works correctly.
This will simulate pressing Ctrl+C during a long-running query.
"""

import subprocess
import time
import signal
import sys
import os

def test_ctrl_c_cancellation():
    """Test that Ctrl+C cancels query but keeps session alive"""
    
    # Connection string for test database
    conn_str = "postgresql://testuser:testpass@localhost:5433/testdb"
    
    print("🧪 Testing Ctrl+C cancellation behavior...")
    print("Starting pgbabble session...")
    
    # Start pgbabble process
    process = subprocess.Popen(
        ["./pgbabble", conn_str],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        bufsize=1
    )
    
    try:
        # Wait for connection to establish
        time.sleep(2)
        
        # Send a long-running query
        print("📊 Sending long-running query (SELECT pg_sleep(10))...")
        process.stdin.write("SELECT pg_sleep(10);\n")
        process.stdin.flush()
        
        # Wait a bit for query to start
        time.sleep(1)
        
        # Send SIGINT (Ctrl+C)
        print("⚡ Sending SIGINT (Ctrl+C)...")
        process.send_signal(signal.SIGINT)
        
        # Wait to see if process handles cancellation
        time.sleep(2)
        
        # Try to send another query to test if session is still alive
        print("🔄 Testing if session is still alive with a simple query...")
        process.stdin.write("SELECT 1 as test_value;\n")
        process.stdin.flush()
        
        # Wait for response
        time.sleep(2)
        
        # Send quit command
        process.stdin.write("/quit\n")
        process.stdin.flush()
        
        # Wait for process to finish
        stdout, stderr = process.communicate(timeout=5)
        
        print("📤 Process output:")
        print(stdout)
        if stderr:
            print("📥 Process errors:")
            print(stderr)
            
        if process.returncode == 0:
            print("✅ SUCCESS: Process exited cleanly")
            return True
        else:
            print(f"❌ FAIL: Process exited with code {process.returncode}")
            return False
            
    except subprocess.TimeoutExpired:
        print("❌ FAIL: Process did not exit within timeout")
        process.kill()
        return False
    except Exception as e:
        print(f"❌ FAIL: Exception occurred: {e}")
        process.kill()
        return False

if __name__ == "__main__":
    success = test_ctrl_c_cancellation()
    sys.exit(0 if success else 1)