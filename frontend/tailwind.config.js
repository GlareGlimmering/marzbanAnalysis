/** @type {import('tailwindcss').Config} */
export default {
    content: [
        "./index.html",
        "./src/**/*.{vue,js,ts,jsx,tsx}",
    ],
    theme: {
        extend: {
            colors: {
                kurz: {
                    purple: '#2A1B4E',
                    magenta: '#FF007F',
                    cyan: '#00F0FF',
                    yellow: '#FFD700',
                    bg: '#0F0C1B'
                }
            }
        },
    },
    plugins: [],
}