import { useAppStore } from '../stores/appStore'
import { translate, translateError } from '../i18n'
import type { Language } from '../types'

// useT provides translation helpers bound to the current language from settings.
export function useT() {
  const lang = useAppStore((s) => (s.settings.language as Language) || 'vi')
  return {
    lang,
    t: (key: string) => translate(lang, key),
    te: (message: string) => translateError(lang, message),
  }
}
