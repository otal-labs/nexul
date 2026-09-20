import {
  Box,
  Braces,
  Container,
  Database,
  Server,
  type LucideIcon,
} from "lucide-react";

import { cn } from "@/lib/utils";

const runtimeIcons: Record<string, LucideIcon> = {
  go: Braces,
  node: Box,
  python: Braces,
  rust: Braces,
  postgres: Database,
  mysql: Database,
  redis: Container,
  docker: Container,
  nginx: Server,
};

interface RuntimeIconProps {
  runtime: string;
  className?: string;
}

export const RuntimeIcon = ({ runtime, className }: RuntimeIconProps) => {
  const Icon = runtimeIcons[runtime] ?? Server;
  return <Icon className={cn("size-4 shrink-0", className)} aria-hidden />;
};
