import vi from './vi.json'
import en from './en.json'
import type { Language } from '../types'

type Dict = Record<string, string>

const resources: Record<Language, Dict> = {
  vi: vi as Dict,
  en: en as Dict,
}

export const languages: Language[] = ['vi', 'en']

// translate looks up a key for a language, falling back to the key itself so a
// missing translation is visible rather than blank.
export function translate(lang: Language, key: string): string {
  const dict = resources[lang] ?? resources.vi
  return dict[key] ?? key
}

// translateError maps a backend error message to a localized string. Backend
// sentinel errors use stable dotted keys (e.g. "runner.noPages"); wrapped
// errors may contain a key as a substring. Unknown messages fall back to a
// generic error so raw/technical text is never shown to the user.
export function translateError(lang: Language, message: string): string {
  if (!message) return translate(lang, 'error.generic')

  const knownKeys = [
    'validation.threads',
    'validation.delay',
    'validation.maxErrors',
    'validation.pagesPerThread',
    'validation.accountPerLink',
    'validation.image',
    'runner.noPages',
    'runner.noAccounts',
    'runner.alreadyRunning',
    'app.notReady',
  ]
  for (const key of knownKeys) {
    if (message === key || message.includes(key)) {
      return translate(lang, `error.${key}`)
    }
  }
  return translate(lang, 'error.generic')
}
