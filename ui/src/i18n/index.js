import { createI18n } from 'vue-i18n'
import en from './locales/en.json'
import zh from './locales/zh.json'
import es from './locales/es.json'
import de from './locales/de.json'
import fr from './locales/fr.json'
import ja from './locales/ja.json'
import ru from './locales/ru.json'

const i18n = createI18n({
  legacy: false,
  locale: localStorage.getItem('yantr-locale') || 'en',
  fallbackLocale: 'en',
  messages: {
    en,
    zh,
    es,
    de,
    fr,
    ja,
    ru
  }
})

export default i18n
