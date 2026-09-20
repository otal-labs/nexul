import { useNavigate } from "react-router";

import { Button } from "@/components/ui/button";
import type { DnsSetupResult } from "@/models/DNS";

interface DnsSetupDoneProps {
  result: DnsSetupResult;
}

export const DnsSetupDone = ({ result }: DnsSetupDoneProps) => {
  const navigate = useNavigate();

  return (
    <div className="space-y-5">
      <div>
        <p className="text-sm">{result.headline}</p>
        <p className="mt-1 font-mono text-xs text-muted-foreground">{result.detail}</p>
      </div>
      <Button className="w-full sm:w-auto" onClick={() => navigate("/")}>
        Continue to Nexul
      </Button>
    </div>
  );
};
