import { Box, Cloud, Cpu, Database, Globe, Layers, Package, Rocket, Server, Shield, Terminal, Zap, type LucideIcon } from "lucide-react";

import { cn } from "@/lib/utils";
import { PROJECT_ICON_NAMES, type ProjectIconName } from "@/models/Project";

const projectIcons: Record<ProjectIconName, LucideIcon> = {
  Box,
  Rocket,
  Server,
  Globe,
  Database,
  Layers,
  Terminal,
  Shield,
  Zap,
  Package,
  Cpu,
  Cloud,
};

export const isProjectIconName = (icon: string): icon is ProjectIconName =>
  (PROJECT_ICON_NAMES as readonly string[]).includes(icon);

interface ProjectIconProps {
  icon?: string;
  /** Fallback when icon is unset: renders the prefix's first letter instead of the generic Box glyph. */
  prefix?: string;
  className?: string;
}

// Standalone (not settings-specific) so any surface can reuse the icon without importing the settings picker.
export const ProjectIcon = ({ icon, prefix, className }: ProjectIconProps) => {
  if (icon && isProjectIconName(icon)) {
    const Icon = projectIcons[icon];
    return <Icon className={className} aria-hidden />;
  }
  if (prefix) {
    return (
      <span
        className={cn("inline-flex items-center justify-center font-mono font-semibold", className)}
        aria-hidden
      >
        {prefix.charAt(0).toUpperCase()}
      </span>
    );
  }
  return <Box className={className} aria-hidden />;
};
