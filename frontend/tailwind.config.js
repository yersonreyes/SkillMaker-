// Tailwind 3 config — Angular @angular/build autodetects this file.
//
// The `content` field is the JIT scanner: only utility classes referenced
// in those files are emitted in the final CSS.
//
// tailwindcss-primeui bridges Tailwind colors with PrimeNG's --p-* CSS
// variables so classes like `bg-primary-500` map to the active theme.

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./src/**/*.{html,ts}'],
  plugins: [require('tailwindcss-primeui')],
};
