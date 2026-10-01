import type { ReactNode } from "react";
import { FolderLockIcon } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { Container } from "@/components/Container";
import { EmptyState } from "@/components/EmptyState";
import { useRevokedProject } from "@/hooks/useRevokedProject";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

interface ProjectRevokedGateProps {
  children: ReactNode;
}

// The body of a page whose project was taken away while it was open; the sidebar around it stays.
export const ProjectRevokedGate = ({ children }: ProjectRevokedGateProps) => {
  const revoked = useRevokedProject();
  const wsPath = useWorkspacePath();
  return (
    <>
      {revoked && (
        <Container className="p-6">
          <EmptyState
            role="status"
            icon={FolderLockIcon}
            title="You no longer have access to this project"
            message="Someone changed your access. Projects you can still open are in the sidebar."
            action={
              <Button asChild>
                <Link to={wsPath("/")}>Go to Home</Link>
              </Button>
            }
          />
        </Container>
      )}
      {!revoked && children}
    </>
  );
};
