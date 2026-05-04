(function(){
  function copyText(text, okMsg, errMsg){
    text = String(text||'');
    if(!text)return;
    okMsg = okMsg || 'Скопировано';
    errMsg = errMsg || 'Не удалось скопировать';
    var done=false;
    if(navigator.clipboard && window.isSecureContext){
      navigator.clipboard.writeText(text).then(function(){toastMsg(okMsg,true)}).catch(function(){fallback()});
      return;
    }
    fallback();
    function fallback(){
      try{
        var ta=document.createElement('textarea');
        ta.value=text;
        ta.style.position='fixed';ta.style.top='-9999px';ta.style.opacity='0';
        document.body.appendChild(ta);
        ta.select();
        done=document.execCommand('copy');
        document.body.removeChild(ta);
        toastMsg(done?okMsg:errMsg,done);
      }catch(e){toastMsg(errMsg,false)}
    }
    function toastMsg(m,ok){
      if(typeof window.showToast==='function')window.showToast(m,ok);
      else if(typeof window.toast==='function')window.toast(m,ok);
      else alert(m);
    }
  }
  window.lastopCopy=copyText;
})();
