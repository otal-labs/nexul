export const TeamReadOnlyReason = ({ workspaceName }: { workspaceName: string }) => (
  <p className="text-xs text-muted-foreground">Read only: you need members:write in {workspaceName} to change this.</p>
);
