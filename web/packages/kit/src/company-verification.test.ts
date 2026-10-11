import { describe, expect, it } from 'vitest';
import { errorMessages } from './messages';

describe('company verification messages', () => {
  it('explains why an unverified company cannot start a new commercial operation', () => {
    expect(errorMessages.company_verification_required).toMatch(/тип компании/);
    expect(errorMessages.company_verification_required).toMatch(/провер/);
    expect(errorMessages.invalid_verification_transition).toMatch(/статус/);
  });
});
