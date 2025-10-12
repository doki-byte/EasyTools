// tailwind.config.cjs
module.exports = {
    important: false,
    content: [
        './index.html',
        './src/views/restmate/**/*.{vue,js,jsx,ts,tsx}',
        './src/views/Restmate.vue',
    ],
    theme: {
        extend: {},
    },
    plugins: [],
}
