import { GenesisData } from "../lib/types";

export type SetupSection = React.FC<{
  setup: GenesisData;
  update: (s: GenesisData) => unknown;
}>;
