$(() => {
    const btnSidebarToggle = $('#btn-toggle-sidebar'),
        btnSidebarClose = $('#btn-close-sidebar'),
        inputSignupPrefix = $('#input-signup-prefix'),
        inputSignupSuffix = $('#input-signup-suffix'),
        inputSignupUser = $('#input-signup-user'),
        inputSignupPassword = $('#input-signup-password'),
        formSignup = $('#form-signup')

    // Sidebar
    btnSidebarToggle.click(toggleSidebar)
    btnSidebarClose.click(toggleSidebar)

    // Signup
    if (document.hasOwnProperty('signupConfig')) {
        inputSignupSuffix.change(() => {
            inputSignupPrefix.attr('pattern', signupConfig[inputSignupSuffix.val()][1])
            inputSignupPrefix.attr('placeholder', signupConfig[inputSignupSuffix.val()][0])
            inputSignupPassword.attr('pattern', signupConfig[inputSignupSuffix.val()][2])
        })
    }

    if (formSignup) {
        formSignup.submit(() => {
            let user = `${inputSignupPrefix.val()}@${inputSignupSuffix.val()}`
            inputSignupUser.val(user)
        })
    }
})

function toggleSidebar() {
    let open = !$('#sidebar').hasClass('hidden')
    $('#sidebar').toggleClass('hidden', open)
    $('#main-content').toggleClass('hidden', !open)
}

function closeAlert(prefix, i) {
    $(`#${prefix}-${i}`).addClass('hidden')
}