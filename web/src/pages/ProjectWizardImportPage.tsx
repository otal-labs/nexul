import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";
import { ImportWizard } from "@/components/wizard/ImportWizard";

// Door 3: adopt what already runs on a machine, in the wizards' frame.
export const ProjectWizardImportPage = () => (
  <ShowcaseSurface quiet>
    <ImportWizard />
  </ShowcaseSurface>
);
