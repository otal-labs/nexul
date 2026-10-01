import { useEffect, useState } from "react";

import { createFileStage } from "@/components/doc/image/fileStage";

// One stage per open create dialog; whatever is still staged when it closes has its object URLs released.
export const useFileStage = () => {
  const [stage] = useState(createFileStage);
  useEffect(() => () => stage.clear(), [stage]);
  return stage;
};
