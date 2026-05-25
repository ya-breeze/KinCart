import { describe, it, expect } from 'vitest';
import { getCategoryEmoji } from './categoryEmoji';

describe('getCategoryEmoji', () => {
    it('returns explicit icon as-is', () => {
        expect(getCategoryEmoji('Dairy', '🥛')).toBe('🥛');
    });

    it('trims whitespace from explicit icon', () => {
        expect(getCategoryEmoji('Dairy', '  🥛  ')).toBe('🥛');
    });

    it('treats legacy "package" sentinel as empty', () => {
        expect(getCategoryEmoji('Dairy', 'package')).toBe('');
    });

    it('returns empty string when no icon is set', () => {
        expect(getCategoryEmoji('Dairy', '')).toBe('');
        expect(getCategoryEmoji('Dairy')).toBe('');
    });

    it('returns empty string even when name matches a former keyword', () => {
        expect(getCategoryEmoji('dairy')).toBe('');
        expect(getCategoryEmoji('produce')).toBe('');
        expect(getCategoryEmoji('bakery')).toBe('');
    });
});
