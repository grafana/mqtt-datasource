import { ChangeEvent } from 'react';
import { handlerFactory } from './handleEvent';

const changeEvent = {
  currentTarget: {
    value: 'test',
  },
} as ChangeEvent<HTMLInputElement>;

describe('handlerFactory', () => {
  it('returns value from event', () => {
    const cb = jest.fn();
    const handler = handlerFactory({ a: { b: 'c' } }, cb);
    handler('a.b')(changeEvent);
    expect(cb.mock.calls[0][0]).toMatchInlineSnapshot(`
      Object {
        "a": Object {
          "b": "test",
        },
      }
    `);
  });

  it('calls the formatting function', () => {
    const cb = jest.fn();
    const handler = handlerFactory({ a: { b: 'c' } }, cb);
    handler('a.b', (v) => v.toUpperCase())(changeEvent);
    expect(cb.mock.calls[0][0]).toMatchInlineSnapshot(`
      Object {
        "a": Object {
          "b": "TEST",
        },
      }
    `);
  });
});
