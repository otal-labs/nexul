import { KeyboardSensor, MouseSensor, TouchSensor, useSensor, useSensors } from "@dnd-kit/core";
import { sortableKeyboardCoordinates } from "@dnd-kit/sortable";

const MOUSE = { activationConstraint: { distance: 2 } };
const TOUCH = { activationConstraint: { delay: 200, tolerance: 8 } };
const KEYS = { coordinateGetter: sortableKeyboardCoordinates };

// A mouse drag, a held touch, and the keyboard: the ways every ranked list lets a row be moved.
export const useRowSortSensors = () => useSensors(useSensor(MouseSensor, MOUSE), useSensor(TouchSensor, TOUCH), useSensor(KeyboardSensor, KEYS));
