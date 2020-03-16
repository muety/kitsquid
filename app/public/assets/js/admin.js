function browse(action) {
    let vm = {
        action,
        entity: $('#select-admin-entity').val().toLowerCase(),
        key: $('#input-admin-browser-key').val(),
        value: $('#textarea-admin-browser-value').val()
    }

    let resultElement = $('#textarea-admin-browser-result')
    resultElement.val('')

    function printResult(data) {
        switch (action) {
            case 'list':
                resultElement.val(data.map(JSON.stringify).join('\n\n###\n\n'))
                break
            default:
                resultElement.val(JSON.stringify(data))
        }
    }

    fetch(`/api/admin/query`, {
        method: 'POST',
        headers: {
            'Accept': 'application/json',
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(vm)
    })
        .then(response => {
            if (!response.ok) {
                return response.json().then((data) => {
                    showSnackbar(`Fehler: ${data.error}`)
                })
            } else if (response.status >= 200 && response.status <= 299) {
                showSnackbar('Erfolgreich')
                if (hasJsonBody(response)) {
                    return response.json().then(printResult)
                }
            } else {
                showSnackbar('Fehler')
            }
        })
        .catch(() => {
            showSnackbar('Fehler')
        })
}

function flushAll() {
    fetch(`/api/admin/flush`, {
        method: 'POST',
        headers: {
            'Accept': 'application/json',
            'Content-Type': 'application/json'
        }
    })
        .then(response => {
            if (!response.ok) {
                return response.json().then((data) => {
                    showSnackbar(`Fehler: ${data.error}`)
                })
            } else if (response.status >= 200 && response.status <= 299) {
                showSnackbar('Erfolgreich')
            }
        })
        .catch(() => {
            showSnackbar('Fehler')
        })
}

function hasJsonBody(response) {
    return response.headers.get('content-type') &&
        response.headers.get('content-type').startsWith('application/json') &&
        (
            (response.headers.get('content-length') && parseInt(response.headers.get('content-length')) > 0) ||
            (response.headers.get('transfer-encoding') === 'chunked')
        )
}