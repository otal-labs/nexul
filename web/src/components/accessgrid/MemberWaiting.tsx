import { FolderLock } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { Container } from "@/components/Container";
import { EmptyState } from "@/components/EmptyState";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

// What a member no project lists meets until an Owner finishes the setup: no project anywhere, and one plain reason.
export const MemberWaiting = () => {
  const wsPath = useWorkspacePath();
  return (
    <Container className="flex min-h-full items-center justify-center py-16">
      <EmptyState
        role="status"
        icon={FolderLock}
        title="Your projects are on their way"
        message="This update rebuilt permissions, and an Owner is now choosing who's in each project. Yours will show up here once they're done. Chat works as usual."
        action={
          <Button asChild variant="outline">
            <Link to={wsPath("/chat")}>Open chat</Link>
          </Button>
        }
      />
    </Container>
  );
};
