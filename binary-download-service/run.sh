#!/bin/bash

APP_NAME="binary-download-service"
PID_FILE="/tmp/${APP_NAME}.pid"
LOG_FILE="/tmp/${APP_NAME}.log"

cd "$(dirname "$0")"

start() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if ps -p "$PID" > /dev/null 2>&1; then
            echo "Service is already running (PID: $PID)"
            return 1
        fi
        rm -f "$PID_FILE"
    fi

    pip install -r requirements.txt -q
    nohup uvicorn main:app --host 0.0.0.0 --port 8080 > "$LOG_FILE" 2>&1 &
    echo $! > "$PID_FILE"
    echo "Service started (PID: $(cat $PID_FILE))"
}

stop() {
    if [ ! -f "$PID_FILE" ]; then
        echo "Service is not running"
        return 1
    fi

    PID=$(cat "$PID_FILE")
    if ps -p "$PID" > /dev/null 2>&1; then
        kill "$PID"
        sleep 1
        if ps -p "$PID" > /dev/null 2>&1; then
            kill -9 "$PID"
        fi
        rm -f "$PID_FILE"
        echo "Service stopped"
    else
        echo "Service is not running (stale PID file)"
        rm -f "$PID_FILE"
    fi
}

status() {
    if [ ! -f "$PID_FILE" ]; then
        echo "Service is not running"
        return 1
    fi

    PID=$(cat "$PID_FILE")
    if ps -p "$PID" > /dev/null 2>&1; then
        echo "Service is running (PID: $PID)"
    else
        echo "Service is not running (stale PID file)"
        rm -f "$PID_FILE"
    fi
}

case "$1" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        stop
        sleep 1
        start
        ;;
    status)
        status
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status}"
        exit 1
        ;;
esac