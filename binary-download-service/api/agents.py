from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session

from database import AgentRecord, get_db
from schemas import (
    AgentDetailResponse,
    AgentHeartbeatRequest,
    AgentListResponse,
    AgentRegisterRequest,
    AgentRegisterResponse,
)
from services.agent_service import (
    build_agent_detail,
    heartbeat_agent,
    register_agent,
    serialize_agent,
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
