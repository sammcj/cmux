use crate::errors::SandboxResult;
use crate::models::{CreateSandboxRequest, ExecRequest, ExecResponse, SandboxSummary};
use async_trait::async_trait;
use std::sync::Arc;
use uuid::Uuid;

#[async_trait]
pub trait SandboxService: Send + Sync + 'static {
    async fn create(&self, request: CreateSandboxRequest) -> SandboxResult<SandboxSummary>;
    async fn list(&self) -> SandboxResult<Vec<SandboxSummary>>;
    async fn get(&self, id: Uuid) -> SandboxResult<Option<SandboxSummary>>;
    async fn exec(&self, id: Uuid, exec: ExecRequest) -> SandboxResult<ExecResponse>;
    async fn delete(&self, id: Uuid) -> SandboxResult<Option<SandboxSummary>>;
}

#[derive(Clone)]
pub struct AppState {
    pub service: Arc<dyn SandboxService>,
}

impl AppState {
    pub fn new(service: Arc<dyn SandboxService>) -> Self {
        Self { service }
    }
}

#[allow(dead_code)]
fn assert_app_state_bounds() {
    fn assert_state<T: Clone + Send + Sync + 'static>() {}
    assert_state::<AppState>();
}
