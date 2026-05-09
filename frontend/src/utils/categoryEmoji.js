const CATEGORY_EMOJI_MAP = [
    ['produce', '🥬'], ['vegetable', '🥬'], ['fruit', '🍎'],
    ['meat', '🥩'], ['fish', '🐟'], ['seafood', '🦐'],
    ['dairy', '🥛'], ['milk', '🥛'],
    ['bakery', '🍞'], ['bread', '🍞'],
    ['grain', '🌾'], ['cereal', '🌾'], ['pasta', '🍝'],
    ['frozen', '❄️'],
    ['drink', '🥤'], ['beverage', '🥤'], ['juice', '🥤'], ['water', '💧'],
    ['household', '🧴'], ['cleaning', '🧴'], ['hygiene', '🧴'],
    ['snack', '🍿'], ['sweet', '🍬'], ['candy', '🍬'],
    ['egg', '🥚'], ['cheese', '🧀'],
];

export const getCategoryEmoji = (name = '', icon = '') => {
    // 'package' was a legacy sentinel stored before emoji support existed — treat as empty
    if (icon && icon.trim() && icon.trim() !== 'package') return icon.trim();
    const lower = name.toLowerCase();
    for (const [key, emoji] of CATEGORY_EMOJI_MAP) {
        if (lower.includes(key)) return emoji;
    }
    return '📦';
};
