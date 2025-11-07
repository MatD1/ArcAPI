/**
 * Multilingual text extraction utilities
 * Supports language codes: en, de, es, fr, it, ja, kr, no, pl, pt, ru, tr, uk, zh-CN, zh-TW, da, hr, sr
 */

export type SupportedLanguage = 
  | 'en' | 'de' | 'es' | 'fr' | 'it' | 'ja' | 'kr' | 'no' | 'pl' | 'pt' | 'ru' | 'tr' | 'uk' 
  | 'zh-CN' | 'zh-TW' | 'da' | 'hr' | 'sr';

/**
 * Get the user's preferred language from browser or localStorage
 * Falls back to 'en' if not available
 */
export function getPreferredLanguage(): SupportedLanguage {
  // Check localStorage first
  const stored = localStorage.getItem('preferred_language');
  if (stored && isValidLanguage(stored)) {
    return stored as SupportedLanguage;
  }

  // Try browser language
  if (typeof window !== 'undefined') {
    const browserLang = navigator.language.split('-')[0]; // Get base language (e.g., 'en' from 'en-US')
    if (isValidLanguage(browserLang)) {
      return browserLang as SupportedLanguage;
    }
  }

  return 'en'; // Default fallback
}

/**
 * Check if a language code is valid
 */
function isValidLanguage(lang: string): boolean {
  const validLanguages: SupportedLanguage[] = [
    'en', 'de', 'es', 'fr', 'it', 'ja', 'kr', 'no', 'pl', 'pt', 'ru', 'tr', 'uk',
    'zh-CN', 'zh-TW', 'da', 'hr', 'sr'
  ];
  return validLanguages.includes(lang as SupportedLanguage);
}

/**
 * Extract text from a multilingual object
 * @param multilingualText - Object with language codes as keys (e.g., { en: "Text", de: "Text" })
 * @param preferredLang - Preferred language code (defaults to user's preferred language)
 * @returns The text in the preferred language, or English, or the first available language, or empty string
 */
export function getMultilingualText(
  multilingualText: Record<string, string> | string | undefined | null,
  preferredLang?: SupportedLanguage
): string {
  if (!multilingualText) {
    return '';
  }

  // If it's already a string, return it (backward compatibility)
  if (typeof multilingualText === 'string') {
    return multilingualText;
  }

  // If it's not an object, return empty string
  if (typeof multilingualText !== 'object') {
    return '';
  }

  const lang = preferredLang || getPreferredLanguage();

  // Try preferred language first
  if (multilingualText[lang]) {
    return multilingualText[lang];
  }

  // Try English as fallback
  if (multilingualText['en']) {
    return multilingualText['en'];
  }

  // Try any available language
  const firstKey = Object.keys(multilingualText)[0];
  if (firstKey && multilingualText[firstKey]) {
    return multilingualText[firstKey];
  }

  return '';
}

/**
 * Extract an array of multilingual texts (e.g., objectives)
 * @param multilingualArray - Array of multilingual objects
 * @param preferredLang - Preferred language code
 * @returns Array of strings in the preferred language
 */
export function getMultilingualArray(
  multilingualArray: Array<Record<string, string> | string> | undefined | null,
  preferredLang?: SupportedLanguage
): string[] {
  if (!multilingualArray || !Array.isArray(multilingualArray)) {
    return [];
  }

  return multilingualArray.map(item => getMultilingualText(item, preferredLang));
}

/**
 * Set the preferred language
 */
export function setPreferredLanguage(lang: SupportedLanguage): void {
  localStorage.setItem('preferred_language', lang);
}

