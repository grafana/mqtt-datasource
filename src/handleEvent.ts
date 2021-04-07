import { ChangeEvent } from 'react';

export type ReturnValue = string | number | boolean | null | void;
export type EventHandler = (value: any) => ReturnValue;

/**
 * calls functions passed as args with `event?.currentTarget.value`
 * passed to the first function, and the return value of the preceeding function
 * for every function after the first.
 */
export const handleEvent = (...fns: EventHandler[]) => (event: ChangeEvent<HTMLInputElement>) =>
  fns.reduce((result: ReturnValue, fn: EventHandler) => fn.call(null, result), event?.currentTarget.value);
