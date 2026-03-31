from datetime import datetime, timedelta, timezone
from typing import Optional

from fastapi import HTTPException
from sqlalchemy.orm import Session

from app_settings import (
    HEARTBEAT_INTERVAL_SECONDS,
    OFFLINE_TIMEOUT_SECONDS,
    RECENT_EVENTS_LIMIT,
)
from database import AgentEventRecord, AgentRecord, localnow
from schemas import (
    AgentDetailResponse,
    AgentEventResponse,
    AgentHeartbeatRequest,
    AgentRegisterRequest,
    AgentRegisterResponse,
    AgentResponse,
)


def utcnow() -> datetime:
    return localnow()


def normalize_datetime(value: Optional[datetime]) -> Optional[datetime]:
    if value is None:
        return None
    if value.tzinfo is not None:
        return value.astimezone(timezone(timedelta(hours=8))).replace(tzinfo=None)
    return value


def add_agent_event(db: Session, agent_id: str, event_type: str, message: str) -> None:
    db.add(
        AgentEventRecord(
            agent_id=agent_id,
            event_type=event_type,
            message=message,
        )
    )


def serialize_agent(agent: AgentRecord) -> AgentResponse:
    now = utcnow()
    last_seen = agent.last_seen_at
    online = False
    effective_status = "offline"
    if last_seen is not None and now - last_seen <= timedelta(seconds=OFFLINE_TIMEOUT_SECONDS):
        online = True
        degraded = (
            agent.status == "degraded"
            or not agent.node_exporter_up
            or agent.push_fail_count > 0
            or bool(agent.last_error)
        )
        effective_status = "degraded" if degraded else "online"

    return AgentResponse(
        agent_id=agent.agent_id,
        hostname=agent.hostname,
        version=agent.version,
        os=agent.os,
        arch=agent.arch,
        ip=agent.ip,
        status=effective_status,
        online=online,
        last_error=agent.last_error,
        pushgateway_url=agent.pushgateway_url,
        push_interval_seconds=agent.push_interval_seconds,
        node_exporter_port=agent.node_exporter_port,
        node_exporter_metrics_url=agent.node_exporter_metrics_url,
        node_exporter_up=agent.node_exporter_up,
        push_fail_count=agent.push_fail_count,
        started_at=agent.started_at,
        last_seen_at=agent.last_seen_at,
        last_push_at=agent.last_push_at,
        last_push_success_at=agent.last_push_success_at,
        last_push_error_at=agent.last_push_error_at,
        registered_at=agent.registered_at,
        updated_at=agent.updated_at,
    )


def register_agent(db: Session, payload: AgentRegisterRequest) -> AgentRegisterResponse:
    record = db.query(AgentRecord).filter(AgentRecord.agent_id == payload.agent_id).first()
    is_new = record is None
    if record is None:
        record = AgentRecord(agent_id=payload.agent_id)
        db.add(record)

    record.hostname = payload.hostname
    record.version = payload.version
    record.os = payload.os
    record.arch = payload.arch
    record.ip = payload.ip
    record.status = "online"
    record.last_error = None
    record.pushgateway_url = payload.pushgateway_url
    record.push_interval_seconds = payload.push_interval_seconds
    record.node_exporter_port = payload.node_exporter_port
    record.node_exporter_metrics_url = payload.node_exporter_metrics_url
    record.node_exporter_up = True
    record.push_fail_count = 0
    record.started_at = normalize_datetime(payload.started_at)
    record.last_seen_at = utcnow()

    add_agent_event(
        db,
        payload.agent_id,
        "registered" if is_new else "reregistered",
        f"{payload.hostname} 已完成注册",
    )
    db.commit()

    return AgentRegisterResponse(
        heartbeat_interval_seconds=HEARTBEAT_INTERVAL_SECONDS,
        offline_timeout_seconds=OFFLINE_TIMEOUT_SECONDS,
    )


def heartbeat_agent(db: Session, payload: AgentHeartbeatRequest) -> dict:
    record = db.query(AgentRecord).filter(AgentRecord.agent_id == payload.agent_id).first()
    if record is None:
        raise HTTPException(status_code=404, detail="节点未注册")

    previous_status = record.status
    previous_error = record.last_error

    record.status = payload.status
    record.last_error = payload.last_error
    record.last_push_at = normalize_datetime(payload.last_push_at)
    record.last_push_success_at = normalize_datetime(payload.last_push_success_at)
    record.last_push_error_at = normalize_datetime(payload.last_push_error_at)
    record.push_fail_count = payload.push_fail_count
    record.node_exporter_up = payload.node_exporter_up
    record.last_seen_at = utcnow()

    if previous_status != payload.status:
        add_agent_event(
            db,
            payload.agent_id,
            "status_changed",
            f"状态从 {previous_status} 变更为 {payload.status}",
        )
    elif payload.last_error and payload.last_error != previous_error:
        add_agent_event(
            db,
            payload.agent_id,
            "error",
            payload.last_error,
        )

    db.commit()
    return {"message": "心跳更新成功"}


def get_agent_or_404(db: Session, agent_id: str) -> AgentRecord:
    record = db.query(AgentRecord).filter(AgentRecord.agent_id == agent_id).first()
    if record is None:
        raise HTTPException(status_code=404, detail="节点不存在")
    return record


def build_agent_detail(db: Session, agent_id: str) -> AgentDetailResponse:
    record = get_agent_or_404(db, agent_id)
    events = (
        db.query(AgentEventRecord)
        .filter(AgentEventRecord.agent_id == agent_id)
        .order_by(AgentEventRecord.created_at.desc())
        .limit(RECENT_EVENTS_LIMIT)
        .all()
    )
    return AgentDetailResponse(
        agent=serialize_agent(record),
        events=[AgentEventResponse.model_validate(event) for event in events],
    )
