import { ChangeEvent } from 'react';
import { handleEvent } from './handleEvent';

const changeEvent = {
  currentTarget: {
    value: 'test',
  },
} as ChangeEvent<HTMLInputElement>;

describe('handleEvent', () => {
  it('returns value from event', () => {
    const handler = handleEvent();
    expect(handler(changeEvent)).toMatchInlineSnapshot(`"test"`);
  });

  it('calls the handler functions left to right', () => {
    const one = (v: string) => `${v}-1`;
    const two = (v: string) => `${v}-2`;
    const three = (v: string) => `${v}-3`;

    const handler = handleEvent(one, two, three);
    expect(handler(changeEvent)).toMatchInlineSnapshot(`"test-1-2-3"`);
  });
});
