import { create } from "zustand";

/**
 * Global UI state for the command palette and keyboard-shortcuts help dialog.
 * Lets the topbar search button, the ⌘K handler, and the `?` handler all drive
 * the same overlays without prop-drilling.
 */
interface CommandState {
  paletteOpen: boolean;
  helpOpen: boolean;
  openPalette: () => void;
  closePalette: () => void;
  setPaletteOpen: (open: boolean) => void;
  openHelp: () => void;
  closeHelp: () => void;
  setHelpOpen: (open: boolean) => void;
}

export const useCommandStore = create<CommandState>((set) => ({
  paletteOpen: false,
  helpOpen: false,
  openPalette: () => set({ paletteOpen: true, helpOpen: false }),
  closePalette: () => set({ paletteOpen: false }),
  setPaletteOpen: (open) => set({ paletteOpen: open }),
  openHelp: () => set({ helpOpen: true, paletteOpen: false }),
  closeHelp: () => set({ helpOpen: false }),
  setHelpOpen: (open) => set({ helpOpen: open }),
}));
