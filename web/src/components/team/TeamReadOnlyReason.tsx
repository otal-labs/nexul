export const TeamReadOnlyReason = ({ workspaceName }: { workspaceName: string }) => (
  <p className="pt-2 text-xs text-muted-foreground">Read only: you can&apos;t manage members in {workspaceName}.</p>
);
