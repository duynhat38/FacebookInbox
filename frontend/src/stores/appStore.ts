import { create } from 'zustand'
import * as App from '../../wailsjs/go/app/App'
import type { Language, Settings } from '../types'
import { debounce } from '../utils/debounce'

const DEFAULT_SETTINGS: Settings = {
  threads: 5,
  delaySeconds: 3,
  maxErrors: 5,
  pagesPerThread: 10,
  accountPerLink: 1,
  imagePath: '',
  language: 'vi',
  deleteThreadAfterSend: false,
  deleteMessageAfterSend: false,
}

interface AppState {
  settings: Settings
  pagesText: string
  proxiesText: string
  contentText: string
  shareContentText: string
  linksText: string
  ready: boolean
  error: string | null

  init: () => Promise<void>
  updateSettings: (patch: Partial<Settings>) => Promise<void>
  setLanguage: (lang: Language) => Promise<void>
  setPagesText: (text: string) => void
  applyPagesFromBackend: (pages: string[]) => void
  setProxiesText: (text: string) => void
  setContentText: (text: string) => void
  setShareContentText: (text: string) => void
  setLinksText: (text: string) => void
  pickImage: () => Promise<void>
  clearImage: () => Promise<void>
  setError: (message: string | null) => void
}

// Debounced auto-savers: persist a few hundred ms after the user stops typing,
// so there is no Save button and no write on every keystroke.
const savePages = debounce((text: string) => {
  void App.SavePages(text).catch((e: unknown) =>
    useAppStore.getState().setError(String(e)),
  )
}, 500)

const saveProxies = debounce((text: string) => {
  void App.SaveProxies(text).catch((e: unknown) =>
    useAppStore.getState().setError(String(e)),
  )
}, 500)

const saveContent = debounce((text: string) => {
  void App.SaveContent(text).catch((e: unknown) =>
    useAppStore.getState().setError(String(e)),
  )
}, 500)

const saveShareContent = debounce((text: string) => {
  void App.SaveShareContent(text).catch((e: unknown) =>
    useAppStore.getState().setError(String(e)),
  )
}, 500)

const saveLinks = debounce((text: string) => {
  void App.SaveLinks(text).catch((e: unknown) =>
    useAppStore.getState().setError(String(e)),
  )
}, 500)

export const useAppStore = create<AppState>((set, get) => ({
  settings: DEFAULT_SETTINGS,
  pagesText: '',
  proxiesText: '',
  contentText: '',
  shareContentText: '',
  linksText: '',
  ready: false,
  error: null,

  init: async () => {
    try {
      const [settings, pages, proxies, content, shareContent, links] = await Promise.all([
        App.GetSettings(),
        App.GetPages(),
        App.GetProxies(),
        App.GetContent(),
        App.GetShareContent(),
        App.GetLinks(),
      ])
      set({
        settings: settings ?? DEFAULT_SETTINGS,
        pagesText: pages.join('\n'),
        proxiesText: proxies.join('\n'),
        contentText: content.join('|'),
        shareContentText: shareContent.join('|'),
        linksText: links.join('\n'),
        ready: true,
      })
    } catch (e) {
      set({ error: String(e), ready: true })
    }
  },

  updateSettings: async (patch) => {
    const next = { ...get().settings, ...patch }
    set({ settings: next })
    try {
      const saved = await App.SaveSettings(next)
      if (saved) set({ settings: saved })
    } catch (e) {
      set({ error: String(e) })
    }
  },

  setLanguage: async (lang) => {
    await get().updateSettings({ language: lang })
  },

  setPagesText: (text) => {
    set({ pagesText: text })
    savePages(text)
  },

  // Applied when the runner consumes pages (success) and pushes the remaining
  // list. The backend already persisted it, so this does NOT trigger a save.
  applyPagesFromBackend: (pages) => {
    set({ pagesText: pages.join('\n') })
  },

  setProxiesText: (text) => {
    set({ proxiesText: text })
    saveProxies(text)
  },

  setContentText: (text) => {
    set({ contentText: text })
    saveContent(text)
  },

  setShareContentText: (text) => {
    set({ shareContentText: text })
    saveShareContent(text)
  },

  setLinksText: (text) => {
    set({ linksText: text })
    saveLinks(text)
  },

  pickImage: async () => {
    try {
      const path = await App.PickImage()
      set({ settings: { ...get().settings, imagePath: path } })
    } catch (e) {
      set({ error: String(e) })
    }
  },

  clearImage: async () => {
    try {
      await App.ClearImage()
      set({ settings: { ...get().settings, imagePath: '' } })
    } catch (e) {
      set({ error: String(e) })
    }
  },

  setError: (message) => set({ error: message }),
}))
