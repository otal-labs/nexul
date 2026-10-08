import type { Play } from "@/models/Play";

interface PlayMenuRowProps {
  play: Play;
  disabled: boolean;
  onChoose: (play: Play) => void;
}

// One row in the Plays menu: label and description, disabled together with the reason shown above the list.
export const PlayMenuRow = ({ play, disabled, onChoose }: PlayMenuRowProps) => (
  <button
    type="button"
    disabled={disabled}
    onClick={() => onChoose(play)}
    className="flex w-full flex-col items-start rounded-md px-2 py-1.5 text-left transition-colors duration-150 ease-standard hover:bg-accent disabled:opacity-50"
  >
    <span className="text-sm font-medium">{play.label}</span>
    {play.description !== "" && <span className="font-mono text-[11px] text-muted-foreground">{play.description}</span>}
  </button>
);
