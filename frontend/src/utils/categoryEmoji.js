// 'package' was a legacy sentinel stored before emoji support existed — treat as empty
export const getCategoryEmoji = (_name = '', icon = '') => {
    if (icon && icon.trim() && icon.trim() !== 'package') return icon.trim();
    return '';
};
