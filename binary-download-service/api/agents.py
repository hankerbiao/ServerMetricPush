from fastapi import APIRouter, Depends, HTTPException, Request
from sqlalchemy.orm import Session

from database import AgentRecord, get_db, localnow
from schemas import (
    AgentDetailResponse,
    AgentHeartbeatRequest,
    AgentListResponse,
    AgentRegisterRequest,
    AgentRegisterResponse,
    AgentUpdateCreateRequest,
    AgentUpdateResponse,
    ConfigTemplateCreateRequest,
    ConfigTemplateResponse,
    VersionsResponse,
    BinaryVersionResponse,
)
from services.agent_service import (
    build_agent_detail,
    heartbeat_agent,
    get_agent_or_404,
    register_agent,
    serialize_agent,
)
from services.update_service import (
    create_binary_update,
    create_config_template,
    create_config_update,
    build_agent_update_payload,
    dispatch_agent_update,
    get_agent_update_or_404,
    list_agent_updates,
    list_binary_versions,
    list_config_templates,
)

router = APIRouter()


@router.post("/api/agents/register", response_model=AgentRegisterResponse)
def register_agent_route(payload: AgentRegisterRequest, db: Session = Depends(get_db)):
    return register_agent(db, payload)


@router.post("/api/agents/heartbeat")
def heartbeat_agent_route(payload: AgentHeartbeatRequest, db: Session = Depends(get_db)):
    return heartbeat_agent(db, payload)


@router.get("/api/agents", response_model=AgentListResponse)
def list_agents(db: Session = Depends(get_db)):
    rows = db.query(AgentRecord).order_by(AgentRecord.updated_at.desc()).all()
    return AgentListResponse(agents=[serialize_agent(row) for row in rows])


@router.get("/api/agents/{agent_id}", response_model=AgentDetailResponse)
def get_agent(agent_id: str, db: Session = Depends(get_db)):
    return build_agent_detail(db, agent_id)


@router.post("/api/agents/{agent_id}/updates", response_model=AgentUpdateResponse, status_code=202)
def create_agent_update(agent_id: str, payload: AgentUpdateCreateRequest, request: Request, db: Session = Depends(get_db)):
    agent = get_agent_or_404(db, agent_id)
    update_record = None
    if payload.update_type == "binary_update":
        if not payload.target_version:
            raise HTTPException(status_code=400, detail="target_version is required for binary_update")
        update_record = create_binary_update(db, agent, payload.target_version)
    elif payload.update_type == "config_update":
        if payload.config_template_id is None:
            raise HTTPException(status_code=400, detail="config_template_id is required for config_update")
        update_record = create_config_update(db, agent, payload.config_template_id)
    else:
        raise HTTPException(status_code=400, detail="unsupported update_type")

    dispatch_payload = build_agent_update_payload(db, agent, update_record, str(request.base_url).rstrip("/"))
    try:
        dispatch_agent_update(agent, update_record, dispatch_payload)
    except Exception as exc:
        update_record.status = "failed"
        update_record.detail_message = str(exc)
        update_record.finished_at = localnow()
        agent.update_status = "failed"
        agent.update_error = str(exc)
        agent.update_in_progress = False
        db.commit()
        raise HTTPException(status_code=502, detail=f"更新指令下发失败: {exc}") from exc
    return update_record


@router.get("/api/agents/{agent_id}/updates", response_model=list[AgentUpdateResponse])
def list_updates(agent_id: str, db: Session = Depends(get_db)):
    get_agent_or_404(db, agent_id)
    return list_agent_updates(db, agent_id)


@router.get("/api/agents/{agent_id}/updates/{request_id}", response_model=AgentUpdateResponse)
def get_update(agent_id: str, request_id: str, db: Session = Depends(get_db)):
    get_agent_or_404(db, agent_id)
    return get_agent_update_or_404(db, agent_id, request_id)


@router.get("/api/versions", response_model=VersionsResponse)
def list_versions(db: Session = Depends(get_db)):
    binaries = list_binary_versions(db)
    templates = list_config_templates(db)
    return VersionsResponse(
        binary_versions=[
            BinaryVersionResponse(
                id=item.id,
                filename=item.filename,
                program=item.program,
                version=item.version,
                os=item.os,
                arch=item.arch,
                uploaded_at=item.uploaded_at,
            )
            for item in binaries
        ],
        config_templates=[ConfigTemplateResponse.model_validate(item) for item in templates],
    )


@router.post("/api/config-templates", response_model=ConfigTemplateResponse)
def upsert_config_template(payload: ConfigTemplateCreateRequest, db: Session = Depends(get_db)):
    template = create_config_template(db, payload.name, payload.version, payload.content, payload.notes)
    return ConfigTemplateResponse.model_validate(template)
