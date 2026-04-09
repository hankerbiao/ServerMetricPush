import os
from datetime import datetime
from typing import Optional
from zoneinfo import ZoneInfo

from sqlalchemy import Boolean, Text, create_engine, inspect
from sqlalchemy.orm import sessionmaker, declarative_base
from sqlalchemy import Column, Integer, String, DateTime

SHANGHAI_TZ = ZoneInfo("Asia/Shanghai")


def localnow() -> datetime:
    return datetime.now(SHANGHAI_TZ).replace(tzinfo=None)

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
DATABASE_URL = f"sqlite:///{os.path.join(BASE_DIR, 'files.db')}"

engine = create_engine(DATABASE_URL, connect_args={"check_same_thread": False})
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)
Base = declarative_base()


class FileRecord(Base):
    __tablename__ = "files"

    id = Column(Integer, primary_key=True, index=True)
    filename = Column(String, unique=True, nullable=False)
    program = Column(String, nullable=False)
    version = Column(String, nullable=False)
    os = Column(String, nullable=False)
    arch = Column(String, nullable=False)
    file_path = Column(String, nullable=False)
    file_size = Column(Integer, nullable=False)
    uploaded_at = Column(DateTime, default=localnow)


class AgentRecord(Base):
    __tablename__ = "agents"

    id = Column(Integer, primary_key=True, index=True)
    agent_id = Column(String, unique=True, nullable=False, index=True)
    hostname = Column(String, nullable=False)
    version = Column(String, nullable=False)
    os = Column(String, nullable=False)
    arch = Column(String, nullable=False)
    ip = Column(String, nullable=True)
    status = Column(String, nullable=False, default="online")
    last_error = Column(Text, nullable=True)
    pushgateway_url = Column(String, nullable=False)
    push_interval_seconds = Column(Integer, nullable=False)
    node_exporter_port = Column(Integer, nullable=False)
    node_exporter_metrics_url = Column(String, nullable=False)
    node_exporter_up = Column(Boolean, nullable=False, default=True)
    push_fail_count = Column(Integer, nullable=False, default=0)
    update_listen_addr = Column(String, nullable=True)
    update_status = Column(String, nullable=False, default="idle")
    update_target = Column(String, nullable=True)
    update_error = Column(Text, nullable=True)
    update_in_progress = Column(Boolean, nullable=False, default=False)
    current_config_version = Column(String, nullable=True)
    started_at = Column(DateTime, nullable=True)
    last_seen_at = Column(DateTime, nullable=True)
    last_push_at = Column(DateTime, nullable=True)
    last_push_success_at = Column(DateTime, nullable=True)
    last_push_error_at = Column(DateTime, nullable=True)
    registered_at = Column(DateTime, default=localnow)
    updated_at = Column(DateTime, default=localnow, onupdate=localnow)


class AgentEventRecord(Base):
    __tablename__ = "agent_events"

    id = Column(Integer, primary_key=True, index=True)
    agent_id = Column(String, nullable=False, index=True)
    event_type = Column(String, nullable=False)
    message = Column(Text, nullable=False)
    created_at = Column(DateTime, default=localnow)


class ConfigTemplateRecord(Base):
    __tablename__ = "config_templates"

    id = Column(Integer, primary_key=True, index=True)
    name = Column(String, unique=True, nullable=False, index=True)
    version = Column(String, nullable=False)
    content = Column(Text, nullable=False)
    notes = Column(Text, nullable=True)
    created_at = Column(DateTime, default=localnow)
    updated_at = Column(DateTime, default=localnow, onupdate=localnow)


class AgentUpdateRecord(Base):
    __tablename__ = "agent_updates"

    id = Column(Integer, primary_key=True, index=True)
    request_id = Column(String, unique=True, nullable=False, index=True)
    agent_id = Column(String, nullable=False, index=True)
    update_type = Column(String, nullable=False)
    status = Column(String, nullable=False, default="accepted")
    from_binary_version = Column(String, nullable=True)
    to_binary_version = Column(String, nullable=True)
    from_config_version = Column(String, nullable=True)
    to_config_version = Column(String, nullable=True)
    config_template_id = Column(Integer, nullable=True)
    config_template_name = Column(String, nullable=True)
    detail_message = Column(Text, nullable=True)
    rollback_performed = Column(Boolean, nullable=False, default=False)
    created_at = Column(DateTime, default=localnow)
    accepted_at = Column(DateTime, nullable=True)
    finished_at = Column(DateTime, nullable=True)
    updated_at = Column(DateTime, default=localnow, onupdate=localnow)

    @property
    def target_version(self) -> Optional[str]:
        return self.to_binary_version or self.to_config_version


def ensure_column(table_name: str, column_name: str, definition: str) -> None:
    inspector = inspect(engine)
    existing = {column["name"] for column in inspector.get_columns(table_name)}
    if column_name in existing:
        return

    with engine.begin() as connection:
        connection.exec_driver_sql(f"ALTER TABLE {table_name} ADD COLUMN {column_name} {definition}")


Base.metadata.create_all(bind=engine)
ensure_column("agents", "update_listen_addr", "VARCHAR")
ensure_column("agents", "update_status", "VARCHAR NOT NULL DEFAULT 'idle'")
ensure_column("agents", "update_target", "VARCHAR")
ensure_column("agents", "update_error", "TEXT")
ensure_column("agents", "update_in_progress", "BOOLEAN NOT NULL DEFAULT 0")
ensure_column("agents", "current_config_version", "VARCHAR")


def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()
