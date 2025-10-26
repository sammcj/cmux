const LOCAL_WORKSPACE_PREFIX = "Local workspace ";
const LOCAL_WORKSPACE_AGENT = "local-workspace";

const matchesLocalWorkspaceDescriptor = (value?: string | null): boolean =>
  typeof value === "string" && value.startsWith(LOCAL_WORKSPACE_PREFIX);

export const isLocalWorkspaceTask = (task: {
  isLocalWorkspace?: boolean | null;
  text?: string | null;
  description?: string | null;
  pullRequestTitle?: string | null;
  pullRequestDescription?: string | null;
}): boolean => {
  if (task.isLocalWorkspace === true) {
    return true;
  }

  return (
    matchesLocalWorkspaceDescriptor(task.text) ||
    matchesLocalWorkspaceDescriptor(task.description) ||
    matchesLocalWorkspaceDescriptor(task.pullRequestTitle) ||
    matchesLocalWorkspaceDescriptor(task.pullRequestDescription)
  );
};

export const isLocalWorkspaceRun = (run: {
  isLocalWorkspace?: boolean | null;
  agentName?: string | null;
}): boolean => {
  if (run.isLocalWorkspace === true) {
    return true;
  }
  return run.agentName === LOCAL_WORKSPACE_AGENT;
};
