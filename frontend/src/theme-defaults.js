export const CV_DEFAULT_KEY = 'spicebag:default-theme-cv'
export const CL_DEFAULT_KEY = 'spicebag:default-theme-cl'

export function getDefaultTheme(key) {
  return localStorage.getItem(key) ?? ''
}
