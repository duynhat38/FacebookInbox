// debounce returns a function that delays invoking fn until `wait` ms have
// elapsed since the last call. Used to auto-save text inputs without writing on
// every keystroke.
export function debounce<A extends unknown[]>(
  fn: (...args: A) => void,
  wait: number,
): (...args: A) => void {
  let timer: ReturnType<typeof setTimeout> | undefined
  return (...args: A) => {
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => fn(...args), wait)
  }
}
