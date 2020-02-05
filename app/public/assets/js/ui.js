$(() => {
    $('#btn-toggle-sidebar').click(toggleSidebar)
    $('#btn-close-sidebar').click(toggleSidebar)
})

function toggleSidebar() {
    let open = !$('#sidebar').hasClass('hidden')
    $('#sidebar').toggleClass('hidden', open)
    $('#main-content').toggleClass('hidden', !open)
}