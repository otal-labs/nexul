import { SignedOutHome } from "@/components/home/SignedOutHome";
import { WorkspaceHome } from "@/components/home/WorkspaceHome";
import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";
import { useSessionStore } from "@/stores/sessionStore";

export const HomePage = () => {
  const isLoggedIn = useSessionStore((s) => s.isLoggedIn);
  return (
    <div className="h-full">
      {!isLoggedIn && (
        <ShowcaseSurface>
          <SignedOutHome />
        </ShowcaseSurface>
      )}
      {isLoggedIn && <WorkspaceHome />}
    </div>
  );
};
