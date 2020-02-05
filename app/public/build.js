const fs = require('fs'),
    path = require('path')

const DST_DIR = 'build/'

const ASSETS = {
    base: '',
    files: [
        { src: 'js/ui.js' },
        ...fs.readdirSync('images').map(f => Object.assign({}, { src: `images/${f}` }))
    ]
}

const CORE_UI_ICONS = {
    base: 'node_modules/@coreui/icons/',
    files: [
        { src: 'css/all.min.css', dst: 'css/icons.min.css' },
        ...fs.readdirSync('node_modules/@coreui/icons/fonts').map(f => Object.assign({}, { src: `fonts/${f}` }))
    ]
}

const JQUERY = {
    base: 'node_modules/jquery/dist',
    files: [
        { src: 'jquery.min.js', dst: 'js/jquery.min.js' }
    ]
}

const configs = [ASSETS, CORE_UI_ICONS, JQUERY]

function copyAssets() {
    configs.forEach(copyCfg => {
        console.log(`Copying assets from ${copyCfg.base}`)
        copyCfg.files.forEach(file => {
            const from = path.normalize(path.join(copyCfg.base, file.src))
            const to = path.normalize(path.join(DST_DIR, file.hasOwnProperty('dst') ? file.dst : file.src))
            const toDir = path.join(...(to.split('/').slice(0, -1)))
            if (!fs.existsSync(toDir)) {
                fs.mkdirSync(toDir, {recursive: true})
            }
            fs.copyFileSync(from, to)
        })
    })
}

function run() {
    if (!process.argv.includes('--no-build')) {
        copyAssets()
    }

    if (process.argv.includes('--watch')) {
        console.log('Watching for file system changes ...')

        configs.forEach(copyCfg => {
            copyCfg.files.forEach(file => {
                let ready = true
                const from = path.normalize(path.join(copyCfg.base, file.src))
                const to = path.normalize(path.join(DST_DIR, file.hasOwnProperty('dst') ? file.dst : file.src))

                fs.watch(from, event => {
                    if (!ready) {
                        return
                    }

                    ready = false
                    setTimeout(() => ready = true, 1000)

                    console.log(`${from} updated.`)
                    fs.copyFileSync(from, to)
                })
            })
        })
    }
}

run()