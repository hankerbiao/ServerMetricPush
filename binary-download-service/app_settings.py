import os

INSTALL_SCRIPT_NAME = "install.sh"
UNINSTALL_SCRIPT_NAME = "uninstall.sh"
INSTALL_SCRIPT_PROGRAM = "install-script"
HEARTBEAT_INTERVAL_SECONDS = 30
OFFLINE_TIMEOUT_SECONDS = 90
RECENT_EVENTS_LIMIT = 20

BASE_DIR = os.path.dirname(__file__)
UPLOAD_DIR = os.path.join(BASE_DIR, "uploads")
STATIC_DIR = os.path.join(BASE_DIR, "static")

os.makedirs(UPLOAD_DIR, exist_ok=True)
