// falling edge only
export function debounce<T extends (...args: any[]) => any>(
  fn: T,
  pause: number
): (...args: Parameters<T>) => void {
  let t: number;
  return (...args: Parameters<T>) => {
    clearTimeout(t);
    t = setTimeout(() => fn.apply(null, args), pause);
  };
}
