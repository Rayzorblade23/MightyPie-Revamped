// Shared types for pie button components
import type {ButtonType} from './pieButtonTypes.ts';

/**
 * Union type for all possible button properties
 */
export type ButtonPropertiesUnion =
    | import('./pieButtonTypes.ts').ShowProgramWindowProperties
    | import('./pieButtonTypes.ts').ShowAnyWindowProperties
    | import('./pieButtonTypes.ts').CallFunctionProperties
    | import('./pieButtonTypes.ts').LaunchProgramProperties
    | import('./pieButtonTypes.ts').OpenSpecificPieMenuPageProperties;

/**
 * Base props shared by all pie button components
 */
export interface PieButtonBaseProps {
    // Layout props
    width: number;
    height: number;

    // Display props
    taskType: ButtonType | 'empty';
    properties: ButtonPropertiesUnion | undefined;
    buttonTextUpper?: string;
    buttonTextLower?: string;

    // Styling props
    allowSelectWhenDisabled?: boolean;
}

/**
 * Mouse state interface for tracking all mouse interactions
 */
export interface MouseState {
    hovered: boolean;
    leftDown: boolean;
    leftUp: boolean;
    rightDown: boolean;
    rightUp: boolean;
    middleDown: boolean;
    middleUp: boolean;
}
