window.addEventListener('load', function() {
    var el = document.getElementsByClassName('hljs')
    
    for (let i = 0; i < el.length; i++) {
        var lineNumber = 0;
        var lineId;
        
        el[i].innerHTML = el[i].innerHTML.replace(/(^|\n)/g, function() {
            lineNumber++;
            lineId = 'block-' + i + '-line-' + lineNumber
            return arguments[1] + '<a class="line" href="#' + lineId + '" id="' + lineId + '">' + lineNumber + '</a>';
        })
        var lastLine = document.getElementById(lineId);

        lastLine.parentElement.removeChild(lastLine);
    }
});
