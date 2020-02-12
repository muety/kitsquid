const { fontFamily } = require('tailwindcss/defaultTheme')

module.exports = {
    theme: {
        extend: {
            colors: {
                'kit': '#009682',
                'kit-dark': '#007061'
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
    plugins: [
        require('@tailwindcss/custom-forms'),
    ],
    corePlugins: {
        gridTemplateColumns: true
    }
}
