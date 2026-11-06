export const PIN_SIZES = [
  { value: 'small', label: 'S', name: 'Small pins', px: 170 },
  { value: 'medium', label: 'M', name: 'Medium pins', px: 236 },
  { value: 'large', label: 'L', name: 'Large pins', px: 340 },
] as const;

export type PinSize = (typeof PIN_SIZES)[number]['value'];

const KEY = 'cork:pin-size';
const saved = localStorage.getItem(KEY);

export const pinSize = $state<{ value: PinSize }>({
  value: PIN_SIZES.some((s) => s.value === saved) ? (saved as PinSize) : 'medium',
});

export function setPinSize(value: PinSize) {
  pinSize.value = value;
  localStorage.setItem(KEY, value);
}
