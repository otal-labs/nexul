export const TeamReadOnlyReason = ({ workspaceName }: { workspaceName: string }) => (
  <p className="text-xs text-muted-foreground">Read only: you can&apos;t manage members in {workspaceName}.</p>
);
