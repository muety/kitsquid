$(() => {
    const btnSidebarToggle = $('#btn-toggle-sidebar'),
        btnSidebarClose = $('#btn-close-sidebar'),
        btnLogout = $('#btn-logout'),
        btnLogin = $('#btn-login'),
        inputSignupPrefix = $('#input-signup-prefix'),
        inputSignupSuffix = $('#input-signup-suffix'),
        inputSignupUser = $('#input-signup-user'),
        inputSignupPassword = $('#input-signup-password'),
        formSignup = $('#form-signup'),
        formLogout = $('#form-logout')

    $(window).click(function() {
        toggleLogoutButton(false)
        toggleLoginButton(false)
    })

    // Sidebar
    btnSidebarToggle.click(toggleSidebar)
    btnSidebarClose.click(toggleSidebar)

    // Login / Logout
    btnLogout.click(() => formLogout.submit())

    // Signup
    if (window.hasOwnProperty('signupConfig')) {
        inputSignupSuffix.change(() => {
            inputSignupPrefix.attr('pattern', window.signupConfig[inputSignupSuffix.val()][1])
            inputSignupPrefix.attr('placeholder', window.signupConfig[inputSignupSuffix.val()][0])
            inputSignupPassword.attr('pattern', window.signupConfig[inputSignupSuffix.val()][2])
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

function toggleLogoutButton(forceState) {
    let open = !$('#btn-logout').hasClass('hidden')
    $('#btn-logout').toggleClass('hidden', forceState === undefined ? open : !forceState)
}

function toggleLoginButton(forceState) {
    let open = !$('#btn-login').hasClass('hidden')
    $('#btn-login').toggleClass('hidden', forceState === undefined ? open : !forceState)
}

function closeAlert(prefix, i) {
    $(`#${prefix}-${i}`).addClass('hidden')
}