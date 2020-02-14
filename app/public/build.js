const fs = require('fs'),
    path = require('path')

const ASSET_DIR = 'assets/'
const DST_DIR = 'build/'

const ASSETS = {
    base: '',
    files: [
        ...fs.readdirSync(`${ASSET_DIR}/css`).filter(f => !f.includes('tailwind')).map(f => Object.assign({}, { src: `${ASSET_DIR}/css/${f}`, dst: `css/${f}` })),
        ...fs.readdirSync(`${ASSET_DIR}/js`).map(f => Object.assign({}, { src: `${ASSET_DIR}/js/${f}`, dst: `js/${f}` })),
        ...fs.readdirSync(`${ASSET_DIR}/images`).map(f => Object.assign({}, { src: `${ASSET_DIR}/images/${f}`, dst: `images/${f}` })),
        ...fs.readdirSync(`${ASSET_DIR}/font`).map(f => Object.assign({}, { src: `${ASSET_DIR}/font/${f}`, dst: `font/${f}` }))
    ]
}

const JQUERY = {
    base: 'node_modules/jquery/dist',
    files: [
        { src: 'jquery.min.js', dst: 'js/jquery.min.js' }
    ]
}

const configs = [ASSETS, JQUERY]

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