const { fontFamily } = require('tailwindcss/defaultTheme')

module.exports = {
    theme: {
        extend: {
            colors: {
                kit: '#009682'
            },
            fontFamily: {
                sans: [
                    'Noto Sans',
                    ...fontFamily.sans
                ]
            }
        }
    },
    variants: {},
    plugins: [],
    corePlugins: {
        gridTemplateColumns: true
    }
}
