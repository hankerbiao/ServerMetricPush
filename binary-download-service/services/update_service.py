import uuid
from typing import Optional
from urllib.parse import urlsplit, urlunsplit

import httpx
from fastapi import HTTPException
from sqlalchemy.orm import Session

from database import AgentRecord, AgentUpdateRecord, ConfigTemplateRecord, FileRecord, localnow


def get_binary_artifact_or_404(db: Session, agent: AgentRecord, target_version: str) -> FileRecord:
    artifact = (
        db.query(FileRecord)
        .filter(
            FileRecord.program == "node-push-exporter",
            FileRecord.version == target_version,
            FileRecord.os == agent.os,
            FileRecord.arch == agent.arch,
        )
        .order_by(FileRecord.uploaded_at.desc(), FileRecord.id.desc())
        .first()
    )
    if artifact is None:
        raise HTTPException(status_code=404, detail="未找到匹配平台的目标版本")
    return artifact


def get_config_template_or_404(db: Session, template_id: int) -> ConfigTemplateRecord:
    template = db.query(ConfigTemplateRecord).filter(ConfigTemplateRecord.id == template_id).first()
    if template is None:
        raise HTTPException(status_code=404, detail="配置模板不存在")
    return template


def create_binary_update(
    db: Session,
    agent: AgentRecord,
    target_version: str,
) -> AgentUpdateRecord:
    artifact = get_binary_artifact_or_404(db, agent, target_version)
    update = AgentUpdateRecord(
        request_id=uuid.uuid4().hex,
        agent_id=agent.agent_id,
        update_type="binary_update",
        status="accepted",
        from_binary_version=agent.version,
        to_binary_version=artifact.version,
        accepted_at=localnow(),
    )
    db.add(update)
    agent.update_status = "accepted"
    agent.update_target = artifact.version
    agent.update_error = None
    agent.update_in_progress = True
    db.commit()
    db.refresh(update)
    return update


def create_config_update(
    db: Session,
    agent: AgentRecord,
    template_id: int,
) -> AgentUpdateRecord:
    template = get_config_template_or_404(db, template_id)
    update = AgentUpdateRecord(
        request_id=uuid.uuid4().hex,
        agent_id=agent.agent_id,
        update_type="config_update",
        status="accepted",
        from_config_version=agent.current_config_version,
        to_config_version=template.version,
        config_template_id=template.id,
        config_template_name=template.name,
        accepted_at=localnow(),
    )
    db.add(update)
    agent.update_status = "accepted"
    agent.update_target = template.version
    agent.update_error = None
    agent.update_in_progress = True
    db.commit()
    db.refresh(update)
    return update


def list_agent_updates(db: Session, agent_id: str) -> list[AgentUpdateRecord]:
    return (
        db.query(AgentUpdateRecord)
        .filter(AgentUpdateRecord.agent_id == agent_id)
        .order_by(AgentUpdateRecord.created_at.desc(), AgentUpdateRecord.id.desc())
        .all()
    )


def get_agent_update_or_404(db: Session, agent_id: str, request_id: str) -> AgentUpdateRecord:
    update = (
        db.query(AgentUpdateRecord)
        .filter(
            AgentUpdateRecord.agent_id == agent_id,
            AgentUpdateRecord.request_id == request_id,
        )
        .first()
    )
    if update is None:
        raise HTTPException(status_code=404, detail="更新记录不存在")
    return update


def list_binary_versions(db: Session) -> list[FileRecord]:
    return (
        db.query(FileRecord)
        .filter(FileRecord.program == "node-push-exporter")
        .order_by(FileRecord.uploaded_at.desc(), FileRecord.id.desc())
        .all()
    )


def list_config_templates(db: Session) -> list[ConfigTemplateRecord]:
    return (
        db.query(ConfigTemplateRecord)
        .order_by(ConfigTemplateRecord.updated_at.desc(), ConfigTemplateRecord.id.desc())
        .all()
    )


def create_config_template(
    db: Session,
    name: str,
    version: str,
    content: str,
    notes: Optional[str],
) -> ConfigTemplateRecord:
    template = db.query(ConfigTemplateRecord).filter(ConfigTemplateRecord.name == name).first()
    if template is None:
        template = ConfigTemplateRecord(name=name)
        db.add(template)

    template.version = version
    template.content = content
    template.notes = notes
    db.commit()
    db.refresh(template)
    return template


def build_agent_update_payload(db: Session, agent: AgentRecord, update: AgentUpdateRecord, service_base_url: str) -> dict:
    payload = {
        "request_id": update.request_id,
        "update_type": update.update_type,
        "force": False,
    }

    if update.update_type == "binary_update":
        artifact = get_binary_artifact_or_404(db, agent, update.to_binary_version)
        payload.update(
            {
                "target_version": update.to_binary_version,
                "download_url": build_download_url(service_base_url, artifact.filename),
                "file_name": artifact.filename,
                "package_type": "tar.gz" if artifact.filename.endswith(".tar.gz") else "binary",
            }
        )
    elif update.update_type == "config_update":
        template = get_config_template_or_404(db, update.config_template_id)
        payload.update(
            {
                "config_template_id": str(template.id),
                "config_content": template.content,
                "config_version": template.version,
            }
        )

    return payload


def dispatch_agent_update(agent: AgentRecord, update: AgentUpdateRecord, payload: dict) -> dict:
    base_url = agent_update_base_url(agent)
    response = httpx.post(f"{base_url}/internal/update", json=payload, timeout=10.0)
    response.raise_for_status()
    body = response.json()
    return body if isinstance(body, dict) else {"status": "accepted"}


def agent_update_base_url(agent: AgentRecord) -> str:
    if agent.update_listen_addr:
        listen_addr = agent.update_listen_addr.strip()
        if listen_addr.startswith("http://") or listen_addr.startswith("https://"):
            return rewrite_loopback_url(listen_addr, agent.ip).rstrip("/")

        host, sep, port = listen_addr.rpartition(":")
        if sep and port and agent.ip and host in {"", "0.0.0.0", "127.0.0.1", "localhost"}:
            return f"http://{agent.ip}:{port}"
        return f"http://{listen_addr}"
    if agent.ip:
        return f"http://{agent.ip}:18080"
    raise HTTPException(status_code=400, detail="节点未上报更新监听地址")


def build_download_url(service_base_url: str, filename: str) -> str:
    return service_base_url.rstrip("/") + f"/download/{filename}"


def rewrite_loopback_url(url: str, ip: Optional[str]) -> str:
    if not ip:
        return url

    parsed = urlsplit(url)
    host = parsed.hostname
    if host not in {"127.0.0.1", "localhost", "0.0.0.0"}:
        return url

    netloc = ip
    if parsed.port is not None:
        netloc = f"{netloc}:{parsed.port}"
    return urlunsplit((parsed.scheme, netloc, parsed.path, parsed.query, parsed.fragment))
