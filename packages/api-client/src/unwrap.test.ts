import { describe, expect, it } from 'vitest';
import { AxiosError } from 'axios';
import { getErrorMessage } from './errors.ts';
import { unwrapListResponse, unwrapSingleEntity } from './unwrap.ts';

describe('unwrapSingleEntity', () => {
  it('returns data', () => {
    expect(unwrapSingleEntity({ success: true, data: { id: 1 } })).toEqual({ id: 1 });
  });
  it('throws the backend message on failure', () => {
    expect(() =>
      unwrapSingleEntity({ success: false, error: { message: 'nope' } }),
    ).toThrow('nope');
  });
  it('throws when data is missing', () => {
    expect(() => unwrapSingleEntity({ success: true })).toThrow(/empty response/);
  });
  it('accepts falsy-but-present data', () => {
    expect(unwrapSingleEntity({ success: true, data: 0 })).toBe(0);
  });
});

describe('unwrapListResponse', () => {
  it('returns data and meta', () => {
    const meta = { page: 1 };
    expect(unwrapListResponse({ success: true, data: [1, 2], meta })).toEqual({
      data: [1, 2],
      meta,
    });
  });
  it('defaults missing data to []', () => {
    expect(unwrapListResponse({ success: true })).toEqual({ data: [], meta: undefined });
  });
  it('throws on failure', () => {
    expect(() => unwrapListResponse({ success: false, error: { message: 'bad' } })).toThrow('bad');
  });
});

describe('getErrorMessage', () => {
  it('prefers the backend error body for AxiosErrors', () => {
    const err = new AxiosError('Request failed with status code 400');
    err.response = {
      data: { error: { message: 'Email taken' } },
      status: 400,
      statusText: '',
      headers: {},
      config: {} as never,
    };
    expect(getErrorMessage(err)).toBe('Email taken');
  });
  it('falls back to err.message, then the fallback', () => {
    expect(getErrorMessage(new AxiosError('net down'))).toBe('net down');
    expect(getErrorMessage(new Error('boom'))).toBe('boom');
    expect(getErrorMessage('str')).toBe('Something went wrong');
    expect(getErrorMessage(null, 'custom')).toBe('custom');
  });
});
