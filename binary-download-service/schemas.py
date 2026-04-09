from pydantic import BaseModel
from datetime import datetime
from typing import Optional


class FileResponse(BaseModel):
    id: int
    filename: str
    program: str
    version: str
    os: str
    arch: str
    file_size: int
    uploaded_at: datetime

    class Config:
        from_attributes = True


class FileListResponse(BaseModel):
    files: list[FileResponse]


class AgentRegisterRequest(BaseModel):
    agent_id: str
    hostname: str
    version: str
    os: str
    arch: str
    ip: Optional[str] = None
    pushgateway_url: str
    push_interval_seconds: int
    node_exporter_port: int
    node_exporter_metrics_url: str
    update_listen_addr: Optional[str] = None
    current_config_version: Optional[str] = None
    started_at: datetime


class AgentHeartbeatRequest(BaseModel):
    agent_id: str
    status: str
    last_error: Optional[str] = None
    last_push_at: Optional[datetime] = None
    last_push_success_at: Optional[datetime] = None
    last_push_error_at: Optional[datetime] = None
    push_fail_count: int = 0
    node_exporter_up: bool = True
    update_in_progress: bool = False
    last_update_request_id: Optional[str] = None
    last_update_type: Optional[str] = None
    last_update_status: Optional[str] = None
    last_update_target: Optional[str] = None
    last_update_error: Optional[str] = None
    current_config_version: Optional[str] = None


class AgentEventResponse(BaseModel):
    event_type: str
    message: str
    created_at: datetime

    class Config:
        from_attributes = True


class AgentResponse(BaseModel):
    agent_id: str
    hostname: str
    version: str
    os: str
    arch: str
    ip: Optional[str] = None
    status: str
    online: bool
    last_error: Optional[str] = None
    pushgateway_url: str
    push_interval_seconds: int
    node_exporter_port: int
    node_exporter_metrics_url: str
    node_exporter_up: bool
    push_fail_count: int
    update_listen_addr: Optional[str] = None
    update_status: str
    update_target: Optional[str] = None
    update_error: Optional[str] = None
    update_in_progress: bool
    current_config_version: Optional[str] = None
    started_at: Optional[datetime] = None
    last_seen_at: Optional[datetime] = None
    last_push_at: Optional[datetime] = None
    last_push_success_at: Optional[datetime] = None
    last_push_error_at: Optional[datetime] = None
    registered_at: datetime
    updated_at: datetime


class AgentListResponse(BaseModel):
    agents: list[AgentResponse]


class AgentDetailResponse(BaseModel):
    agent: AgentResponse
    events: list[AgentEventResponse]
    updates: list["AgentUpdateResponse"]


class AgentRegisterResponse(BaseModel):
    heartbeat_interval_seconds: int
    offline_timeout_seconds: int


class ConfigTemplateResponse(BaseModel):
    id: int
    name: str
    version: str
    notes: Optional[str] = None
    created_at: datetime

    class Config:
        from_attributes = True


class ConfigTemplateCreateRequest(BaseModel):
    name: str
    version: str
    content: str
    notes: Optional[str] = None


class BinaryVersionResponse(BaseModel):
    id: int
    filename: str
    program: str
    version: str
    os: str
    arch: str
    uploaded_at: datetime


class VersionsResponse(BaseModel):
    binary_versions: list[BinaryVersionResponse]
    config_templates: list[ConfigTemplateResponse]


class AgentUpdateCreateRequest(BaseModel):
    update_type: str
    target_version: Optional[str] = None
    config_template_id: Optional[int] = None
    force: bool = False


class AgentUpdateResponse(BaseModel):
    request_id: str
    update_type: str
    status: str
    target_version: Optional[str] = None
    from_binary_version: Optional[str] = None
    to_binary_version: Optional[str] = None
    from_config_version: Optional[str] = None
    to_config_version: Optional[str] = None
    config_template_id: Optional[int] = None
    config_template_name: Optional[str] = None
    detail_message: Optional[str] = None
    rollback_performed: bool
    created_at: datetime
    accepted_at: Optional[datetime] = None
    finished_at: Optional[datetime] = None

    class Config:
        from_attributes = True


AgentDetailResponse.model_rebuild()
