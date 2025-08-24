// piemenuConfigTypes.ts

import type {MenuConfigData} from "$lib/data/types/pieButtonTypes.ts";

export interface ShortcutEntry {
    codes: number[];
    label: string;
}

export type ShortcutsMap = Record<string, ShortcutEntry>;

export interface StarredFavorite {
    menuID: number;
    pageID: number;
}

export interface PieMenuConfig {
    buttons: MenuConfigData; // existing nested record structure
    shortcuts: ShortcutsMap; // keys stored as strings in file
    starred: StarredFavorite | null; // null if unset
}
