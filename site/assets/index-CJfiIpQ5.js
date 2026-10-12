(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const c of document.querySelectorAll('link[rel="modulepreload"]'))r(c);new MutationObserver(c=>{for(const f of c)if(f.type==="childList")for(const d of f.addedNodes)d.tagName==="LINK"&&d.rel==="modulepreload"&&r(d)}).observe(document,{childList:!0,subtree:!0});function i(c){const f={};return c.integrity&&(f.integrity=c.integrity),c.referrerPolicy&&(f.referrerPolicy=c.referrerPolicy),c.crossOrigin==="use-credentials"?f.credentials="include":c.crossOrigin==="anonymous"?f.credentials="omit":f.credentials="same-origin",f}function r(c){if(c.ep)return;c.ep=!0;const f=i(c);fetch(c.href,f)}})();var oh={exports:{}},al={};/**
 * @license React
 * react-jsx-runtime.production.js
 *
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var rv;function BM(){if(rv)return al;rv=1;var o=Symbol.for("react.transitional.element"),t=Symbol.for("react.fragment");function i(r,c,f){var d=null;if(f!==void 0&&(d=""+f),c.key!==void 0&&(d=""+c.key),"key"in c){f={};for(var h in c)h!=="key"&&(f[h]=c[h])}else f=c;return c=f.ref,{$$typeof:o,type:r,key:d,ref:c!==void 0?c:null,props:f}}return al.Fragment=t,al.jsx=i,al.jsxs=i,al}var ov;function zM(){return ov||(ov=1,oh.exports=BM()),oh.exports}var w=zM(),lh={exports:{}},rt={};/**
 * @license React
 * react.production.js
 *
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var lv;function FM(){if(lv)return rt;lv=1;var o=Symbol.for("react.transitional.element"),t=Symbol.for("react.portal"),i=Symbol.for("react.fragment"),r=Symbol.for("react.strict_mode"),c=Symbol.for("react.profiler"),f=Symbol.for("react.consumer"),d=Symbol.for("react.context"),h=Symbol.for("react.forward_ref"),m=Symbol.for("react.suspense"),g=Symbol.for("react.memo"),x=Symbol.for("react.lazy"),_=Symbol.for("react.activity"),v=Symbol.for("react.view_transition"),E=Symbol.iterator;function R(H){return H===null||typeof H!="object"?null:(H=E&&H[E]||H["@@iterator"],typeof H=="function"?H:null)}var N={isMounted:function(){return!1},enqueueForceUpdate:function(){},enqueueReplaceState:function(){},enqueueSetState:function(){}},y=Object.assign,S={};function z(H,F,ge){this.props=H,this.context=F,this.refs=S,this.updater=ge||N}z.prototype.isReactComponent={},z.prototype.setState=function(H,F){if(typeof H!="object"&&typeof H!="function"&&H!=null)throw Error("takes an object of state variables to update or a function which returns an object of state variables.");this.updater.enqueueSetState(this,H,F,"setState")},z.prototype.forceUpdate=function(H){this.updater.enqueueForceUpdate(this,H,"forceUpdate")};function W(){}W.prototype=z.prototype;function C(H,F,ge){this.props=H,this.context=F,this.refs=S,this.updater=ge||N}var D=C.prototype=new W;D.constructor=C,y(D,z.prototype),D.isPureReactComponent=!0;var U=Array.isArray;function I(){}var b={H:null,A:null,T:null,S:null},L=Object.prototype.hasOwnProperty;function k(H,F,ge){var X=ge.ref;return{$$typeof:o,type:H,key:F,ref:X!==void 0?X:null,props:ge}}function Y(H,F){return k(H.type,F,H.props)}function $(H){return typeof H=="object"&&H!==null&&H.$$typeof===o}function O(H){var F={"=":"=0",":":"=2"};return"$"+H.replace(/[=:]/g,function(ge){return F[ge]})}var q=/\/+/g;function ne(H,F){return typeof H=="object"&&H!==null&&H.key!=null?O(""+H.key):F.toString(36)}function j(H){switch(H.status){case"fulfilled":return H.value;case"rejected":throw H.reason;default:switch(typeof H.status=="string"?H.then(I,I):(H.status="pending",H.then(function(F){H.status==="pending"&&(H.status="fulfilled",H.value=F)},function(F){H.status==="pending"&&(H.status="rejected",H.reason=F)})),H.status){case"fulfilled":return H.value;case"rejected":throw H.reason}}throw H}function Q(H,F,ge,X,ue){var Ae=typeof H;(Ae==="undefined"||Ae==="boolean")&&(H=null);var Ie=!1;if(H===null)Ie=!0;else switch(Ae){case"bigint":case"string":case"number":Ie=!0;break;case"object":switch(H.$$typeof){case o:case t:Ie=!0;break;case x:return Ie=H._init,Q(Ie(H._payload),F,ge,X,ue)}}if(Ie)return ue=ue(H),Ie=X===""?"."+ne(H,0):X,U(ue)?(ge="",Ie!=null&&(ge=Ie.replace(q,"$&/")+"/"),Q(ue,F,ge,"",function(Vt){return Vt})):ue!=null&&($(ue)&&(ue=Y(ue,ge+(ue.key==null||H&&H.key===ue.key?"":(""+ue.key).replace(q,"$&/")+"/")+Ie)),F.push(ue)),1;Ie=0;var xe=X===""?".":X+":";if(U(H))for(var we=0;we<H.length;we++)X=H[we],Ae=xe+ne(X,we),Ie+=Q(X,F,ge,Ae,ue);else if(we=R(H),typeof we=="function")for(H=we.call(H),we=0;!(X=H.next()).done;)X=X.value,Ae=xe+ne(X,we++),Ie+=Q(X,F,ge,Ae,ue);else if(Ae==="object"){if(typeof H.then=="function")return Q(j(H),F,ge,X,ue);throw F=String(H),Error("Objects are not valid as a React child (found: "+(F==="[object Object]"?"object with keys {"+Object.keys(H).join(", ")+"}":F)+"). If you meant to render a collection of children, use an array instead.")}return Ie}function pe(H,F,ge){if(H==null)return H;var X=[],ue=0;return Q(H,X,"","",function(Ae){return F.call(ge,Ae,ue++)}),X}function ce(H){if(H._status===-1){var F=H._result,ge=F();ge.then(function(X){(H._status===0||H._status===-1)&&(H._status=1,H._result=X,ge.status===void 0&&(ge.status="fulfilled",ge.value=X))},function(X){(H._status===0||H._status===-1)&&(H._status=2,H._result=X,ge.status===void 0&&(ge.status="rejected",ge.reason=X))}),H._status===-1&&(H._status=0,H._result=ge)}if(H._status===1)return H._result.default;throw H._result}var me=typeof reportError=="function"?reportError:function(H){if(typeof window=="object"&&typeof window.ErrorEvent=="function"){var F=new window.ErrorEvent("error",{bubbles:!0,cancelable:!0,message:typeof H=="object"&&H!==null&&typeof H.message=="string"?String(H.message):String(H),error:H});if(!window.dispatchEvent(F))return}else if(typeof process=="object"&&typeof process.emit=="function"){process.emit("uncaughtException",H);return}console.error(H)};function be(H){var F=b.T,ge={};ge.types=F!==null?F.types:null,b.T=ge;try{var X=H(),ue=b.S;ue!==null&&ue(ge,X),typeof X=="object"&&X!==null&&typeof X.then=="function"&&X.then(I,me)}catch(Ae){me(Ae)}finally{F!==null&&ge.types!==null&&(F.types=ge.types),b.T=F}}function Ze(H){var F=b.T;if(F!==null){var ge=F.types;ge===null?F.types=[H]:ge.indexOf(H)===-1&&ge.push(H)}else be(Ze.bind(null,H))}var je={map:pe,forEach:function(H,F,ge){pe(H,function(){F.apply(this,arguments)},ge)},count:function(H){var F=0;return pe(H,function(){F++}),F},toArray:function(H){return pe(H,function(F){return F})||[]},only:function(H){if(!$(H))throw Error("React.Children.only expected to receive a single React element child.");return H}};return rt.Activity=_,rt.Children=je,rt.Component=z,rt.Fragment=i,rt.Profiler=c,rt.PureComponent=C,rt.StrictMode=r,rt.Suspense=m,rt.ViewTransition=v,rt.__CLIENT_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE=b,rt.__COMPILER_RUNTIME={__proto__:null,c:function(H){return b.H.useMemoCache(H)}},rt.addTransitionType=Ze,rt.cache=function(H){return function(){return H.apply(null,arguments)}},rt.cacheSignal=function(){return null},rt.cloneElement=function(H,F,ge){if(H==null)throw Error("The argument must be a React element, but you passed "+H+".");var X=y({},H.props),ue=H.key;if(F!=null)for(Ae in F.key!==void 0&&(ue=""+F.key),F)!L.call(F,Ae)||Ae==="key"||Ae==="__self"||Ae==="__source"||Ae==="ref"&&F.ref===void 0||(X[Ae]=F[Ae]);var Ae=arguments.length-2;if(Ae===1)X.children=ge;else if(1<Ae){for(var Ie=Array(Ae),xe=0;xe<Ae;xe++)Ie[xe]=arguments[xe+2];X.children=Ie}return k(H.type,ue,X)},rt.createContext=function(H){return H={$$typeof:d,_currentValue:H,_currentValue2:H,_threadCount:0,Provider:null,Consumer:null},H.Provider=H,H.Consumer={$$typeof:f,_context:H},H},rt.createElement=function(H,F,ge){var X,ue={},Ae=null;if(F!=null)for(X in F.key!==void 0&&(Ae=""+F.key),F)L.call(F,X)&&X!=="key"&&X!=="__self"&&X!=="__source"&&(ue[X]=F[X]);var Ie=arguments.length-2;if(Ie===1)ue.children=ge;else if(1<Ie){for(var xe=Array(Ie),we=0;we<Ie;we++)xe[we]=arguments[we+2];ue.children=xe}if(H&&H.defaultProps)for(X in Ie=H.defaultProps,Ie)ue[X]===void 0&&(ue[X]=Ie[X]);return k(H,Ae,ue)},rt.createRef=function(){return{current:null}},rt.forwardRef=function(H){return{$$typeof:h,render:H}},rt.isValidElement=$,rt.lazy=function(H){return{$$typeof:x,_payload:{_status:-1,_result:H},_init:ce}},rt.memo=function(H,F){return{$$typeof:g,type:H,compare:F===void 0?null:F}},rt.startTransition=be,rt.unstable_useCacheRefresh=function(){return b.H.useCacheRefresh()},rt.use=function(H){return b.H.use(H)},rt.useActionState=function(H,F,ge){return b.H.useActionState(H,F,ge)},rt.useCallback=function(H,F){return b.H.useCallback(H,F)},rt.useContext=function(H){return b.H.useContext(H)},rt.useDebugValue=function(){},rt.useDeferredValue=function(H,F){return b.H.useDeferredValue(H,F)},rt.useEffect=function(H,F){return b.H.useEffect(H,F)},rt.useEffectEvent=function(H){return b.H.useEffectEvent(H)},rt.useId=function(){return b.H.useId()},rt.useImperativeHandle=function(H,F,ge){return b.H.useImperativeHandle(H,F,ge)},rt.useInsertionEffect=function(H,F){return b.H.useInsertionEffect(H,F)},rt.useLayoutEffect=function(H,F){return b.H.useLayoutEffect(H,F)},rt.useMemo=function(H,F){return b.H.useMemo(H,F)},rt.useOptimistic=function(H,F){return b.H.useOptimistic(H,F)},rt.useReducer=function(H,F,ge){return b.H.useReducer(H,F,ge)},rt.useRef=function(H){return b.H.useRef(H)},rt.useState=function(H){return b.H.useState(H)},rt.useSyncExternalStore=function(H,F,ge){return b.H.useSyncExternalStore(H,F,ge)},rt.useTransition=function(){return b.H.useTransition()},rt.version="19.3.0",rt}var cv;function zp(){return cv||(cv=1,lh.exports=FM()),lh.exports}var Gt=zp(),ch={exports:{}},sl={},uh={exports:{}},fh={};/**
 * @license React
 * scheduler.production.js
 *
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var uv;function HM(){return uv||(uv=1,(function(o){function t(j,Q){var pe=j.length;j.push(Q);e:for(;0<pe;){var ce=pe-1>>>1,me=j[ce];if(0<c(me,Q))j[ce]=Q,j[pe]=me,pe=ce;else break e}}function i(j){return j.length===0?null:j[0]}function r(j){if(j.length===0)return null;var Q=j[0],pe=j.pop();if(pe!==Q){j[0]=pe;e:for(var ce=0,me=j.length,be=me>>>1;ce<be;){var Ze=2*(ce+1)-1,je=j[Ze],H=Ze+1,F=j[H];if(0>c(je,pe))H<me&&0>c(F,je)?(j[ce]=F,j[H]=pe,ce=H):(j[ce]=je,j[Ze]=pe,ce=Ze);else if(H<me&&0>c(F,pe))j[ce]=F,j[H]=pe,ce=H;else break e}}return Q}function c(j,Q){var pe=j.sortIndex-Q.sortIndex;return pe!==0?pe:j.id-Q.id}if(o.unstable_now=void 0,typeof performance=="object"&&typeof performance.now=="function"){var f=performance;o.unstable_now=function(){return f.now()}}else{var d=Date,h=d.now();o.unstable_now=function(){return d.now()-h}}var m=[],g=[],x=1,_=null,v=3,E=!1,R=!1,N=!1,y=!1,S=typeof setTimeout=="function"?setTimeout:null,z=typeof clearTimeout=="function"?clearTimeout:null,W=typeof setImmediate<"u"?setImmediate:null;function C(j){for(var Q=i(g);Q!==null;){if(Q.callback===null)r(g);else if(Q.startTime<=j)r(g),Q.sortIndex=Q.expirationTime,t(m,Q);else break;Q=i(g)}}function D(j){if(N=!1,C(j),!R)if(i(m)!==null)R=!0,U||(U=!0,$());else{var Q=i(g);Q!==null&&ne(D,Q.startTime-j)}}var U=!1,I=-1,b=5,L=-1;function k(){return y?!0:!(o.unstable_now()-L<b)}function Y(){if(y=!1,U){var j=o.unstable_now();L=j;var Q=!0;try{e:{R=!1,N&&(N=!1,z(I),I=-1),E=!0;var pe=v;try{t:{for(C(j),_=i(m);_!==null&&!(_.expirationTime>j&&k());){var ce=_.callback;if(typeof ce=="function"){_.callback=null,v=_.priorityLevel;var me=ce(_.expirationTime<=j);if(j=o.unstable_now(),typeof me=="function"){_.callback=me,C(j),Q=!0;break t}_===i(m)&&r(m),C(j)}else r(m);_=i(m)}if(_!==null)Q=!0;else{var be=i(g);be!==null&&ne(D,be.startTime-j),Q=!1}}break e}finally{_=null,v=pe,E=!1}Q=void 0}}finally{Q?$():U=!1}}}var $;if(typeof W=="function")$=function(){W(Y)};else if(typeof MessageChannel<"u"){var O=new MessageChannel,q=O.port2;O.port1.onmessage=Y,$=function(){q.postMessage(null)}}else $=function(){S(Y,0)};function ne(j,Q){I=S(function(){j(o.unstable_now())},Q)}o.unstable_IdlePriority=5,o.unstable_ImmediatePriority=1,o.unstable_LowPriority=4,o.unstable_NormalPriority=3,o.unstable_Profiling=null,o.unstable_UserBlockingPriority=2,o.unstable_cancelCallback=function(j){j.callback=null},o.unstable_forceFrameRate=function(j){0>j||125<j?console.error("forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported"):b=0<j?Math.floor(1e3/j):5},o.unstable_getCurrentPriorityLevel=function(){return v},o.unstable_next=function(j){switch(v){case 1:case 2:case 3:var Q=3;break;default:Q=v}var pe=v;v=Q;try{return j()}finally{v=pe}},o.unstable_requestPaint=function(){y=!0},o.unstable_runWithPriority=function(j,Q){switch(j){case 1:case 2:case 3:case 4:case 5:break;default:j=3}var pe=v;v=j;try{return Q()}finally{v=pe}},o.unstable_scheduleCallback=function(j,Q,pe){var ce=o.unstable_now();switch(typeof pe=="object"&&pe!==null?(pe=pe.delay,pe=typeof pe=="number"&&0<pe?ce+pe:ce):pe=ce,j){case 1:var me=-1;break;case 2:me=250;break;case 5:me=1073741823;break;case 4:me=1e4;break;default:me=5e3}return me=pe+me,j={id:x++,callback:Q,priorityLevel:j,startTime:pe,expirationTime:me,sortIndex:-1},pe>ce?(j.sortIndex=pe,t(g,j),i(m)===null&&j===i(g)&&(N?(z(I),I=-1):N=!0,ne(D,pe-ce))):(j.sortIndex=me,t(m,j),R||E||(R=!0,U||(U=!0,$()))),j},o.unstable_shouldYield=k,o.unstable_wrapCallback=function(j){var Q=v;return function(){var pe=v;v=Q;try{return j.apply(this,arguments)}finally{v=pe}}}})(fh)),fh}var fv;function GM(){return fv||(fv=1,uh.exports=HM()),uh.exports}var dh={exports:{}},Dn={};/**
 * @license React
 * react-dom.production.js
 *
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var dv;function VM(){if(dv)return Dn;dv=1;var o=zp();function t(x){var _="https://react.dev/errors/"+x;if(1<arguments.length){_+="?args[]="+encodeURIComponent(arguments[1]);for(var v=2;v<arguments.length;v++)_+="&args[]="+encodeURIComponent(arguments[v])}return"Minified React error #"+x+"; visit "+_+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}function i(){}var r={d:{f:i,r:function(){throw Error(t(522))},D:i,C:i,L:i,m:i,X:i,S:i,M:i},p:0,findDOMNode:null},c=Symbol.for("react.portal"),f=Symbol.for("react.recoverable"),d=Symbol.for("react.optimistic_key");function h(x,_,v){var E=3<arguments.length&&arguments[3]!==void 0?arguments[3]:null;return{$$typeof:c,key:E==null?null:E===d?d:""+E,children:x,containerInfo:_,implementation:v}}var m=o.__CLIENT_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE;function g(x,_){if(x==="font")return"";if(typeof _=="string")return _==="use-credentials"?_:""}return Dn.__DOM_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE=r,Dn.browser=function(x){return{$$typeof:f,_reason:x}},Dn.createPortal=function(x,_){var v=2<arguments.length&&arguments[2]!==void 0?arguments[2]:null;if(!_||_.nodeType!==1&&_.nodeType!==9&&_.nodeType!==11)throw Error(t(299));return h(x,_,null,v)},Dn.flushSync=function(x){var _=m.T,v=r.p;try{if(m.T=null,r.p=2,x)return x()}finally{m.T=_,r.p=v,r.d.f()}},Dn.preconnect=function(x,_){typeof x=="string"&&(_?(_=_.crossOrigin,_=typeof _=="string"?_==="use-credentials"?_:"":void 0):_=null,r.d.C(x,_))},Dn.prefetchDNS=function(x){typeof x=="string"&&r.d.D(x)},Dn.preinit=function(x,_){if(typeof x=="string"&&_&&typeof _.as=="string"){var v=_.as,E=g(v,_.crossOrigin),R=typeof _.integrity=="string"?_.integrity:void 0,N=typeof _.fetchPriority=="string"?_.fetchPriority:void 0;v==="style"?r.d.S(x,typeof _.precedence=="string"?_.precedence:void 0,{crossOrigin:E,integrity:R,fetchPriority:N}):v==="script"&&r.d.X(x,{crossOrigin:E,integrity:R,fetchPriority:N,nonce:typeof _.nonce=="string"?_.nonce:void 0})}},Dn.preinitModule=function(x,_){if(typeof x=="string")if(typeof _=="object"&&_!==null){if(_.as==null||_.as==="script"){var v=g(_.as,_.crossOrigin);r.d.M(x,{crossOrigin:v,integrity:typeof _.integrity=="string"?_.integrity:void 0,nonce:typeof _.nonce=="string"?_.nonce:void 0,fetchPriority:typeof _.fetchPriority=="string"?_.fetchPriority:void 0})}}else _==null&&r.d.M(x)},Dn.preload=function(x,_){if(typeof x=="string"&&typeof _=="object"&&_!==null&&typeof _.as=="string"){var v=_.as,E=g(v,_.crossOrigin);r.d.L(x,v,{crossOrigin:E,integrity:typeof _.integrity=="string"?_.integrity:void 0,nonce:typeof _.nonce=="string"?_.nonce:void 0,type:typeof _.type=="string"?_.type:void 0,fetchPriority:typeof _.fetchPriority=="string"?_.fetchPriority:void 0,referrerPolicy:typeof _.referrerPolicy=="string"?_.referrerPolicy:void 0,imageSrcSet:typeof _.imageSrcSet=="string"?_.imageSrcSet:void 0,imageSizes:typeof _.imageSizes=="string"?_.imageSizes:void 0,media:typeof _.media=="string"?_.media:void 0})}},Dn.preloadModule=function(x,_){if(typeof x=="string")if(_){var v=g(_.as,_.crossOrigin);r.d.m(x,{as:typeof _.as=="string"&&_.as!=="script"?_.as:void 0,crossOrigin:v,integrity:typeof _.integrity=="string"?_.integrity:void 0,nonce:typeof _.nonce=="string"?_.nonce:void 0,fetchPriority:typeof _.fetchPriority=="string"?_.fetchPriority:void 0})}else r.d.m(x)},Dn.requestFormReset=function(x){r.d.r(x)},Dn.unstable_batchedUpdates=function(x,_){return x(_)},Dn.useFormState=function(x,_,v){return m.H.useFormState(x,_,v)},Dn.useFormStatus=function(){return m.H.useHostTransitionStatus()},Dn.version="19.3.0",Dn}var hv;function kM(){if(hv)return dh.exports;hv=1;function o(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(o)}catch(t){console.error(t)}}return o(),dh.exports=VM(),dh.exports}/**
 * @license React
 * react-dom-client.production.js
 *
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var pv;function XM(){if(pv)return sl;pv=1;var o=GM(),t=zp(),i=kM();function r(e){var n="https://react.dev/errors/"+e;if(1<arguments.length){n+="?args[]="+encodeURIComponent(arguments[1]);for(var a=2;a<arguments.length;a++)n+="&args[]="+encodeURIComponent(arguments[a])}return"Minified React error #"+e+"; visit "+n+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}function c(e){return!(!e||e.nodeType!==1&&e.nodeType!==9&&e.nodeType!==11)}function f(e){for(var n=e,a=n;a&&!a.alternate;)n=a,(n.flags&4098)!==0&&(e=n.return),a=n.return;for(;n.return;)n=n.return;return n.tag===3?e:null}function d(e){if(e.tag===13){var n=e.memoizedState;if(n===null&&(e=e.alternate,e!==null&&(n=e.memoizedState)),n!==null)return n.dehydrated}return null}function h(e){if(e.tag===31){var n=e.memoizedState;if(n===null&&(e=e.alternate,e!==null&&(n=e.memoizedState)),n!==null)return n.dehydrated}return null}function m(e){if(f(e)!==e)throw Error(r(188))}function g(e){var n=e.alternate;if(!n){if(n=f(e),n===null)throw Error(r(188));return n!==e?null:e}for(var a=e,s=n;;){var l=a.return;if(l===null)break;var u=l.alternate;if(u===null){if(s=l.return,s!==null){a=s;continue}break}if(l.child===u.child){for(u=l.child;u;){if(u===a)return m(l),e;if(u===s)return m(l),n;u=u.sibling}throw Error(r(188))}if(a.return!==s.return)a=l,s=u;else{for(var p=!1,A=l.child;A;){if(A===a){p=!0,a=l,s=u;break}if(A===s){p=!0,s=l,a=u;break}A=A.sibling}if(!p){for(A=u.child;A;){if(A===a){p=!0,a=u,s=l;break}if(A===s){p=!0,s=u,a=l;break}A=A.sibling}if(!p)throw Error(r(189))}}if(a.alternate!==s)throw Error(r(190))}if(a.tag!==3)throw Error(r(188));return a.stateNode.current===a?e:n}function x(e){var n=e.tag;if(n===5||n===26||n===27||n===6)return e;for(e=e.child;e!==null;){if(n=x(e),n!==null)return n;e=e.sibling}return null}function _(e,n,a,s,l,u){for(;e!==null;){if((e.tag===5||e.tag===27||e.tag===6)&&a(e,s,l,u)||(e.tag!==22||e.memoizedState===null)&&(n||e.tag!==5&&e.tag!==27)&&_(e.child,n,a,s,l,u))return!0;e=e.sibling}return!1}function v(e){for(e=e.return;e!==null;){if(e.tag===3||e.tag===5||e.tag===27)return e;e=e.return}return null}function E(e){var n=!1;for(e=e.return;e!==null&&(e.tag===4&&(n=!0),!(e.tag===3||e.tag===5||e.tag===27));)e=e.return;return n}function R(e){var n=[null,null],a=v(e);return a===null||N(n,e,a.child,{foundSelf:!1}),n}function N(e,n,a,s){for(;a!==null;){if(a===n)s.foundSelf=!0;else if(a.tag===5||a.tag===27||a.tag===6){if(s.foundSelf)return e[1]=a,!0;e[0]=a}else if((a.tag!==22||a.memoizedState===null)&&N(e,n,a.child,s))return!0;a=a.sibling}return!1}function y(e){switch(e.tag){case 5:case 27:case 6:return e.stateNode;case 3:return e.stateNode.containerInfo;default:throw Error(r(559))}}var S=null,z=null;function W(e,n,a){return e===a?!0:e===n?(S=e,!0):!1}function C(e,n,a){return e===a?(z=e,!1):e===n?(z!==null&&(S=e),!0):!1}function D(e){if(e===null)return null;do e=e===null?null:e.return;while(e&&e.tag!==5&&e.tag!==27&&e.tag!==3);return e||null}function U(e,n,a){for(var s=0,l=e;l;l=a(l))s++;l=0;for(var u=n;u;u=a(u))l++;for(;0<s-l;)e=a(e),s--;for(;0<l-s;)n=a(n),l--;for(;s--;){if(e===n||n!==null&&e===n.alternate)return e;e=a(e),n=a(n)}return null}var I=Object.assign,b=Symbol.for("react.element"),L=Symbol.for("react.transitional.element"),k=Symbol.for("react.portal"),Y=Symbol.for("react.fragment"),$=Symbol.for("react.strict_mode"),O=Symbol.for("react.profiler"),q=Symbol.for("react.consumer"),ne=Symbol.for("react.context"),j=Symbol.for("react.forward_ref"),Q=Symbol.for("react.suspense"),pe=Symbol.for("react.suspense_list"),ce=Symbol.for("react.memo"),me=Symbol.for("react.lazy"),be=Symbol.for("react.activity"),Ze=Symbol.for("react.legacy_hidden"),je=Symbol.for("react.memo_cache_sentinel"),H=Symbol.for("react.view_transition"),F=Symbol.for("react.recoverable"),ge=Symbol.iterator;function X(e){return e===null||typeof e!="object"?null:(e=ge&&e[ge]||e["@@iterator"],typeof e=="function"?e:null)}var ue=Symbol.for("react.client.reference");function Ae(e){if(e==null)return null;if(typeof e=="function")return e.$$typeof===ue?null:e.displayName||e.name||null;if(typeof e=="string")return e;switch(e){case Y:return"Fragment";case O:return"Profiler";case $:return"StrictMode";case Q:return"Suspense";case pe:return"SuspenseList";case be:return"Activity";case H:return"ViewTransition"}if(typeof e=="object")switch(e.$$typeof){case k:return"Portal";case ne:return e.displayName||"Context";case q:return(e._context.displayName||"Context")+".Consumer";case j:var n=e.render;return e=e.displayName,e||(e=n.displayName||n.name||"",e=e!==""?"ForwardRef("+e+")":"ForwardRef"),e;case ce:return n=e.displayName||null,n!==null?n:Ae(e.type)||"Memo";case me:n=e._payload,e=e._init;try{return Ae(e(n))}catch{}}return null}var Ie=Array.isArray,xe=t.__CLIENT_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE,we=i.__DOM_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE,Vt={pending:!1,data:null,method:null,action:null},pt=[],gt=-1;function St(e){return{current:e}}function tt(e){0>gt||(e.current=pt[gt],pt[gt]=null,gt--)}function it(e,n){gt++,pt[gt]=e.current,e.current=n}var kt=St(null),dn=St(null),Pt=St(null),en=St(null);function K(e,n){switch(it(Pt,n),it(dn,e),it(kt,null),n.nodeType){case 9:case 11:e=(e=n.documentElement)&&(e=e.namespaceURI)?m_(e):0;break;default:if(e=n.tagName,n=n.namespaceURI)n=m_(n),e=g_(n,e);else switch(e){case"svg":e=1;break;case"math":e=2;break;default:e=0}}tt(kt),it(kt,e)}function nn(){tt(kt),tt(dn),tt(Pt)}function Ot(e){var n=e.memoizedState;n!==null&&(zr._currentValue=n.memoizedState,it(en,e)),n=kt.current;var a=g_(n,e.type);n!==a&&(it(dn,e),it(kt,a))}function P(e){dn.current===e&&(tt(kt),tt(dn)),en.current===e&&(tt(en),zr._currentValue=Vt)}var M,ie;function le(e){if(M===void 0)try{throw Error()}catch(a){var n=a.stack.trim().match(/\n( *(at )?)/);M=n&&n[1]||"",ie=-1<a.stack.indexOf(`
    at`)?" (<anonymous>)":-1<a.stack.indexOf("@")?"@unknown:0:0":""}return`
`+M+e+ie}var _e=!1;function Re(e,n){if(!e||_e)return"";_e=!0;var a=Error.prepareStackTrace;Error.prepareStackTrace=void 0;try{var s={DetermineComponentFrameRoot:function(){try{if(n){var ye=function(){throw Error()};if(Object.defineProperty(ye.prototype,"props",{set:function(){throw Error()}}),typeof Reflect=="object"&&Reflect.construct){try{Reflect.construct(ye,[])}catch(Oe){var J=Oe}Reflect.construct(e,[],ye)}else{try{ye.call()}catch(Oe){J=Oe}ye=!1;try{var oe=Object.getOwnPropertyDescriptor(e.prototype,"props");Object.defineProperty(e.prototype,"props",{configurable:!0,set:function(){throw Error()}}),ye=!0,new e}finally{ye&&(oe!==void 0?Object.defineProperty(e.prototype,"props",oe):delete e.prototype.props)}}}else{try{throw Error()}catch(Oe){J=Oe}(ye=e())&&typeof ye.catch=="function"&&ye.catch(function(){})}}catch(Oe){if(Oe&&J&&typeof Oe.stack=="string")return[Oe.stack,J.stack]}return[null,null]}};s.DetermineComponentFrameRoot.displayName="DetermineComponentFrameRoot";var l=Object.getOwnPropertyDescriptor(s.DetermineComponentFrameRoot,"name");l&&l.configurable&&Object.defineProperty(s.DetermineComponentFrameRoot,"name",{value:"DetermineComponentFrameRoot"});var u=s.DetermineComponentFrameRoot(),p=u[0],A=u[1];if(p&&A){var B=p.split(`
`),te=A.split(`
`);for(l=s=0;s<B.length&&!B[s].includes("DetermineComponentFrameRoot");)s++;for(;l<te.length&&!te[l].includes("DetermineComponentFrameRoot");)l++;if(s===B.length||l===te.length)for(s=B.length-1,l=te.length-1;1<=s&&0<=l&&B[s]!==te[l];)l--;for(;1<=s&&0<=l;s--,l--)if(B[s]!==te[l]){if(s!==1||l!==1)do if(s--,l--,0>l||B[s]!==te[l]){var fe=`
`+B[s].replace(" at new "," at ");return e.displayName&&fe.includes("<anonymous>")&&(fe=fe.replace("<anonymous>",e.displayName)),fe}while(1<=s&&0<=l);break}}}finally{_e=!1,Error.prepareStackTrace=a}return(a=e?e.displayName||e.name:"")?le(a):""}function De(e,n){switch(e.tag){case 26:case 27:case 5:return le(e.type);case 16:return le("Lazy");case 13:return e.child!==n&&n!==null?le("Suspense Fallback"):le("Suspense");case 19:return le("SuspenseList");case 0:case 15:return Re(e.type,!1);case 11:return Re(e.type.render,!1);case 1:return Re(e.type,!0);case 31:return le("Activity");case 30:return le("ViewTransition");default:return""}}function ve(e){try{var n="",a=null;do n+=De(e,a),a=e,e=e.return;while(e);return n}catch(s){return`
Error generating stack: `+s.message+`
`+s.stack}}var Ee=Object.prototype.hasOwnProperty,Ne=o.unstable_scheduleCallback,$e=o.unstable_cancelCallback,Be=o.unstable_shouldYield,Pe=o.unstable_requestPaint,ke=o.unstable_now,nt=o.unstable_getCurrentPriorityLevel,lt=o.unstable_ImmediatePriority,Z=o.unstable_UserBlockingPriority,Ce=o.unstable_NormalPriority,Me=o.unstable_LowPriority,Ue=o.unstable_IdlePriority,Ve=o.log,Te=o.unstable_setDisableYieldValue,Je=null,Ge=null;function Rt(e){if(typeof Ve=="function"&&Te(e),Ge&&typeof Ge.setStrictMode=="function")try{Ge.setStrictMode(Je,e)}catch{}}var ct=Math.clz32?Math.clz32:Pu,Jn=Math.log,di=Math.LN2;function Pu(e){return e>>>=0,e===0?32:31-(Jn(e)/di|0)|0}var er=256,Ss=262144,Ba=4194304;function fa(e){var n=e&42;if(n!==0)return n;switch(e&-e){case 1:return 1;case 2:return 2;case 4:return 4;case 8:return 8;case 16:return 16;case 32:return 32;case 64:return 64;case 128:return 128;case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:return e&-e;case 262144:case 524288:case 1048576:case 2097152:return e&3932160;case 4194304:case 8388608:case 16777216:case 33554432:return e&62914560;case 67108864:return 67108864;case 134217728:return 134217728;case 268435456:return 268435456;case 536870912:return 536870912;case 1073741824:return 0;default:return e}}function ys(e,n,a){var s=e.pendingLanes;if(s===0)return 0;var l=0,u=e.suspendedLanes,p=e.pingedLanes;e=e.warmLanes;var A=s&134217727;return A!==0?(s=A&~u,s!==0?l=fa(s):(p&=A,p!==0?l=fa(p):a||(a=A&~e,a!==0&&(l=fa(a))))):(A=s&~u,A!==0?l=fa(A):p!==0?l=fa(p):a||(a=s&~e,a!==0&&(l=fa(a)))),l===0?0:n!==0&&n!==l&&(n&u)===0&&(u=l&-l,a=n&-n,u>=a||u===32&&(a&4194048)!==0)?n:l}function za(e,n){return(e.pendingLanes&~(e.suspendedLanes&~e.pingedLanes)&n)===0}function zi(e,n){(n&8)!==0&&(n|=n&32);var a=e.entangledLanes;if(a!==0)for(e=e.entanglements,a&=n;0<a;){var s=31-ct(a),l=1<<s;n|=e[s],a&=~l}return n}function co(e,n){switch(e){case 1:case 2:case 4:case 8:case 64:return n+250;case 16:case 32:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return n+5e3;case 4194304:case 8388608:case 16777216:case 33554432:return-1;case 67108864:case 134217728:case 268435456:case 536870912:case 1073741824:return-1;default:return-1}}function uo(){var e=Ba;return Ba<<=1,(Ba&62914560)===0&&(Ba=4194304),e}function tr(e){for(var n=[],a=0;31>a;a++)n.push(e);return n}function Fi(e,n){e.pendingLanes|=n,n!==268435456&&(e.suspendedLanes=0,e.pingedLanes=0,e.warmLanes=0)}function Rl(e,n,a,s,l,u){var p=e.pendingLanes;e.pendingLanes=a,e.suspendedLanes=0,e.pingedLanes=0,e.warmLanes=0,e.expiredLanes&=a,e.entangledLanes&=a,e.errorRecoveryDisabledLanes&=a,e.shellSuspendCounter=0;var A=e.entanglements,B=e.expirationTimes,te=e.hiddenUpdates;for(a=p&~a;0<a;){var fe=31-ct(a),ye=1<<fe;A[fe]=0,B[fe]=-1;var J=te[fe];if(J!==null)for(te[fe]=null,fe=0;fe<J.length;fe++){var oe=J[fe];oe!==null&&(oe.lane&=-536870913)}a&=~ye}s!==0&&Ms(e,s,0),u!==0&&l===0&&e.tag!==0&&(e.suspendedLanes|=u&~(p&~n))}function Ms(e,n,a){e.pendingLanes|=n,e.suspendedLanes&=~n;var s=31-ct(n);e.entangledLanes|=n,e.entanglements[s]=e.entanglements[s]|1073741824|a&261930}function fo(e,n){var a=e.entangledLanes|=n;for(e=e.entanglements;a;){var s=31-ct(a),l=1<<s;l&n|e[s]&n&&(e[s]|=n),a&=~l}}function ho(e,n){var a=n&-n;return a=(a&42)!==0?1:po(a),(a&(e.suspendedLanes|n))!==0?0:a}function po(e){switch(e){case 2:e=1;break;case 8:e=4;break;case 32:e=16;break;case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:case 4194304:case 8388608:case 16777216:case 33554432:e=128;break;case 268435456:e=134217728;break;default:e=0}return e}function mo(e){return e&=-e,2<e?8<e?(e&134217727)!==0?32:268435456:8:2}function wl(){var e=we.p;return e!==0?e:(e=window.event,e===void 0?32:$_(e.type))}function Cl(e,n){var a=we.p;try{return we.p=e,n()}finally{we.p=a}}var hi=Math.random().toString(36).slice(2),T="__reactFiber$"+hi,G="__reactProps$"+hi,de="__reactContainer$"+hi,se="__reactEvents$"+hi,re="__reactListeners$"+hi,ze="__reactHandles$"+hi,Xe="__reactResources$"+hi,Le="__reactMarker$"+hi,Ye="__reactLoad$"+hi;function Ke(e){delete e[T],delete e[G],delete e[re],delete e[ze]}function st(e){var n;if(n=e[T])return n;for(var a=e.parentNode;a;){if(n=a[de]||a[T]){if(a=n.alternate,n.child!==null||a!==null&&a.child!==null)for(e=L_(e);e!==null;){if(a=e[T])return a;e=L_(e)}return n}e=a,a=e.parentNode}return null}function ut(e){if(e=e[T]||e[de]){var n=e.tag;if(n===5||n===6||n===13||n===31||n===26||n===27||n===3)return e}return null}function We(e){var n=e.tag;if(n===5||n===26||n===27||n===6)return e.stateNode;throw Error(r(33))}function yt(e){var n=e[Xe];return n||(n=e[Xe]={hoistableStyles:new Map,hoistableScripts:new Map}),n}function _t(e){e[Le]=!0}function jt(e){e[Ye]=void 0}var Ft=new Set,vn={};function Fe(e,n){rn(e,n),rn(e+"Capture",n)}function rn(e,n){for(vn[e]=n,e=0;e<n.length;e++)Ft.add(n[e])}var wt=RegExp("^[:A-Z_a-z\\u00C0-\\u00D6\\u00D8-\\u00F6\\u00F8-\\u02FF\\u0370-\\u037D\\u037F-\\u1FFF\\u200C-\\u200D\\u2070-\\u218F\\u2C00-\\u2FEF\\u3001-\\uD7FF\\uF900-\\uFDCF\\uFDF0-\\uFFFD][:A-Z_a-z\\u00C0-\\u00D6\\u00D8-\\u00F6\\u00F8-\\u02FF\\u0370-\\u037D\\u037F-\\u1FFF\\u200C-\\u200D\\u2070-\\u218F\\u2C00-\\u2FEF\\u3001-\\uD7FF\\uF900-\\uFDCF\\uFDF0-\\uFFFD\\-.0-9\\u00B7\\u0300-\\u036F\\u203F-\\u2040]*$"),Bn={},$n={};function Hi(e){return Ee.call($n,e)?!0:Ee.call(Bn,e)?!1:wt.test(e)?$n[e]=!0:(Bn[e]=!0,!1)}var vt=!1;function Bt(){var e=vt;return vt=!1,e}function Qt(e,n,a){if(Hi(n))if(a===null)e.removeAttribute(n);else{switch(typeof a){case"undefined":case"function":case"symbol":e.removeAttribute(n);return;case"boolean":var s=n.toLowerCase().slice(0,5);if(s!=="data-"&&s!=="aria-"){e.removeAttribute(n);return}}e.setAttribute(n,a)}}function ei(e,n,a){if(a===null)e.removeAttribute(n);else{switch(typeof a){case"undefined":case"function":case"symbol":case"boolean":e.removeAttribute(n);return}e.setAttribute(n,a)}}function bt(e,n,a,s){if(s===null)e.removeAttribute(a);else{switch(typeof s){case"undefined":case"function":case"symbol":case"boolean":e.removeAttribute(a);return}e.setAttributeNS(n,a,s)}}function on(e){switch(typeof e){case"bigint":case"boolean":case"number":case"string":case"undefined":return e;case"object":return e;default:return""}}function da(e){var n=e.type;return(e=e.nodeName)&&e.toLowerCase()==="input"&&(n==="checkbox"||n==="radio")}function Nl(e,n,a){var s=Object.getOwnPropertyDescriptor(e.constructor.prototype,n);if(!e.hasOwnProperty(n)&&typeof s<"u"&&typeof s.get=="function"&&typeof s.set=="function"){var l=s.get,u=s.set;return Object.defineProperty(e,n,{configurable:!0,get:function(){return l.call(this)},set:function(p){a=""+p,u.call(this,p)}}),Object.defineProperty(e,n,{enumerable:s.enumerable}),{getValue:function(){return a},setValue:function(p){a=""+p},stopTracking:function(){e._valueTracker=null,delete e[n]}}}}function Iu(e){if(!e._valueTracker){var n=da(e)?"checked":"value";e._valueTracker=Nl(e,n,""+e[n])}}function im(e){if(!e)return!1;var n=e._valueTracker;if(!n)return!0;var a=n.getValue(),s="";return e&&(s=da(e)?e.checked?"true":"false":e.value),e=s,e!==a?(n.setValue(e),!0):!1}var aS=/[\n"\\]/g;function pi(e){return e.replace(aS,function(n){return"\\"+n.charCodeAt(0).toString(16)+" "})}function Bu(e,n,a,s,l,u,p,A){e.name="",p!=null&&typeof p!="function"&&typeof p!="symbol"&&typeof p!="boolean"?e.type=p:e.removeAttribute("type"),n!=null?p==="number"?(n===0&&e.value===""||e.value!=n)&&(e.value=""+on(n)):e.value!==""+on(n)&&(e.value=""+on(n)):p!=="submit"&&p!=="reset"||e.removeAttribute("value"),n!=null?p==="number"&&e.value==n?zu(e,on(e.value)):zu(e,on(n)):a!=null?zu(e,on(a)):s!=null&&e.removeAttribute("value"),l==null&&u!=null&&(e.defaultChecked=!!u),l!=null&&(e.checked=l&&typeof l!="function"&&typeof l!="symbol"),A!=null&&typeof A!="function"&&typeof A!="symbol"&&typeof A!="boolean"?e.name=""+on(A):e.removeAttribute("name")}function am(e,n,a,s,l,u,p,A){if(u!=null&&typeof u!="function"&&typeof u!="symbol"&&typeof u!="boolean"&&(e.type=u),n!=null||a!=null){if(!(u!=="submit"&&u!=="reset"||n!=null)){Iu(e);return}a=a!=null?""+on(a):"",n=n!=null?""+on(n):a,A||n===e.value||(e.value=n),e.defaultValue=n}s=s??l,s=typeof s!="function"&&typeof s!="symbol"&&!!s,e.checked=A?e.checked:!!s,e.defaultChecked=!!s,p!=null&&typeof p!="function"&&typeof p!="symbol"&&typeof p!="boolean"&&(e.name=p),Iu(e)}function zu(e,n){e.defaultValue!==""+n&&(e.defaultValue=""+n)}function nr(e,n,a,s){if(e=e.options,n){n={};for(var l=0;l<a.length;l++)n["$"+a[l]]=!0;for(a=0;a<e.length;a++)l=n.hasOwnProperty("$"+e[a].value),e[a].selected!==l&&(e[a].selected=l),l&&s&&(e[a].defaultSelected=!0)}else{for(a=""+on(a),n=null,l=0;l<e.length;l++){if(e[l].value===a){e[l].selected=!0,s&&(e[l].defaultSelected=!0);return}n!==null||e[l].disabled||(n=e[l])}n!==null&&(n.selected=!0)}}function sm(e,n,a){if(n!=null&&(n=""+on(n),n!==e.value&&(e.value=n),a==null)){e.defaultValue!==n&&(e.defaultValue=n);return}e.defaultValue=a!=null?""+on(a):""}function rm(e,n,a,s){if(n==null){if(s!=null){if(a!=null)throw Error(r(92));if(Ie(s)){if(1<s.length)throw Error(r(93));s=s[0]}a=s}a==null&&(a=""),n=a}a=on(n),e.defaultValue=a,s=e.textContent,s===a&&s!==""&&s!==null&&(e.value=s),Iu(e)}function ir(e,n){if(n){var a=e.firstChild;if(a&&a===e.lastChild&&a.nodeType===3){a.nodeValue=n;return}}e.textContent=n}var sS=new Set("animationIterationCount aspectRatio borderImageOutset borderImageSlice borderImageWidth boxFlex boxFlexGroup boxOrdinalGroup columnCount columns flex flexGrow flexPositive flexShrink flexNegative flexOrder gridArea gridRow gridRowEnd gridRowSpan gridRowStart gridColumn gridColumnEnd gridColumnSpan gridColumnStart fontWeight lineClamp lineHeight opacity order orphans scale tabSize widows zIndex zoom fillOpacity floodOpacity stopOpacity strokeDasharray strokeDashoffset strokeMiterlimit strokeOpacity strokeWidth MozAnimationIterationCount MozBoxFlex MozBoxFlexGroup MozLineClamp msAnimationIterationCount msFlex msZoom msFlexGrow msFlexNegative msFlexOrder msFlexPositive msFlexShrink msGridColumn msGridColumnSpan msGridRow msGridRowSpan WebkitAnimationIterationCount WebkitBoxFlex WebKitBoxFlexGroup WebkitBoxOrdinalGroup WebkitColumnCount WebkitColumns WebkitFlex WebkitFlexGrow WebkitFlexPositive WebkitFlexShrink WebkitLineClamp".split(" "));function om(e,n,a){var s=n.indexOf("--")===0;a==null||typeof a=="boolean"||a===""?s?e.setProperty(n,""):n==="float"?e.cssFloat="":e[n]="":s?e.setProperty(n,a):typeof a!="number"||a===0||sS.has(n)?n==="float"?e.cssFloat=a:e[n]=(""+a).trim():e[n]=a+"px"}function lm(e,n,a){if(n!=null&&typeof n!="object")throw Error(r(62));if(e=e.style,a!=null){for(var s in a)!a.hasOwnProperty(s)||n!=null&&n.hasOwnProperty(s)||(s.indexOf("--")===0?e.setProperty(s,""):s==="float"?e.cssFloat="":e[s]="",vt=!0);for(var l in n)s=n[l],n.hasOwnProperty(l)&&a[l]!==s&&(om(e,l,s),vt=!0)}else for(var u in n)n.hasOwnProperty(u)&&om(e,u,n[u])}function Fu(e){if(e.indexOf("-")===-1)return!1;switch(e){case"annotation-xml":case"color-profile":case"font-face":case"font-face-src":case"font-face-uri":case"font-face-format":case"font-face-name":case"missing-glyph":return!1;default:return!0}}var rS=new Map([["acceptCharset","accept-charset"],["htmlFor","for"],["httpEquiv","http-equiv"],["crossOrigin","crossorigin"],["accentHeight","accent-height"],["alignmentBaseline","alignment-baseline"],["arabicForm","arabic-form"],["baselineShift","baseline-shift"],["capHeight","cap-height"],["clipPath","clip-path"],["clipRule","clip-rule"],["colorInterpolation","color-interpolation"],["colorInterpolationFilters","color-interpolation-filters"],["colorProfile","color-profile"],["colorRendering","color-rendering"],["dominantBaseline","dominant-baseline"],["enableBackground","enable-background"],["fillOpacity","fill-opacity"],["fillRule","fill-rule"],["floodColor","flood-color"],["floodOpacity","flood-opacity"],["fontFamily","font-family"],["fontSize","font-size"],["fontSizeAdjust","font-size-adjust"],["fontStretch","font-stretch"],["fontStyle","font-style"],["fontVariant","font-variant"],["fontWeight","font-weight"],["glyphName","glyph-name"],["glyphOrientationHorizontal","glyph-orientation-horizontal"],["glyphOrientationVertical","glyph-orientation-vertical"],["horizAdvX","horiz-adv-x"],["horizOriginX","horiz-origin-x"],["imageRendering","image-rendering"],["letterSpacing","letter-spacing"],["lightingColor","lighting-color"],["markerEnd","marker-end"],["markerMid","marker-mid"],["markerStart","marker-start"],["maskType","mask-type"],["overlinePosition","overline-position"],["overlineThickness","overline-thickness"],["paintOrder","paint-order"],["panose-1","panose-1"],["pointerEvents","pointer-events"],["renderingIntent","rendering-intent"],["shapeRendering","shape-rendering"],["stopColor","stop-color"],["stopOpacity","stop-opacity"],["strikethroughPosition","strikethrough-position"],["strikethroughThickness","strikethrough-thickness"],["strokeDasharray","stroke-dasharray"],["strokeDashoffset","stroke-dashoffset"],["strokeLinecap","stroke-linecap"],["strokeLinejoin","stroke-linejoin"],["strokeMiterlimit","stroke-miterlimit"],["strokeOpacity","stroke-opacity"],["strokeWidth","stroke-width"],["textAnchor","text-anchor"],["textDecoration","text-decoration"],["textRendering","text-rendering"],["transformOrigin","transform-origin"],["underlinePosition","underline-position"],["underlineThickness","underline-thickness"],["unicodeBidi","unicode-bidi"],["unicodeRange","unicode-range"],["unitsPerEm","units-per-em"],["vAlphabetic","v-alphabetic"],["vHanging","v-hanging"],["vIdeographic","v-ideographic"],["vMathematical","v-mathematical"],["vectorEffect","vector-effect"],["vertAdvY","vert-adv-y"],["vertOriginX","vert-origin-x"],["vertOriginY","vert-origin-y"],["wordSpacing","word-spacing"],["writingMode","writing-mode"],["xmlnsXlink","xmlns:xlink"],["xHeight","x-height"]]),oS=/^[\u0000-\u001F ]*j[\r\n\t]*a[\r\n\t]*v[\r\n\t]*a[\r\n\t]*s[\r\n\t]*c[\r\n\t]*r[\r\n\t]*i[\r\n\t]*p[\r\n\t]*t[\r\n\t]*:/i;function Dl(e){return oS.test(""+e)?"javascript:throw new Error('React has blocked a javascript: URL as a security precaution.')":e}function Gi(){}var Hu=null;function Gu(e){return e=e.target||e.srcElement||window,e.correspondingUseElement&&(e=e.correspondingUseElement),e.nodeType===3?e.parentNode:e}var ar=null,sr=null;function cm(e){var n=ut(e);if(n&&(e=n.stateNode)){var a=e[G]||null;e:switch(e=n.stateNode,n.type){case"input":if(Bu(e,a.value,a.defaultValue,a.defaultValue,a.checked,a.defaultChecked,a.type,a.name),n=a.name,a.type==="radio"&&n!=null){for(a=e;a.parentNode;)a=a.parentNode;for(a=a.querySelectorAll('input[name="'+pi(""+n)+'"][type="radio"]'),n=0;n<a.length;n++){var s=a[n];if(s!==e&&s.form===e.form){var l=s[G]||null;if(!l)throw Error(r(90));Bu(s,l.value,l.defaultValue,l.defaultValue,l.checked,l.defaultChecked,l.type,l.name)}}for(n=0;n<a.length;n++)s=a[n],s.form===e.form&&im(s)}break e;case"textarea":sm(e,a.value,a.defaultValue);break e;case"select":n=a.value,n!=null&&nr(e,!!a.multiple,n,!1)}}}var Vu=!1;function um(e,n,a){if(Vu)return e(n,a);Vu=!0;try{var s=e(n);return s}finally{if(Vu=!1,(ar!==null||sr!==null)&&(Dc(),ar&&(n=ar,e=sr,sr=ar=null,cm(n),e)))for(n=0;n<e.length;n++)cm(e[n])}}function go(e,n){var a=e.stateNode;if(a===null)return null;var s=a[G]||null;if(s===null)return null;a=s[n];e:switch(n){case"onClick":case"onClickCapture":case"onDoubleClick":case"onDoubleClickCapture":case"onMouseDown":case"onMouseDownCapture":case"onMouseMove":case"onMouseMoveCapture":case"onMouseUp":case"onMouseUpCapture":case"onMouseEnter":(s=!s.disabled)||(e=e.type,s=!(e==="button"||e==="input"||e==="select"||e==="textarea")),e=!s;break e;default:e=!1}if(e)return null;if(a&&typeof a!="function")throw Error(r(231,n,typeof a));return a}var ha=!(typeof window>"u"||typeof window.document>"u"||typeof window.document.createElement>"u"),ku=!1;if(ha)try{var _o={};Object.defineProperty(_o,"passive",{get:function(){ku=!0}}),window.addEventListener("test",_o,_o),window.removeEventListener("test",_o,_o)}catch{ku=!1}var Fa=null,Xu=null,Ul=null;function fm(){if(Ul)return Ul;var e,n=Xu,a=n.length,s,l="value"in Fa?Fa.value:Fa.textContent,u=l.length;for(e=0;e<a&&n[e]===l[e];e++);var p=a-e;for(s=1;s<=p&&n[a-s]===l[u-s];s++);return Ul=l.slice(e,1<s?1-s:void 0)}function Ll(e){var n=e.keyCode;return"charCode"in e?(e=e.charCode,e===0&&n===13&&(e=13)):e=n,e===10&&(e=13),32<=e||e===13?e:0}function Ol(){return!0}function dm(){return!1}function zn(e){function n(a,s,l,u,p){this._reactName=a,this._targetInst=l,this.type=s,this.nativeEvent=u,this.target=p,this.currentTarget=null;for(var A in e)e.hasOwnProperty(A)&&(a=e[A],this[A]=a?a(u):u[A]);return this.isDefaultPrevented=(u.defaultPrevented!=null?u.defaultPrevented:u.returnValue===!1)?Ol:dm,this.isPropagationStopped=dm,this}return I(n.prototype,{preventDefault:function(){this.defaultPrevented=!0;var a=this.nativeEvent;a&&(a.preventDefault?a.preventDefault():typeof a.returnValue!="unknown"&&(a.returnValue=!1),this.isDefaultPrevented=Ol)},stopPropagation:function(){var a=this.nativeEvent;a&&(a.stopPropagation?a.stopPropagation():typeof a.cancelBubble!="unknown"&&(a.cancelBubble=!0),this.isPropagationStopped=Ol)},persist:function(){},isPersistent:Ol}),n}var Ha={eventPhase:0,bubbles:0,cancelable:0,timeStamp:function(e){return e.timeStamp||Date.now()},defaultPrevented:0,isTrusted:0},Pl=zn(Ha),vo=I({},Ha,{view:0,detail:0}),lS=zn(vo),Wu,qu,xo,Il=I({},vo,{screenX:0,screenY:0,clientX:0,clientY:0,pageX:0,pageY:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,getModifierState:ju,button:0,buttons:0,relatedTarget:function(e){return e.relatedTarget===void 0?e.fromElement===e.srcElement?e.toElement:e.fromElement:e.relatedTarget},movementX:function(e){return"movementX"in e?e.movementX:(e!==xo&&(xo&&e.type==="mousemove"?(Wu=e.screenX-xo.screenX,qu=e.screenY-xo.screenY):qu=Wu=0,xo=e),Wu)},movementY:function(e){return"movementY"in e?e.movementY:qu}}),hm=zn(Il),cS=I({},Il,{dataTransfer:0}),uS=zn(cS),fS=I({},vo,{relatedTarget:0}),Yu=zn(fS),dS=I({},Ha,{animationName:0,elapsedTime:0,pseudoElement:0}),hS=zn(dS),pS=I({},Ha,{clipboardData:function(e){return"clipboardData"in e?e.clipboardData:window.clipboardData}}),mS=zn(pS),gS=I({},Ha,{data:0}),pm=zn(gS),_S={Esc:"Escape",Spacebar:" ",Left:"ArrowLeft",Up:"ArrowUp",Right:"ArrowRight",Down:"ArrowDown",Del:"Delete",Win:"OS",Menu:"ContextMenu",Apps:"ContextMenu",Scroll:"ScrollLock",MozPrintableKey:"Unidentified"},vS={8:"Backspace",9:"Tab",12:"Clear",13:"Enter",16:"Shift",17:"Control",18:"Alt",19:"Pause",20:"CapsLock",27:"Escape",32:" ",33:"PageUp",34:"PageDown",35:"End",36:"Home",37:"ArrowLeft",38:"ArrowUp",39:"ArrowRight",40:"ArrowDown",45:"Insert",46:"Delete",112:"F1",113:"F2",114:"F3",115:"F4",116:"F5",117:"F6",118:"F7",119:"F8",120:"F9",121:"F10",122:"F11",123:"F12",144:"NumLock",145:"ScrollLock",224:"Meta"},xS={Alt:"altKey",Control:"ctrlKey",Meta:"metaKey",Shift:"shiftKey"};function SS(e){var n=this.nativeEvent;return n.getModifierState?n.getModifierState(e):(e=xS[e])?!!n[e]:!1}function ju(){return SS}var yS=I({},vo,{key:function(e){if(e.key){var n=_S[e.key]||e.key;if(n!=="Unidentified")return n}return e.type==="keypress"?(e=Ll(e),e===13?"Enter":String.fromCharCode(e)):e.type==="keydown"||e.type==="keyup"?vS[e.keyCode]||"Unidentified":""},code:0,location:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,repeat:0,locale:0,getModifierState:ju,charCode:function(e){return e.type==="keypress"?Ll(e):0},keyCode:function(e){return e.type==="keydown"||e.type==="keyup"?e.keyCode:0},which:function(e){return e.type==="keypress"?Ll(e):e.type==="keydown"||e.type==="keyup"?e.keyCode:0}}),MS=zn(yS),ES=I({},Il,{pointerId:0,width:0,height:0,pressure:0,tangentialPressure:0,tiltX:0,tiltY:0,twist:0,pointerType:0,isPrimary:0}),mm=zn(ES),bS=I({},Ha,{submitter:0}),TS=zn(bS),AS=I({},vo,{touches:0,targetTouches:0,changedTouches:0,altKey:0,metaKey:0,ctrlKey:0,shiftKey:0,getModifierState:ju}),RS=zn(AS),wS=I({},Ha,{propertyName:0,elapsedTime:0,pseudoElement:0}),CS=zn(wS),NS=I({},Il,{deltaX:function(e){return"deltaX"in e?e.deltaX:"wheelDeltaX"in e?-e.wheelDeltaX:0},deltaY:function(e){return"deltaY"in e?e.deltaY:"wheelDeltaY"in e?-e.wheelDeltaY:"wheelDelta"in e?-e.wheelDelta:0},deltaZ:0,deltaMode:0}),DS=zn(NS),US=I({},Ha,{newState:0,oldState:0,source:0}),LS=zn(US),OS=[9,13,27,32],Zu=ha&&"CompositionEvent"in window,So=null;ha&&"documentMode"in document&&(So=document.documentMode);var PS=ha&&"TextEvent"in window&&!So,gm=ha&&(!Zu||So&&8<So&&11>=So),_m=" ",vm=!1;function xm(e,n){switch(e){case"keyup":return OS.indexOf(n.keyCode)!==-1;case"keydown":return n.keyCode!==229;case"keypress":case"mousedown":case"focusout":return!0;default:return!1}}function Sm(e){return e=e.detail,typeof e=="object"&&"data"in e?e.data:null}var rr=!1;function IS(e,n){switch(e){case"compositionend":return Sm(n);case"keypress":return n.which!==32?null:(vm=!0,_m);case"textInput":return e=n.data,e===_m&&vm?null:e;default:return null}}function BS(e,n){if(rr)return e==="compositionend"||!Zu&&xm(e,n)?(e=fm(),Ul=Xu=Fa=null,rr=!1,e):null;switch(e){case"paste":return null;case"keypress":if(!(n.ctrlKey||n.altKey||n.metaKey)||n.ctrlKey&&n.altKey){if(n.char&&1<n.char.length)return n.char;if(n.which)return String.fromCharCode(n.which)}return null;case"compositionend":return gm&&n.locale!=="ko"?null:n.data;default:return null}}var zS={color:!0,date:!0,datetime:!0,"datetime-local":!0,email:!0,month:!0,number:!0,password:!0,range:!0,search:!0,tel:!0,text:!0,time:!0,url:!0,week:!0};function ym(e){var n=e&&e.nodeName&&e.nodeName.toLowerCase();return n==="input"?!!zS[e.type]:n==="textarea"}function Mm(e,n,a,s){ar?sr?sr.push(s):sr=[s]:ar=s,n=Bc(n,"onChange"),0<n.length&&(a=new Pl("onChange","change",null,a,s),e.push({event:a,listeners:n}))}var yo=null,Mo=null;function FS(e){c_(e,0)}function Bl(e){var n=We(e);if(im(n))return e}function Em(e,n){if(e==="change")return n}var bm=!1;if(ha){var Ku;if(ha){var Qu="oninput"in document;if(!Qu){var Tm=document.createElement("div");Tm.setAttribute("oninput","return;"),Qu=typeof Tm.oninput=="function"}Ku=Qu}else Ku=!1;bm=Ku&&(!document.documentMode||9<document.documentMode)}function Am(){yo&&(yo.detachEvent("onpropertychange",Rm),Mo=yo=null)}function Rm(e){if(e.propertyName==="value"&&Bl(Mo)){var n=[];Mm(n,Mo,e,Gu(e)),um(FS,n)}}function HS(e,n,a){e==="focusin"?(Am(),yo=n,Mo=a,yo.attachEvent("onpropertychange",Rm)):e==="focusout"&&Am()}function GS(e){if(e==="selectionchange"||e==="keyup"||e==="keydown")return Bl(Mo)}function VS(e,n){if(e==="click")return Bl(n)}function kS(e,n){if(e==="input"||e==="change")return Bl(n)}function XS(e,n){return e===n&&(e!==0||1/e===1/n)||e!==e&&n!==n}var ti=typeof Object.is=="function"?Object.is:XS;function Eo(e,n){if(ti(e,n))return!0;if(typeof e!="object"||e===null||typeof n!="object"||n===null)return!1;var a=Object.keys(e),s=Object.keys(n);if(a.length!==s.length)return!1;for(s=0;s<a.length;s++){var l=a[s];if(!Ee.call(n,l)||!ti(e[l],n[l]))return!1}return!0}function Ju(e){if(e=e||(typeof document<"u"?document:void 0),typeof e>"u")return null;try{return e.activeElement||e.body}catch{return e.body}}function wm(e){for(;e&&e.firstChild;)e=e.firstChild;return e}function Cm(e,n){var a=wm(e);e=0;for(var s;a;){if(a.nodeType===3){if(s=e+a.textContent.length,e<=n&&s>=n)return{node:a,offset:n-e};e=s}e:{for(;a;){if(a.nextSibling){a=a.nextSibling;break e}a=a.parentNode}a=void 0}a=wm(a)}}function Nm(e,n){return e&&n?e===n?!0:e&&e.nodeType===3?!1:n&&n.nodeType===3?Nm(e,n.parentNode):"contains"in e?e.contains(n):e.compareDocumentPosition?!!(e.compareDocumentPosition(n)&16):!1:!1}function Dm(e){e=e!=null&&e.ownerDocument!=null&&e.ownerDocument.defaultView!=null?e.ownerDocument.defaultView:window;for(var n=Ju(e.document);n instanceof e.HTMLIFrameElement;){try{var a=typeof n.contentWindow.location.href=="string"}catch{a=!1}if(a)e=n.contentWindow;else break;n=Ju(e.document)}return n}function $u(e){var n=e&&e.nodeName&&e.nodeName.toLowerCase();return n&&(n==="input"&&(e.type==="text"||e.type==="search"||e.type==="tel"||e.type==="url"||e.type==="password")||n==="textarea"||e.contentEditable==="true")}var WS=ha&&"documentMode"in document&&11>=document.documentMode,or=null,ef=null,bo=null,tf=!1;function Um(e,n,a){var s=a.window===a?a.document:a.nodeType===9?a:a.ownerDocument;tf||or==null||or!==Ju(s)||(s=or,"selectionStart"in s&&$u(s)?s={start:s.selectionStart,end:s.selectionEnd}:(s=(s.ownerDocument&&s.ownerDocument.defaultView||window).getSelection(),s={anchorNode:s.anchorNode,anchorOffset:s.anchorOffset,focusNode:s.focusNode,focusOffset:s.focusOffset}),bo&&Eo(bo,s)||(bo=s,s=Bc(ef,"onSelect"),0<s.length&&(n=new Pl("onSelect","select",null,n,a),e.push({event:n,listeners:s}),n.target=or)))}function Es(e,n){var a={};return a[e.toLowerCase()]=n.toLowerCase(),a["Webkit"+e]="webkit"+n,a["Moz"+e]="moz"+n,a}var lr={animationend:Es("Animation","AnimationEnd"),animationiteration:Es("Animation","AnimationIteration"),animationstart:Es("Animation","AnimationStart"),transitionrun:Es("Transition","TransitionRun"),transitionstart:Es("Transition","TransitionStart"),transitioncancel:Es("Transition","TransitionCancel"),transitionend:Es("Transition","TransitionEnd")},nf={},Lm={};ha&&(Lm=document.createElement("div").style,"AnimationEvent"in window||(delete lr.animationend.animation,delete lr.animationiteration.animation,delete lr.animationstart.animation),"TransitionEvent"in window||delete lr.transitionend.transition);function bs(e){if(nf[e])return nf[e];if(!lr[e])return e;var n=lr[e],a;for(a in n)if(n.hasOwnProperty(a)&&a in Lm)return nf[e]=n[a];return e}var Om=bs("animationend"),Pm=bs("animationiteration"),Im=bs("animationstart"),qS=bs("transitionrun"),YS=bs("transitionstart"),jS=bs("transitioncancel"),Bm=bs("transitionend"),zm=new Map,af="abort auxClick beforeToggle cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error fullscreenChange fullscreenError gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel".split(" ");af.push("scrollEnd");function Ai(e,n){zm.set(e,n),Fe(n,[e])}var ZS=0;function pa(e,n){if(e.name!=null&&e.name!=="auto")return e.name;if(n.autoName!==null)return n.autoName;e=Ni.identifierPrefix;var a=ZS++;return e="_"+e+"t_"+a.toString(32)+"_",n.autoName=e}function Fm(e){if(e==null||typeof e=="string")return e;var n=null,a=wr;if(a!==null)for(var s=0;s<a.length;s++){var l=e[a[s]];if(l!=null){if(l==="none")return"none";n=n==null?l:n+(" "+l)}}return n??e.default}function ma(e,n){return e=Fm(e),n=Fm(n),n==null?e==="auto"?null:e:n==="auto"?null:n}var zl=typeof reportError=="function"?reportError:function(e){if(typeof window=="object"&&typeof window.ErrorEvent=="function"){var n=new window.ErrorEvent("error",{bubbles:!0,cancelable:!0,message:typeof e=="object"&&e!==null&&typeof e.message=="string"?String(e.message):String(e),error:e});if(!window.dispatchEvent(n))return}else if(typeof process=="object"&&typeof process.emit=="function"){process.emit("uncaughtException",e);return}console.error(e)},mi=[],cr=0,sf=0;function Fl(){for(var e=cr,n=sf=cr=0;n<e;){var a=mi[n];mi[n++]=null;var s=mi[n];mi[n++]=null;var l=mi[n];mi[n++]=null;var u=mi[n];if(mi[n++]=null,s!==null&&l!==null){var p=s.pending;p===null?l.next=l:(l.next=p.next,p.next=l),s.pending=l}u!==0&&Hm(a,l,u)}}function Hl(e,n,a,s){mi[cr++]=e,mi[cr++]=n,mi[cr++]=a,mi[cr++]=s,sf|=s,e.lanes|=s,e=e.alternate,e!==null&&(e.lanes|=s)}function rf(e,n,a,s){return Hl(e,n,a,s),Gl(e)}function Ts(e,n){return Hl(e,null,null,n),Gl(e)}function Hm(e,n,a){e.lanes|=a;var s=e.alternate;s!==null&&(s.lanes|=a);for(var l=!1,u=e.return;u!==null;)u.childLanes|=a,s=u.alternate,s!==null&&(s.childLanes|=a),u.tag===22&&(e=u.stateNode,e===null||e._visibility&1||(l=!0)),e=u,u=u.return;return e.tag===3?(u=e.stateNode,l&&n!==null&&(l=31-ct(a),e=u.hiddenUpdates,s=e[l],s===null?e[l]=[n]:s.push(n),n.lane=a|536870912),u):null}function Gl(e){if(50<qo)throw qo=0,Nc=null,Error(r(185));for(var n=e.return;n!==null;)e=n,n=e.return;return e.tag===3?e.stateNode:null}var ur={};function KS(e,n,a,s){this.tag=e,this.key=a,this.sibling=this.child=this.return=this.stateNode=this.type=this.elementType=null,this.index=0,this.refCleanup=this.ref=null,this.pendingProps=n,this.dependencies=this.memoizedState=this.updateQueue=this.memoizedProps=null,this.mode=s,this.subtreeFlags=this.flags=0,this.deletions=null,this.childLanes=this.lanes=0,this.alternate=null}function Xn(e,n,a,s){return new KS(e,n,a,s)}function of(e){return e=e.prototype,!(!e||!e.isReactComponent)}function ga(e,n){var a=e.alternate;return a===null?(a=Xn(e.tag,n,e.key,e.mode),a.elementType=e.elementType,a.type=e.type,a.stateNode=e.stateNode,a.alternate=e,e.alternate=a):(a.pendingProps=n,a.type=e.type,a.flags=0,a.subtreeFlags=0,a.deletions=null),a.flags=e.flags&1206910976,a.childLanes=e.childLanes,a.lanes=e.lanes,a.child=e.child,a.memoizedProps=e.memoizedProps,a.memoizedState=e.memoizedState,a.updateQueue=e.updateQueue,n=e.dependencies,a.dependencies=n===null?null:{lanes:n.lanes,firstContext:n.firstContext},a.sibling=e.sibling,a.index=e.index,a.ref=e.ref,a.refCleanup=e.refCleanup,a}function Gm(e,n){e.flags&=1206910978;var a=e.alternate;return a===null?(e.childLanes=0,e.lanes=n,e.child=null,e.subtreeFlags=0,e.memoizedProps=null,e.memoizedState=null,e.updateQueue=null,e.dependencies=null,e.stateNode=null):(e.childLanes=a.childLanes,e.lanes=a.lanes,e.child=a.child,e.subtreeFlags=0,e.deletions=null,e.memoizedProps=a.memoizedProps,e.memoizedState=a.memoizedState,e.updateQueue=a.updateQueue,e.type=a.type,n=a.dependencies,e.dependencies=n===null?null:{lanes:n.lanes,firstContext:n.firstContext}),e}function Vl(e,n,a,s,l,u){var p=0;if(s=e,typeof s=="function")of(s)&&(p=1);else if(typeof s=="string")p=bM(e,a,kt.current)?26:e==="html"||e==="head"||e==="body"?27:5;else e:switch(s){case be:return e=Xn(31,a,n,l),e.elementType=be,e.lanes=u,e;case Y:return As(a.children,l,u,n);case $:p=8,l|=24;break;case O:return e=Xn(12,a,n,l|2),e.elementType=O,e.lanes=u,e;case Q:return e=Xn(13,a,n,l),e.elementType=Q,e.lanes=u,e;case pe:return e=Xn(19,a,n,l),e.elementType=pe,e.lanes=u,e;case Ze:case H:return e=l|32,e=Xn(30,a,n,e),e.elementType=H,e.lanes=u,e.stateNode={autoName:null,paired:null,clones:null,ref:null},e;default:if(typeof s=="object"&&s!==null)switch(s.$$typeof){case ne:p=10;break e;case q:p=9;break e;case j:p=11;break e;case ce:p=14;break e;case me:p=16,s=null;break e}p=29,a=Error(r(130,e===null?"null":typeof e,"")),s=null}return n=Xn(p,a,n,l),n.elementType=e,n.type=s,n.lanes=u,n}function As(e,n,a,s){return e=Xn(7,e,s,n),e.lanes=a,e}function lf(e,n,a){return e=Xn(6,e,null,n),e.lanes=a,e}function Vm(e){var n=Xn(18,null,null,0);return n.stateNode=e,n}function cf(e,n,a){return n=Xn(4,e.children!==null?e.children:[],e.key,n),n.lanes=a,n.stateNode={containerInfo:e.containerInfo,pendingChildren:null,implementation:e.implementation},n}var km=new WeakMap;function gi(e,n){if(typeof e=="object"&&e!==null){var a=km.get(e);return a!==void 0?a:(n={value:e,source:n,stack:ve(n)},km.set(e,n),n)}return{value:e,source:n,stack:ve(n)}}var fr=[],dr=0,kl=null,To=0,_i=[],vi=0,Ga=null,Vi=1,ki="";function _a(e,n){fr[dr++]=To,fr[dr++]=kl,kl=e,To=n}function Xm(e,n,a){_i[vi++]=Vi,_i[vi++]=ki,_i[vi++]=Ga,Ga=e;var s=Vi;e=ki;var l=32-ct(s)-1;s&=~(1<<l),a+=1;var u=32-ct(n)+l;if(30<u){var p=l-l%5;u=(s&(1<<p)-1).toString(32),s>>=p,l-=p,Vi=1<<32-ct(n)+l|a<<l|s,ki=u+e}else Vi=1<<u|a<<l|s,ki=e}function Xl(e){e.return!==null&&(_a(e,1),Xm(e,1,0))}function uf(e){for(;e===kl;)kl=fr[--dr],fr[dr]=null,To=fr[--dr],fr[dr]=null;for(;e===Ga;)Ga=_i[--vi],_i[vi]=null,ki=_i[--vi],_i[vi]=null,Vi=_i[--vi],_i[vi]=null}function Wm(e,n){_i[vi++]=Vi,_i[vi++]=ki,_i[vi++]=Ga,Vi=n.id,ki=n.overflow,Ga=e}var Mn=null,Jt=null,xt=!1,Va=null,xi=!1,ff=Error(r(519));function ka(e){var n=Error(r(418,1<arguments.length&&arguments[1]!==void 0&&arguments[1]?"text":"HTML",""));throw Ao(gi(n,e)),ff}function qm(e){var n=e.stateNode,a=e.type,s=e.memoizedProps;switch(n[T]=e,n[G]=s,a){case"dialog":Et("cancel",n),Et("close",n);break;case"iframe":case"object":case"embed":Et("load",n);break;case"video":case"audio":for(a=0;a<jo.length;a++)Et(jo[a],n);break;case"source":Et("error",n);break;case"img":case"image":case"link":Et("error",n),Et("load",n);break;case"details":Et("toggle",n);break;case"input":Et("invalid",n),am(n,s.value,s.defaultValue,s.checked,s.defaultChecked,s.type,s.name,!0);break;case"select":Et("invalid",n);break;case"textarea":Et("invalid",n),rm(n,s.value,s.defaultValue,s.children)}a=s.children,typeof a!="string"&&typeof a!="number"&&typeof a!="bigint"||n.textContent===""+a||s.suppressHydrationWarning===!0||h_(n.textContent,a)?(s.popover!=null&&(Et("beforetoggle",n),Et("toggle",n)),s.onScroll!=null&&Et("scroll",n),s.onScrollEnd!=null&&Et("scrollend",n),s.onClick!=null&&(n.onclick=Gi),n=!0):n=!1,n||ka(e,!0)}function Wl(e){for(Mn=e.return;Mn;)switch(Mn.tag){case 5:case 31:case 13:xi=!1;return;case 27:case 3:xi=!0;return;default:Mn=Mn.return}}function hr(e){if(e!==Mn)return!1;if(!xt)return Wl(e),xt=!0,!1;var n=e.tag,a;if((a=n!==3&&n!==27)&&((a=n===5)&&(a=e.type,a=!(a!=="form"&&a!=="button")||Gd(e.type,e.memoizedProps)),a=!a),a&&Jt&&ka(e),Wl(e),n===13){if(e=e.memoizedState,e=e!==null?e.dehydrated:null,!e)throw Error(r(317));Jt=U_(e)}else if(n===31){if(e=e.memoizedState,e=e!==null?e.dehydrated:null,!e)throw Error(r(317));Jt=U_(e)}else n===27?(n=Jt,ss(e.type)?(e=Kd,Kd=null,Jt=e):Jt=n):Jt=Mn?yi(e.stateNode.nextSibling):null;return!0}function Rs(){Jt=Mn=null,xt=!1}function df(){var e=Va;return e!==null&&(Yn===null?Yn=e:Yn.push.apply(Yn,e),Va=null),e}function Ao(e){Va===null?Va=[e]:Va.push(e)}var hf=St(null),ws=null,va=null;function Xa(e,n,a){it(hf,n._currentValue),n._currentValue=a}function xa(e){e._currentValue=hf.current,tt(hf)}function ql(e,n,a){for(;e!==null;){var s=e.alternate;if((e.childLanes&n)!==n?(e.childLanes|=n,s!==null&&(s.childLanes|=n)):s!==null&&(s.childLanes&n)!==n&&(s.childLanes|=n),e===a)break;e=e.return}}function pf(e,n,a,s){var l=e.child;for(l!==null&&(l.return=e);l!==null;){var u=l.dependencies;if(u!==null){var p=l.child;u=u.firstContext;e:for(;u!==null;){var A=u;u=l;for(var B=0;B<n.length;B++)if(A.context===n[B]){u.lanes|=a,A=u.alternate,A!==null&&(A.lanes|=a),ql(u.return,a,e),s||(p=null);break e}u=A.next}}else if(l.tag===18){if(p=l.return,p===null)throw Error(r(341));p.lanes|=a,u=p.alternate,u!==null&&(u.lanes|=a),ql(p,a,e),p=null}else l.tag===13&&l.memoizedState!==null&&l.memoizedState.dehydrated===null?(l.lanes|=a,p=l.alternate,p!==null&&(p.lanes|=a),ql(l.return,a,e),p=l.child,p=p!==null?p.sibling:null):p=l.child;if(p!==null)p.return=l;else for(p=l;p!==null;){if(p===e){p=null;break}if(l=p.sibling,l!==null){l.return=p.return,p=l;break}p=p.return}l=p}}function Cs(e,n,a,s){e=null;for(var l=n,u=!1;l!==null;){if(!u){if((l.flags&524288)!==0)u=!0;else if((l.flags&262144)!==0)break}if(l.tag===10){var p=l.alternate;if(p===null)throw Error(r(387));if(p=p.memoizedProps,p!==null){var A=l.type;ti(l.pendingProps.value,p.value)||(e!==null?e.push(A):e=[A])}}else if(l===en.current){if(p=l.alternate,p===null)throw Error(r(387));p.memoizedState.memoizedState!==l.memoizedState.memoizedState&&(e!==null?e.push(zr):e=[zr])}l=l.return}return e!==null&&pf(n,e,a,s),n.flags|=262144,e!==null}function Yl(e){for(e=e.firstContext;e!==null;){if(!ti(e.context._currentValue,e.memoizedValue))return!0;e=e.next}return!1}function Ns(e){ws=e,va=null,e=e.dependencies,e!==null&&(e.firstContext=null)}function An(e){return Ym(ws,e)}function jl(e,n){return ws===null&&Ns(e),Ym(e,n)}function Ym(e,n){var a=n._currentValue;if(n={context:n,memoizedValue:a,next:null},va===null){if(e===null)throw Error(r(308));va=n,e.dependencies={lanes:0,firstContext:n},e.flags|=524288}else va=va.next=n;return a}var QS=typeof AbortController<"u"?AbortController:function(){var e=[],n=this.signal={aborted:!1,addEventListener:function(a,s){e.push(s)}};this.abort=function(){n.aborted=!0,e.forEach(function(a){return a()})}},JS=o.unstable_scheduleCallback,$S=o.unstable_NormalPriority,hn={$$typeof:ne,Consumer:null,Provider:null,_currentValue:null,_currentValue2:null,_threadCount:0};function mf(){return{controller:new QS,data:new Map,refCount:0}}function Ro(e){e.refCount--,e.refCount===0&&JS($S,function(){e.controller.abort()})}function jm(e,n){if((e.pendingLanes&4194048)!==0){var a=e.transitionTypes;for(a===null&&(a=e.transitionTypes=[]),e=0;e<n.length;e++){var s=n[e];a.indexOf(s)===-1&&a.push(s)}}}var wo=null;function ey(e){var n=e.transitionTypes;return e.transitionTypes=null,n}var Co=null,gf=0,Ds=0,pr=null;function ty(e,n){if(Co===null){var a=Co=[];gf=0,Ds=Ud(),pr={status:"pending",value:void 0,then:function(s){a.push(s)}}}return gf++,n.then(Zm,Zm),n}function Zm(){if(--gf===0&&(wo=null,Co!==null)){pr!==null&&(pr.status="fulfilled");var e=Co;Co=null,Ds=0,pr=null;for(var n=0;n<e.length;n++)(0,e[n])()}}function ny(e,n){var a=[],s={status:"pending",value:null,reason:null,then:function(l){a.push(l)}};return e.then(function(){s.status="fulfilled",s.value=n;for(var l=0;l<a.length;l++)(0,a[l])(n)},function(l){for(s.status="rejected",s.reason=l,l=0;l<a.length;l++)(0,a[l])(void 0)}),s}var Km=xe.S;xe.S=function(e,n){if(V0=ke(),typeof n=="object"&&n!==null&&typeof n.then=="function"&&ty(e,n),wo!==null)for(var a=Ur;a!==null;)jm(a,wo),a=a.next;if(a=e.types,a!==null){for(var s=Ur;s!==null;)jm(s,a),s=s.next;if(Ds!==0){s=wo,s===null&&(s=wo=[]);for(var l=0;l<a.length;l++){var u=a[l];s.indexOf(u)===-1&&s.push(u)}}}Km!==null&&Km(e,n)};var Us=St(null);function _f(){var e=Us.current;return e!==null?e:Kt.pooledCache}function Zl(e,n){n===null?it(Us,Us.current):it(Us,n.pool)}function Qm(){var e=_f();return e===null?null:{parent:hn._currentValue,pool:e}}var mr=Error(r(460)),vf=Error(r(474)),Kl=Error(r(542)),Ql={then:function(){}};function Jm(e){return e=e.status,e==="fulfilled"||e==="rejected"}function $m(e,n,a){switch(a=e[a],a===void 0?e.push(n):a!==n&&(n.then(Gi,Gi),n=a),n.status){case"fulfilled":return n.value;case"rejected":throw e=n.reason,tg(e),e===void 0&&!("reason"in n)?Error(r(600)):e;default:if(typeof n.status=="string")n.then(Gi,Gi);else{if(e=Kt,e!==null&&100<e.shellSuspendCounter)throw Error(r(482));e=n,e.status="pending",e.then(function(s){if(n.status==="pending"){var l=n;l.status="fulfilled",l.value=s}},function(s){if(n.status==="pending"){var l=n;l.status="rejected",l.reason=s}})}switch(n.status){case"fulfilled":return n.value;case"rejected":throw e=n.reason,tg(e),e}throw Os=n,mr}}function Ls(e){try{var n=e._init;return n(e._payload)}catch(a){throw a!==null&&typeof a=="object"&&typeof a.then=="function"?(Os=a,mr):a}}var Os=null;function eg(){if(Os===null)throw Error(r(459));var e=Os;return Os=null,e}function tg(e){if(e===mr||e===Kl)throw Error(r(483))}var gr=null,No=0;function Jl(e){var n=No;return No+=1,gr===null&&(gr=[]),$m(gr,e,n)}function Wa(e,n){n=n.props.ref,e.ref=n!==void 0?n:null}function $l(e,n){throw n.$$typeof===b?Error(r(525)):(e=Object.prototype.toString.call(n),Error(r(31,e==="[object Object]"?"object with keys {"+Object.keys(n).join(", ")+"}":e)))}function ng(e){function n(ee,V){if(e){var ae=ee.deletions;ae===null?(ee.deletions=[V],ee.flags|=16):ae.push(V)}}function a(ee,V){if(!e)return null;for(;V!==null;)n(ee,V),V=V.sibling;return null}function s(ee){for(var V=new Map;ee!==null;)ee.key===null?V.set(ee.index,ee):V.set(ee.key,ee),ee=ee.sibling;return V}function l(ee,V){return ee=ga(ee,V),ee.index=0,ee.sibling=null,ee}function u(ee,V,ae){return ee.index=ae,e?(ae=ee.alternate,ae!==null?(ae=ae.index,ae<V?(ee.flags|=2,V):ae):(ee.flags|=134217730,V)):(ee.flags|=1048576,V)}function p(ee){return e&&ee.alternate===null&&(ee.flags|=134217730),ee}function A(ee,V,ae,Se){return V===null||V.tag!==6?(V=lf(ae,ee.mode,Se),V.return=ee,V):(V=l(V,ae),V.return=ee,V)}function B(ee,V,ae,Se){var qe=ae.type;return qe===Y?(ee=fe(ee,V,ae.props.children,Se,ae.key),Wa(ee,ae),ee):V!==null&&(V.elementType===qe||typeof qe=="object"&&qe!==null&&qe.$$typeof===me&&Ls(qe)===V.type)?(V=l(V,ae.props),Wa(V,ae),V.return=ee,V):(V=Vl(ae.type,ae.key,ae.props,null,ee.mode,Se),Wa(V,ae),V.return=ee,V)}function te(ee,V,ae,Se){return V===null||V.tag!==4||V.stateNode.containerInfo!==ae.containerInfo||V.stateNode.implementation!==ae.implementation?(V=cf(ae,ee.mode,Se),V.return=ee,V):(V=l(V,ae.children||[]),V.return=ee,V)}function fe(ee,V,ae,Se,qe){return V===null||V.tag!==7?(V=As(ae,ee.mode,Se,qe),V.return=ee,V):(V=l(V,ae),V.return=ee,V)}function ye(ee,V,ae){if(typeof V=="string"&&V!==""||typeof V=="number"||typeof V=="bigint")return V=lf(""+V,ee.mode,ae),V.return=ee,V;if(typeof V=="object"&&V!==null){switch(V.$$typeof){case L:return ae=Vl(V.type,V.key,V.props,null,ee.mode,ae),Wa(ae,V),ae.return=ee,ae;case k:return V=cf(V,ee.mode,ae),V.return=ee,V;case me:return V=Ls(V),ye(ee,V,ae)}if(Ie(V)||X(V))return V=As(V,ee.mode,ae,null),V.return=ee,V;if(typeof V.then=="function")return ye(ee,Jl(V),ae);if(V.$$typeof===ne)return ye(ee,jl(ee,V),ae);$l(ee,V)}return null}function J(ee,V,ae,Se){var qe=V!==null?V.key:null;if(typeof ae=="string"&&ae!==""||typeof ae=="number"||typeof ae=="bigint")return qe!==null?null:A(ee,V,""+ae,Se);if(typeof ae=="object"&&ae!==null){switch(ae.$$typeof){case L:return ae.key===qe?B(ee,V,ae,Se):null;case k:return ae.key===qe?te(ee,V,ae,Se):null;case me:return ae=Ls(ae),J(ee,V,ae,Se)}if(Ie(ae)||X(ae))return qe!==null?null:fe(ee,V,ae,Se,null);if(typeof ae.then=="function")return J(ee,V,Jl(ae),Se);if(ae.$$typeof===ne)return J(ee,V,jl(ee,ae),Se);$l(ee,ae)}return null}function oe(ee,V,ae,Se,qe){if(typeof Se=="string"&&Se!==""||typeof Se=="number"||typeof Se=="bigint")return ee=ee.get(ae)||null,A(V,ee,""+Se,qe);if(typeof Se=="object"&&Se!==null){switch(Se.$$typeof){case L:return ee=ee.get(Se.key===null?ae:Se.key)||null,B(V,ee,Se,qe);case k:return ee=ee.get(Se.key===null?ae:Se.key)||null,te(V,ee,Se,qe);case me:return Se=Ls(Se),oe(ee,V,ae,Se,qe)}if(Ie(Se)||X(Se))return ee=ee.get(ae)||null,fe(V,ee,Se,qe,null);if(typeof Se.then=="function")return oe(ee,V,ae,Jl(Se),qe);if(Se.$$typeof===ne)return oe(ee,V,ae,jl(V,Se),qe);$l(V,Se)}return null}function Oe(ee,V,ae,Se){for(var qe=null,At=null,et=V,at=V=0,gn=null;et!==null&&at<ae.length;at++){et.index>at?(gn=et,et=null):gn=et.sibling;var Lt=J(ee,et,ae[at],Se);if(Lt===null){et===null&&(et=gn);break}e&&et&&Lt.alternate===null&&n(ee,et),V=u(Lt,V,at),At===null?qe=Lt:At.sibling=Lt,At=Lt,et=gn}if(at===ae.length)return a(ee,et),xt&&_a(ee,at),qe;if(et===null){for(;at<ae.length;at++)et=ye(ee,ae[at],Se),et!==null&&(V=u(et,V,at),At===null?qe=et:At.sibling=et,At=et);return xt&&_a(ee,at),qe}for(et=s(et);at<ae.length;at++)gn=oe(et,ee,at,ae[at],Se),gn!==null&&(e&&(Lt=gn.alternate,Lt!==null&&et.delete(Lt.key===null?at:Lt.key)),V=u(gn,V,at),At===null?qe=gn:At.sibling=gn,At=gn);return e&&et.forEach(function(us){return n(ee,us)}),xt&&_a(ee,at),qe}function Qe(ee,V,ae,Se){if(ae==null)throw Error(r(151));for(var qe=null,At=null,et=V,at=V=0,gn=null,Lt=ae.next();et!==null&&!Lt.done;at++,Lt=ae.next()){et.index>at?(gn=et,et=null):gn=et.sibling;var us=J(ee,et,Lt.value,Se);if(us===null){et===null&&(et=gn);break}e&&et&&us.alternate===null&&n(ee,et),V=u(us,V,at),At===null?qe=us:At.sibling=us,At=us,et=gn}if(Lt.done)return a(ee,et),xt&&_a(ee,at),qe;if(et===null){for(;!Lt.done;at++,Lt=ae.next())Lt=ye(ee,Lt.value,Se),Lt!==null&&(V=u(Lt,V,at),At===null?qe=Lt:At.sibling=Lt,At=Lt);return xt&&_a(ee,at),qe}for(et=s(et);!Lt.done;at++,Lt=ae.next())Lt=oe(et,ee,at,Lt.value,Se),Lt!==null&&(e&&(gn=Lt.alternate,gn!==null&&et.delete(gn.key===null?at:gn.key)),V=u(Lt,V,at),At===null?qe=Lt:At.sibling=Lt,At=Lt);return e&&et.forEach(function(IM){return n(ee,IM)}),xt&&_a(ee,at),qe}function ht(ee,V,ae,Se){if(typeof ae=="object"&&ae!==null&&ae.type===Y&&ae.key===null&&ae.props.ref===void 0&&(ae=ae.props.children),typeof ae=="object"&&ae!==null){switch(ae.$$typeof){case L:e:{for(var qe=ae.key;V!==null;){if(V.key===qe){if(qe=ae.type,qe===Y){if(V.tag===7){a(ee,V.sibling),Se=l(V,ae.props.children),Wa(Se,ae),Se.return=ee,ee=Se;break e}}else if(V.elementType===qe||typeof qe=="object"&&qe!==null&&qe.$$typeof===me&&Ls(qe)===V.type){a(ee,V.sibling),Se=l(V,ae.props),Wa(Se,ae),Se.return=ee,ee=Se;break e}a(ee,V);break}else n(ee,V);V=V.sibling}ae.type===Y?(Se=As(ae.props.children,ee.mode,Se,ae.key),Wa(Se,ae),Se.return=ee,ee=Se):(Se=Vl(ae.type,ae.key,ae.props,null,ee.mode,Se),Wa(Se,ae),Se.return=ee,ee=Se)}return p(ee);case k:e:{for(qe=ae.key;V!==null;){if(V.key===qe)if(V.tag===4&&V.stateNode.containerInfo===ae.containerInfo&&V.stateNode.implementation===ae.implementation){a(ee,V.sibling),Se=l(V,ae.children||[]),Se.return=ee,ee=Se;break e}else{a(ee,V);break}else n(ee,V);V=V.sibling}Se=cf(ae,ee.mode,Se),Se.return=ee,ee=Se}return p(ee);case me:return ae=Ls(ae),ht(ee,V,ae,Se)}if(Ie(ae))return Oe(ee,V,ae,Se);if(X(ae)){if(qe=X(ae),typeof qe!="function")throw Error(r(150));return ae=qe.call(ae),Qe(ee,V,ae,Se)}if(typeof ae.then=="function")return ht(ee,V,Jl(ae),Se);if(ae.$$typeof===ne)return ht(ee,V,jl(ee,ae),Se);$l(ee,ae)}return typeof ae=="string"&&ae!==""||typeof ae=="number"||typeof ae=="bigint"?(ae=""+ae,V!==null&&V.tag===6?(a(ee,V.sibling),Se=l(V,ae),Se.return=ee,ee=Se):(a(ee,V),Se=lf(ae,ee.mode,Se),Se.return=ee,ee=Se),p(ee)):a(ee,V)}return function(ee,V,ae,Se){try{No=0;var qe=ht(ee,V,ae,Se);return gr=null,qe}catch(et){if(et===mr||et===Kl)throw et;var At=Xn(29,et,null,ee.mode);return At.lanes=Se,At.return=ee,At}finally{}}}var Ps=ng(!0),ig=ng(!1),qa=!1;function xf(e){e.updateQueue={baseState:e.memoizedState,firstBaseUpdate:null,lastBaseUpdate:null,shared:{pending:null,lanes:0,hiddenCallbacks:null},callbacks:null}}function Sf(e,n){e=e.updateQueue,n.updateQueue===e&&(n.updateQueue={baseState:e.baseState,firstBaseUpdate:e.firstBaseUpdate,lastBaseUpdate:e.lastBaseUpdate,shared:e.shared,callbacks:null})}function Ya(e){return{lane:e,tag:0,payload:null,callback:null,next:null}}function ja(e,n,a){var s=e.updateQueue;if(s===null)return null;if(s=s.shared,(zt&2)!==0){var l=s.pending;return l===null?n.next=n:(n.next=l.next,l.next=n),s.pending=n,n=Gl(e),Hm(e,null,a),n}return Hl(e,s,n,a),Gl(e)}function Do(e,n,a){if(n=n.updateQueue,n!==null&&(n=n.shared,(a&4194048)!==0)){var s=n.lanes;s&=e.pendingLanes,a|=s,n.lanes=a,fo(e,a)}}function yf(e,n){var a=e.updateQueue,s=e.alternate;if(s!==null&&(s=s.updateQueue,a===s)){var l=null,u=null;if(a=a.firstBaseUpdate,a!==null){do{var p={lane:a.lane,tag:a.tag,payload:a.payload,callback:null,next:null};u===null?l=u=p:u=u.next=p,a=a.next}while(a!==null);u===null?l=u=n:u=u.next=n}else l=u=n;a={baseState:s.baseState,firstBaseUpdate:l,lastBaseUpdate:u,shared:s.shared,callbacks:s.callbacks},e.updateQueue=a;return}e=a.lastBaseUpdate,e===null?a.firstBaseUpdate=n:e.next=n,a.lastBaseUpdate=n}var Mf=!1;function Uo(){if(Mf){var e=pr;if(e!==null)throw e}}function Lo(e,n,a,s){Mf=!1;var l=e.updateQueue;qa=!1;var u=l.firstBaseUpdate,p=l.lastBaseUpdate,A=l.shared.pending;if(A!==null){l.shared.pending=null;var B=A,te=B.next;B.next=null,p===null?u=te:p.next=te,p=B;var fe=e.alternate;fe!==null&&(fe=fe.updateQueue,A=fe.lastBaseUpdate,A!==p&&(A===null?fe.firstBaseUpdate=te:A.next=te,fe.lastBaseUpdate=B))}if(u!==null){var ye=l.baseState;p=0,fe=te=B=null,A=u;do{var J=A.lane&-536870913,oe=J!==A.lane;if(oe?(Tt&J)===J:(s&J)===J){J!==0&&J===Ds&&(Mf=!0),fe!==null&&(fe=fe.next={lane:0,tag:A.tag,payload:A.payload,callback:null,next:null});e:{var Oe=e,Qe=A;J=n;var ht=a;switch(Qe.tag){case 1:if(Oe=Qe.payload,typeof Oe=="function"){ye=Oe.call(ht,ye,J);break e}ye=Oe;break e;case 3:Oe.flags=Oe.flags&-65537|128;case 0:if(Oe=Qe.payload,J=typeof Oe=="function"?Oe.call(ht,ye,J):Oe,J==null)break e;ye=I({},ye,J);break e;case 2:qa=!0}}J=A.callback,J!==null&&(e.flags|=64,oe&&(e.flags|=8192),oe=l.callbacks,oe===null?l.callbacks=[J]:oe.push(J))}else oe={lane:J,tag:A.tag,payload:A.payload,callback:A.callback,next:null},fe===null?(te=fe=oe,B=ye):fe=fe.next=oe,p|=J;if(A=A.next,A===null){if(A=l.shared.pending,A===null)break;oe=A,A=oe.next,oe.next=null,l.lastBaseUpdate=oe,l.shared.pending=null}}while(!0);fe===null&&(B=ye),l.baseState=B,l.firstBaseUpdate=te,l.lastBaseUpdate=fe,u===null&&(l.shared.lanes=0),ts|=p,e.lanes=p,e.memoizedState=ye}}function ag(e,n){if(typeof e!="function")throw Error(r(191,e));e.call(n)}function sg(e,n){var a=e.callbacks;if(a!==null)for(e.callbacks=null,e=0;e<a.length;e++)ag(a[e],n)}var Za=St(null),ec=St(0);function rg(e,n){e=ba,it(ec,e),it(Za,n),ba=e|n.baseLanes}function Ef(){it(ec,ba),it(Za,Za.current)}function bf(){ba=ec.current,tt(Za),tt(ec)}var Rn=St(null),Ln=null;function Ka(e){var n=e.alternate;it(wn,wn.current&1),it(Rn,e),Ln===null&&(n===null||Za.current!==null||n.memoizedState!==null)&&(Ln=e)}function Tf(e){it(wn,wn.current),it(Rn,e),Ln===null&&(Ln=e)}function og(e){e.tag===22?(it(wn,wn.current),it(Rn,e),Ln===null&&(Ln=e)):Qa()}function Qa(){it(wn,wn.current),it(Rn,Rn.current)}function ni(e){tt(Rn),Ln===e&&(Ln=null),tt(wn)}var wn=St(0);function Oo(e,n){it(Rn,Rn.current),it(wn,n)}function Af(e){tt(wn),tt(Rn),Ln===e&&(Ln=null)}function tc(e){for(var n=e;n!==null;){if(n.tag===13){var a=n.memoizedState;if(a!==null&&(a=a.dehydrated,a===null||jd(a)||Zd(a)))return n}else if(n.tag===19&&n.memoizedProps.revealOrder!=="independent"){if((n.flags&128)!==0)return n}else if(n.child!==null){n.child.return=n,n=n.child;continue}if(n===e)break;for(;n.sibling===null;){if(n.return===null||n.return===e)return null;n=n.return}n.sibling.return=n.return,n=n.sibling}return null}var Sa=0,dt=null,Zt=null,pn=null,nc=!1,_r=!1,Is=!1,ic=0,Po=0,vr=null,iy=0;function ln(){throw Error(r(321))}function Rf(e,n){if(n===null)return!1;for(var a=0;a<n.length&&a<e.length;a++)if(!ti(e[a],n[a]))return!1;return!0}function wf(e,n,a,s,l,u){return Sa=u,dt=n,n.memoizedState=null,n.updateQueue=null,n.lanes=0,xe.H=e===null||e.memoizedState===null?Xg:Wg,Is=!1,u=a(s,l),Is=!1,_r&&(u=cg(n,a,s,l)),lg(e),u}function lg(e){xe.H=uc;var n=Zt!==null&&Zt.next!==null;if(Sa=0,pn=Zt=dt=null,nc=!1,Po=0,vr=null,n)throw Error(r(300));e===null||mn||(e=e.dependencies,e!==null&&Yl(e)&&(mn=!0))}function cg(e,n,a,s){dt=e;var l=0;do{if(_r&&(vr=null),Po=0,_r=!1,25<=l)throw Error(r(301));if(l+=1,pn=Zt=null,e.updateQueue!=null){var u=e.updateQueue;u.lastEffect=null,u.events=null,u.stores=null,u.memoCache!=null&&(u.memoCache.index=0)}xe.H=fy,u=n(a,s)}while(_r);return u}function ay(){var e=xe.H,n=e.useState()[0];return n=typeof n.then=="function"?Io(n):n,e=e.useState()[0],(Zt!==null?Zt.memoizedState:null)!==e&&(dt.flags|=1024),n}function Cf(){var e=ic!==0;return ic=0,e}function Nf(e,n,a){n.updateQueue=e.updateQueue,n.flags&=-2053,e.lanes&=~a}function Df(e){if(nc){for(e=e.memoizedState;e!==null;){var n=e.queue;n!==null&&(n.pending=null),e=e.next}nc=!1}Sa=0,pn=Zt=dt=null,_r=!1,Po=ic=0,vr=null}function Fn(){var e={memoizedState:null,baseState:null,baseQueue:null,queue:null,next:null};return pn===null?dt.memoizedState=pn=e:pn=pn.next=e,pn}function fn(){if(Zt===null){var e=dt.alternate;e=e!==null?e.memoizedState:null}else e=Zt.next;var n=pn===null?dt.memoizedState:pn.next;if(n!==null)pn=n,Zt=e;else{if(e===null)throw dt.alternate===null?Error(r(467)):Error(r(310));Zt=e,e={memoizedState:Zt.memoizedState,baseState:Zt.baseState,baseQueue:Zt.baseQueue,queue:Zt.queue,next:null},pn===null?dt.memoizedState=pn=e:pn=pn.next=e}return pn}function ac(){return{lastEffect:null,events:null,stores:null,memoCache:null}}function Io(e){var n=Po;return Po+=1,vr===null&&(vr=[]),e=$m(vr,e,n),n=dt,(pn===null?n.memoizedState:pn.next)===null&&(n=n.alternate,xe.H=n===null||n.memoizedState===null?Xg:Wg),e}function sc(e){if(e!==null&&typeof e=="object"){if(typeof e.then=="function")return Io(e);if(e.$$typeof===F)return;if(e.$$typeof===ne)return An(e)}throw Error(r(438,String(e)))}function Uf(e){var n=null,a=dt.updateQueue;if(a!==null&&(n=a.memoCache),n==null){var s=dt.alternate;s!==null&&(s=s.updateQueue,s!==null&&(s=s.memoCache,s!=null&&(n={data:s.data.map(function(l){return l.slice()}),index:0})))}if(n==null&&(n={data:[],index:0}),a===null&&(a=ac(),dt.updateQueue=a),a.memoCache=n,a=n.data[n.index],a===void 0)for(a=n.data[n.index]=Array(e),s=0;s<e;s++)a[s]=je;return n.index++,a}function ya(e,n){return typeof n=="function"?n(e):n}function rc(e){var n=fn();return Lf(n,Zt,e)}function Lf(e,n,a){var s=e.queue;if(s===null)throw Error(r(311));s.lastRenderedReducer=a;var l=e.baseQueue,u=s.pending;if(u!==null){if(l!==null){var p=l.next;l.next=u.next,u.next=p}n.baseQueue=l=u,s.pending=null}if(u=e.baseState,l===null)e.memoizedState=u;else{n=l.next;var A=p=null,B=null,te=n,fe=!1;do{var ye=te.lane&-536870913;if(ye!==te.lane?(Tt&ye)===ye:(Sa&ye)===ye){var J=te.revertLane;if(J===0)B!==null&&(B=B.next={lane:0,revertLane:0,gesture:null,action:te.action,hasEagerState:te.hasEagerState,eagerState:te.eagerState,next:null}),ye===Ds&&(fe=!0);else if((Sa&J)===J){te=te.next,J===Ds&&(fe=!0);continue}else ye={lane:0,revertLane:te.revertLane,gesture:null,action:te.action,hasEagerState:te.hasEagerState,eagerState:te.eagerState,next:null},B===null?(A=B=ye,p=u):B=B.next=ye,dt.lanes|=J,ts|=J;ye=te.action,Is&&a(u,ye),u=te.hasEagerState?te.eagerState:a(u,ye)}else J={lane:ye,revertLane:te.revertLane,gesture:te.gesture,action:te.action,hasEagerState:te.hasEagerState,eagerState:te.eagerState,next:null},B===null?(A=B=J,p=u):B=B.next=J,dt.lanes|=ye,ts|=ye;te=te.next}while(te!==null&&te!==n);if(B===null?p=u:B.next=A,!ti(u,e.memoizedState)&&(mn=!0,fe&&(a=pr,a!==null)))throw a;e.memoizedState=u,e.baseState=p,e.baseQueue=B,s.lastRenderedState=u}return l===null&&(s.lanes=0),[e.memoizedState,s.dispatch]}function Of(e){var n=fn(),a=n.queue;if(a===null)throw Error(r(311));a.lastRenderedReducer=e;var s=a.dispatch,l=a.pending,u=n.memoizedState;if(l!==null){a.pending=null;var p=l=l.next;do u=e(u,p.action),p=p.next;while(p!==l);ti(u,n.memoizedState)||(mn=!0),n.memoizedState=u,n.baseQueue===null&&(n.baseState=u),a.lastRenderedState=u}return[u,s]}function ug(e,n,a){var s=dt,l=fn(),u=xt;if(u){if(a===void 0)throw Error(r(407));a=a()}else a=n();var p=!ti((Zt||l).memoizedState,a);if(p&&(l.memoizedState=a,mn=!0),l=l.queue,Bf(hg.bind(null,s,l,e),[e]),e=l.getSnapshot!==n||p||pn!==null&&(pn.memoizedState.tag&1)!==0,xr(e?9:8,{destroy:void 0},dg.bind(null,s,l,a,n),null),e){if(s.flags|=2048,Kt===null)throw Error(r(349));u||(Sa&127)!==0||fg(s,n,a)}return a}function fg(e,n,a){e.flags|=16384,e={getSnapshot:n,value:a},n=dt.updateQueue,n===null?(n=ac(),dt.updateQueue=n,n.stores=[e]):(a=n.stores,a===null?n.stores=[e]:a.push(e))}function dg(e,n,a,s){n.value=a,n.getSnapshot=s,pg(n)&&mg(e)}function hg(e,n,a){return a(function(){pg(n)&&mg(e)})}function pg(e){var n=e.getSnapshot;e=e.value;try{var a=n();return!ti(e,a)}catch{return!0}}function mg(e){var n=Ts(e,2);n!==null&&jn(n,e,2)}function Pf(e){var n=Fn();if(typeof e=="function"){var a=e;if(e=a(),Is){Rt(!0);try{a()}finally{Rt(!1)}}}return n.memoizedState=n.baseState=e,n.queue={pending:null,lanes:0,dispatch:null,lastRenderedReducer:ya,lastRenderedState:e},n}function gg(e,n,a,s){return e.baseState=a,Lf(e,Zt,typeof s=="function"?s:ya)}function sy(e,n,a,s,l){if(cc(e))throw Error(r(485));if(e=n.action,e!==null){var u={payload:l,action:e,next:null,isTransition:!0,status:"pending",value:null,reason:null,listeners:[],then:function(p){u.listeners.push(p)}};xe.T!==null?a(!0):u.isTransition=!1,s(u),a=n.pending,a===null?(u.next=n.pending=u,_g(n,u)):(u.next=a.next,n.pending=a.next=u)}}function _g(e,n){var a=n.action,s=n.payload,l=e.state;if(n.isTransition){var u=xe.T,p={};p.types=u!==null?u.types:null,xe.T=p;try{var A=a(l,s),B=xe.S;B!==null&&B(p,A),vg(e,n,A)}catch(te){If(e,n,te)}finally{u!==null&&p.types!==null&&(u.types=p.types),xe.T=u}}else try{u=a(l,s),vg(e,n,u)}catch(te){If(e,n,te)}}function vg(e,n,a){a!==null&&typeof a=="object"&&typeof a.then=="function"?a.then(function(s){xg(e,n,s)},function(s){return If(e,n,s)}):xg(e,n,a)}function xg(e,n,a){n.status="fulfilled",n.value=a,Sg(n),e.state=a,n=e.pending,n!==null&&(a=n.next,a===n?e.pending=null:(a=a.next,n.next=a,_g(e,a)))}function If(e,n,a){var s=e.pending;if(e.pending=null,s!==null){s=s.next;do n.status="rejected",n.reason=a,Sg(n),n=n.next;while(n!==s)}e.action=null}function Sg(e){e=e.listeners;for(var n=0;n<e.length;n++)(0,e[n])()}function yg(e,n){return n}function Mg(e,n){if(xt){var a=Kt.formState;if(a!==null){e:{var s=dt;if(xt){if(Jt){t:{for(var l=Jt,u=xi;l.nodeType!==8;){if(!u){l=null;break t}if(l=yi(l.nextSibling),l===null){l=null;break t}}u=l.data,l=u==="F!"||u==="F"?l:null}if(l){Jt=yi(l.nextSibling),s=l.data==="F!";break e}}ka(s)}s=!1}s&&(n=a[0])}}return a=Fn(),a.memoizedState=a.baseState=n,s={pending:null,lanes:0,dispatch:null,lastRenderedReducer:yg,lastRenderedState:n},a.queue=s,a=Gg.bind(null,dt,s),s.dispatch=a,s=Pf(!1),u=Vf.bind(null,dt,!1,s.queue),s=Fn(),l={state:n,dispatch:null,action:e,pending:null},s.queue=l,a=sy.bind(null,dt,l,u,a),l.dispatch=a,s.memoizedState=e,[n,a,!1]}function Eg(e){var n=fn();return bg(n,Zt,e)}function bg(e,n,a){if(n=Lf(e,n,yg)[0],e=rc(ya)[0],typeof n=="object"&&n!==null&&typeof n.then=="function")try{var s=Io(n)}catch(p){throw p===mr?Kl:p}else s=n;n=fn();var l=n.queue,u=l.dispatch;return a!==n.memoizedState&&(dt.flags|=2048,xr(9,{destroy:void 0},ry.bind(null,l,a),null)),[s,u,e]}function ry(e,n){e.action=n}function Tg(e){var n=fn(),a=Zt;if(a!==null)return bg(n,a,e);fn(),n=n.memoizedState,a=fn();var s=a.queue.dispatch;return a.memoizedState=e,[n,s,!1]}function xr(e,n,a,s){return e={tag:e,create:a,deps:s,inst:n,next:null},n=dt.updateQueue,n===null&&(n=ac(),dt.updateQueue=n),a=n.lastEffect,a===null?n.lastEffect=e.next=e:(s=a.next,a.next=e,e.next=s,n.lastEffect=e),e}function Ag(){return fn().memoizedState}function oc(e,n,a,s){var l=Fn();dt.flags|=e,l.memoizedState=xr(1|n,{destroy:void 0},a,s===void 0?null:s)}function lc(e,n,a,s){var l=fn();s=s===void 0?null:s;var u=l.memoizedState.inst;Zt!==null&&s!==null&&Rf(s,Zt.memoizedState.deps)?l.memoizedState=xr(n,u,a,s):(dt.flags|=e,l.memoizedState=xr(1|n,u,a,s))}function Rg(e,n){oc(8390656,8,e,n)}function Bf(e,n){lc(2048,8,e,n)}function oy(e){dt.flags|=4;var n=dt.updateQueue;if(n===null)n=ac(),dt.updateQueue=n,n.events=[e];else{var a=n.events;a===null?n.events=[e]:a.push(e)}}function wg(e){var n=fn().memoizedState;return oy({ref:n,nextImpl:e}),function(){if((zt&2)!==0)throw Error(r(440));return n.impl.apply(void 0,arguments)}}function Cg(e,n){return lc(4,2,e,n)}function Ng(e,n){return lc(4,4,e,n)}function Dg(e,n){if(typeof n=="function"){e=e();var a=n(e);return function(){typeof a=="function"?a():n(null)}}if(n!=null)return e=e(),n.current=e,function(){n.current=null}}function Ug(e,n,a){a=a!=null?a.concat([e]):null,lc(4,4,Dg.bind(null,n,e),a)}function zf(){}function Lg(e,n){var a=fn();n=n===void 0?null:n;var s=a.memoizedState;return n!==null&&Rf(n,s[1])?s[0]:(a.memoizedState=[e,n],e)}function Og(e,n){var a=fn();n=n===void 0?null:n;var s=a.memoizedState;if(n!==null&&Rf(n,s[1]))return s[0];if(s=e(),Is){Rt(!0);try{e()}finally{Rt(!1)}}return a.memoizedState=[s,n],s}function Ff(e,n,a){return a===void 0||(Sa&1073741824)!==0&&(Tt&261930)===0?e.memoizedState=n:(e.memoizedState=a,e=X0(),dt.lanes|=e,ts|=e,a)}function Pg(e,n,a,s){return ti(a,n)?a:Za.current!==null?(e=Ff(e,a,s),ti(e,n)||(mn=!0),e):(Sa&106)===0||(Sa&1073741824)!==0&&(Tt&261930)===0?(mn=!0,e.memoizedState=a):(e=X0(),dt.lanes|=e,ts|=e,n)}function Ig(e,n,a,s,l){var u=we.p;we.p=u!==0&&8>u?u:8;var p=xe.T,A={};A.types=p!==null?p.types:null,xe.T=A,Vf(e,!1,n,a);try{var B=l(),te=xe.S;if(te!==null&&te(A,B),B!==null&&typeof B=="object"&&typeof B.then=="function"){var fe=ny(B,s);Bo(e,n,fe,ri(e))}else Bo(e,n,s,ri(e))}catch(ye){Bo(e,n,{then:function(){},status:"rejected",reason:ye},ri())}finally{we.p=u,p!==null&&A.types!==null&&(p.types=A.types),xe.T=p}}function ly(){}function Hf(e,n,a,s){if(e.tag!==5)throw Error(r(476));var l=Bg(e).queue;Ig(e,l,n,Vt,a===null?ly:function(){return zg(e),a(s)})}function Bg(e){var n=e.memoizedState;if(n!==null)return n;n={memoizedState:Vt,baseState:Vt,baseQueue:null,queue:{pending:null,lanes:0,dispatch:null,lastRenderedReducer:ya,lastRenderedState:Vt},next:null};var a={};return n.next={memoizedState:a,baseState:a,baseQueue:null,queue:{pending:null,lanes:0,dispatch:null,lastRenderedReducer:ya,lastRenderedState:a},next:null},e.memoizedState=n,e=e.alternate,e!==null&&(e.memoizedState=n),n}function zg(e){var n=Bg(e);n.next===null&&(n=e.alternate.memoizedState),Bo(e,n.next.queue,{},ri())}function Gf(){return An(zr)}function Fg(){return fn().memoizedState}function Hg(){return fn().memoizedState}function cy(e){for(var n=e.return;n!==null;){switch(n.tag){case 24:case 3:var a=ri();e=Ya(a);var s=ja(n,e,a);s!==null&&(jn(s,n,a),Do(s,n,a)),n={cache:mf()},e.payload=n;return}n=n.return}}function uy(e,n,a){var s=ri();a={lane:s,revertLane:0,gesture:null,action:a,hasEagerState:!1,eagerState:null,next:null},cc(e)?Vg(n,a):(a=rf(e,n,a,s),a!==null&&(jn(a,e,s),kg(a,n,s)))}function Gg(e,n,a){var s=ri();Bo(e,n,a,s)}function Bo(e,n,a,s){var l={lane:s,revertLane:0,gesture:null,action:a,hasEagerState:!1,eagerState:null,next:null};if(cc(e))Vg(n,l);else{var u=e.alternate;if(e.lanes===0&&(u===null||u.lanes===0)&&(u=n.lastRenderedReducer,u!==null))try{var p=n.lastRenderedState,A=u(p,a);if(l.hasEagerState=!0,l.eagerState=A,ti(A,p))return Hl(e,n,l,0),Kt===null&&Fl(),!1}catch{}finally{}if(a=rf(e,n,l,s),a!==null)return jn(a,e,s),kg(a,n,s),!0}return!1}function Vf(e,n,a,s){if(s={lane:2,revertLane:Ud(),gesture:null,action:s,hasEagerState:!1,eagerState:null,next:null},cc(e)){if(n)throw Error(r(479))}else n=rf(e,a,s,2),n!==null&&jn(n,e,2)}function cc(e){var n=e.alternate;return e===dt||n!==null&&n===dt}function Vg(e,n){_r=nc=!0;var a=e.pending;a===null?n.next=n:(n.next=a.next,a.next=n),e.pending=n}function kg(e,n,a){if((a&4194048)!==0){var s=n.lanes;s&=e.pendingLanes,a|=s,n.lanes=a,fo(e,a)}}var uc={readContext:An,use:sc,useCallback:ln,useContext:ln,useEffect:ln,useImperativeHandle:ln,useLayoutEffect:ln,useInsertionEffect:ln,useMemo:ln,useReducer:ln,useRef:ln,useState:ln,useDebugValue:ln,useDeferredValue:ln,useTransition:ln,useSyncExternalStore:ln,useId:ln,useHostTransitionStatus:ln,useFormState:ln,useActionState:ln,useOptimistic:ln,useMemoCache:ln,useCacheRefresh:ln,useEffectEvent:ln},Xg={readContext:An,use:sc,useCallback:function(e,n){return Fn().memoizedState=[e,n===void 0?null:n],e},useContext:An,useEffect:Rg,useImperativeHandle:function(e,n,a){a=a!=null?a.concat([e]):null,oc(4194308,4,Dg.bind(null,n,e),a)},useLayoutEffect:function(e,n){return oc(4194308,4,e,n)},useInsertionEffect:function(e,n){oc(4,2,e,n)},useMemo:function(e,n){var a=Fn();n=n===void 0?null:n;var s=e();if(Is){Rt(!0);try{e()}finally{Rt(!1)}}return a.memoizedState=[s,n],s},useReducer:function(e,n,a){var s=Fn();if(a!==void 0){var l=a(n);if(Is){Rt(!0);try{a(n)}finally{Rt(!1)}}}else l=n;return s.memoizedState=s.baseState=l,e={pending:null,lanes:0,dispatch:null,lastRenderedReducer:e,lastRenderedState:l},s.queue=e,e=e.dispatch=uy.bind(null,dt,e),[s.memoizedState,e]},useRef:function(e){var n=Fn();return e={current:e},n.memoizedState=e},useState:function(e){e=Pf(e);var n=e.queue,a=Gg.bind(null,dt,n);return n.dispatch=a,[e.memoizedState,a]},useDebugValue:zf,useDeferredValue:function(e,n){var a=Fn();return Ff(a,e,n)},useTransition:function(){var e=Pf(!1);return e=Ig.bind(null,dt,e.queue,!0,!1),Fn().memoizedState=e,[!1,e]},useSyncExternalStore:function(e,n,a){var s=dt,l=Fn();if(xt){if(a===void 0)throw Error(r(407));a=a()}else{if(a=n(),Kt===null)throw Error(r(349));(Tt&127)!==0||fg(s,n,a)}l.memoizedState=a;var u={value:a,getSnapshot:n};return l.queue=u,Rg(hg.bind(null,s,u,e),[e]),s.flags|=2048,xr(9,{destroy:void 0},dg.bind(null,s,u,a,n),null),a},useId:function(){var e=Fn(),n=Kt.identifierPrefix;if(xt){var a=ki,s=Vi;a=(s&~(1<<32-ct(s)-1)).toString(32)+a,n="_"+n+"R_"+a,a=ic++,0<a&&(n+="H"+a.toString(32)),n+="_"}else a=iy++,n="_"+n+"r_"+a.toString(32)+"_";return e.memoizedState=n},useHostTransitionStatus:Gf,useFormState:Mg,useActionState:Mg,useOptimistic:function(e){var n=Fn();n.memoizedState=n.baseState=e;var a={pending:null,lanes:0,dispatch:null,lastRenderedReducer:null,lastRenderedState:null};return n.queue=a,n=Vf.bind(null,dt,!0,a),a.dispatch=n,[e,n]},useMemoCache:Uf,useCacheRefresh:function(){return Fn().memoizedState=cy.bind(null,dt)},useEffectEvent:function(e){var n=Fn(),a={impl:e};return n.memoizedState=a,function(){if((zt&2)!==0)throw Error(r(440));return a.impl.apply(void 0,arguments)}}},Wg={readContext:An,use:sc,useCallback:Lg,useContext:An,useEffect:Bf,useImperativeHandle:Ug,useInsertionEffect:Cg,useLayoutEffect:Ng,useMemo:Og,useReducer:rc,useRef:Ag,useState:function(){return rc(ya)},useDebugValue:zf,useDeferredValue:function(e,n){var a=fn();return Pg(a,Zt.memoizedState,e,n)},useTransition:function(){var e=rc(ya)[0],n=fn().memoizedState;return[typeof e=="boolean"?e:Io(e),n]},useSyncExternalStore:ug,useId:Fg,useHostTransitionStatus:Gf,useFormState:Eg,useActionState:Eg,useOptimistic:function(e,n){var a=fn();return gg(a,Zt,e,n)},useMemoCache:Uf,useCacheRefresh:Hg,useEffectEvent:wg},fy={readContext:An,use:sc,useCallback:Lg,useContext:An,useEffect:Bf,useImperativeHandle:Ug,useInsertionEffect:Cg,useLayoutEffect:Ng,useMemo:Og,useReducer:Of,useRef:Ag,useState:function(){return Of(ya)},useDebugValue:zf,useDeferredValue:function(e,n){var a=fn();return Zt===null?Ff(a,e,n):Pg(a,Zt.memoizedState,e,n)},useTransition:function(){var e=Of(ya)[0],n=fn().memoizedState;return[typeof e=="boolean"?e:Io(e),n]},useSyncExternalStore:ug,useId:Fg,useHostTransitionStatus:Gf,useFormState:Tg,useActionState:Tg,useOptimistic:function(e,n){var a=fn();return Zt!==null?gg(a,Zt,e,n):(a.baseState=e,[e,a.queue.dispatch])},useMemoCache:Uf,useCacheRefresh:Hg,useEffectEvent:wg};function kf(e,n,a,s){n=e.memoizedState,a=a(s,n),a=a==null?n:I({},n,a),e.memoizedState=a,e.lanes===0&&(e.updateQueue.baseState=a)}var Xf={enqueueSetState:function(e,n,a){e=e._reactInternals;var s=ri(),l=Ya(s);l.payload=n,a!=null&&(l.callback=a),n=ja(e,l,s),n!==null&&(jn(n,e,s),Do(n,e,s))},enqueueReplaceState:function(e,n,a){e=e._reactInternals;var s=ri(),l=Ya(s);l.tag=1,l.payload=n,a!=null&&(l.callback=a),n=ja(e,l,s),n!==null&&(jn(n,e,s),Do(n,e,s))},enqueueForceUpdate:function(e,n){e=e._reactInternals;var a=ri(),s=Ya(a);s.tag=2,n!=null&&(s.callback=n),n=ja(e,s,a),n!==null&&(jn(n,e,a),Do(n,e,a))}};function qg(e,n,a,s,l,u,p){return e=e.stateNode,typeof e.shouldComponentUpdate=="function"?e.shouldComponentUpdate(s,u,p):n.prototype&&n.prototype.isPureReactComponent?!Eo(a,s)||!Eo(l,u):!0}function Yg(e,n,a,s){e=n.state,typeof n.componentWillReceiveProps=="function"&&n.componentWillReceiveProps(a,s),typeof n.UNSAFE_componentWillReceiveProps=="function"&&n.UNSAFE_componentWillReceiveProps(a,s),n.state!==e&&Xf.enqueueReplaceState(n,n.state,null)}function Bs(e,n){var a=n;if("ref"in n){a={};for(var s in n)s!=="ref"&&(a[s]=n[s])}if(e=e.defaultProps){a===n&&(a=I({},a));for(var l in e)a[l]===void 0&&(a[l]=e[l])}return a}function jg(e){zl(e)}function Zg(e){console.error(e)}function Kg(e){zl(e)}function fc(e,n){try{var a=e.onUncaughtError;a(n.value,{componentStack:n.stack})}catch(s){setTimeout(function(){throw s})}}function Qg(e,n,a){try{var s=e.onCaughtError;s(a.value,{componentStack:a.stack,errorBoundary:n.tag===1?n.stateNode:null})}catch(l){setTimeout(function(){throw l})}}function Wf(e,n,a){return a=Ya(a),a.tag=3,a.payload={element:null},a.callback=function(){fc(e,n)},a}function Jg(e){return e=Ya(e),e.tag=3,e}function $g(e,n,a,s){var l=a.type.getDerivedStateFromError;if(typeof l=="function"){var u=s.value;e.payload=function(){return l(u)},e.callback=function(){Qg(n,a,s)}}var p=a.stateNode;p!==null&&typeof p.componentDidCatch=="function"&&(e.callback=function(){Qg(n,a,s),typeof l!="function"&&(ns===null?ns=new Set([this]):ns.add(this));var A=s.stack;this.componentDidCatch(s.value,{componentStack:A!==null?A:""})})}function dy(e,n,a,s,l){if(a.flags|=32768,s!==null&&typeof s=="object"&&typeof s.then=="function"){if(n=a.alternate,n!==null&&Cs(n,a,l,!0),a=Rn.current,a!==null){switch(a.tag){case 31:case 13:case 19:return Ln===null?Uc():a.alternate===null&&cn===0&&(cn=3),a.flags&=-257,a.flags|=65536,a.lanes=l,s===Ql?a.flags|=16384:(n=a.updateQueue,n===null?a.updateQueue=new Set([s]):n.add(s),Cd(e,s,l)),!1;case 22:return a.flags|=65536,s===Ql?a.flags|=16384:(n=a.updateQueue,n===null?(n={transitions:null,markerInstances:null,retryQueue:new Set([s])},a.updateQueue=n):(a=n.retryQueue,a===null?n.retryQueue=new Set([s]):a.add(s)),Cd(e,s,l)),!1}throw Error(r(435,a.tag))}return Cd(e,s,l),Uc(),!1}if(xt)return n=Rn.current,n!==null?((n.flags&65536)===0&&(n.flags|=256),n.flags|=65536,n.lanes=l,s!==ff&&(e=Error(r(422),{cause:s}),Ao(gi(e,a)))):(s!==ff&&(n=Error(r(423),{cause:s}),Ao(gi(n,a))),e=e.current.alternate,e.flags|=65536,l&=-l,e.lanes|=l,s=gi(s,a),l=Wf(e.stateNode,s,l),yf(e,l),cn!==4&&(cn=2)),!1;var u=Error(r(520),{cause:s});if(u=gi(u,a),Wo===null?Wo=[u]:Wo.push(u),cn!==4&&(cn=2),n===null)return!0;s=gi(s,a),a=n;do{switch(a.tag){case 3:return a.flags|=65536,e=l&-l,a.lanes|=e,e=Wf(a.stateNode,s,e),yf(a,e),!1;case 1:if(n=a.type,u=a.stateNode,(a.flags&128)===0&&(typeof n.getDerivedStateFromError=="function"||u!==null&&typeof u.componentDidCatch=="function"&&(ns===null||!ns.has(u))))return a.flags|=65536,l&=-l,a.lanes|=l,l=Jg(l),$g(l,e,a,s),yf(a,l),!1;break;case 22:if(a.memoizedState!==null)return a.flags|=65536,!1}a=a.return}while(a!==null);return!1}var qf=Error(r(461)),mn=!1;function xn(e,n,a,s){n.child=e===null?ig(n,null,a,s):Ps(n,e.child,a,s)}function e0(e,n,a,s,l){a=a.render;var u=n.ref;if("ref"in s){var p={};for(var A in s)A!=="ref"&&(p[A]=s[A])}else p=s;return Ns(n),s=wf(e,n,a,p,u,l),A=Cf(),e!==null&&!mn?(Nf(e,n,l),Ma(e,n,l)):(xt&&A&&Xl(n),n.flags|=1,xn(e,n,s,l),n.child)}function t0(e,n,a,s,l){if(e===null){var u=a.type;return typeof u=="function"&&!of(u)&&u.defaultProps===void 0&&a.compare===null?(n.tag=15,n.type=u,n0(e,n,u,s,l)):(e=Vl(a.type,null,s,n,n.mode,l),e.ref=n.ref,e.return=n,n.child=e)}if(u=e.child,!ed(e,l)){var p=u.memoizedProps;if(a=a.compare,a=a!==null?a:Eo,a(p,s)&&e.ref===n.ref)return Ma(e,n,l)}return n.flags|=1,e=ga(u,s),e.ref=n.ref,e.return=n,n.child=e}function n0(e,n,a,s,l){if(e!==null){var u=e.memoizedProps;if(Eo(u,s)&&e.ref===n.ref)if(mn=!1,n.pendingProps=s=u,ed(e,l))(e.flags&131072)!==0&&(mn=!0);else return n.lanes=e.lanes,Ma(e,n,l)}return Yf(e,n,a,s,l)}function i0(e,n,a,s){var l=s.children,u=e!==null?e.memoizedState:null;if(e===null&&n.stateNode===null&&(n.stateNode={_visibility:1,_pendingMarkers:null,_retryCache:null,_transitions:null}),s.mode==="hidden"){if((n.flags&128)!==0){if(u=u!==null?u.baseLanes|a:a,e!==null){for(s=n.child=e.child,l=0;s!==null;)l=l|s.lanes|s.childLanes,s=s.sibling;s=l&~u}else s=0,n.child=null;return a0(e,n,u,a,s)}if((a&536870912)!==0)n.memoizedState={baseLanes:0,cachePool:null},e!==null&&Zl(n,u!==null?u.cachePool:null),u!==null?rg(n,u):Ef(),og(n);else return s=n.lanes=536870912,a0(e,n,u!==null?u.baseLanes|a:a,a,s)}else u!==null?(Zl(n,u.cachePool),rg(n,u),Qa(),n.memoizedState=null):(e!==null&&Zl(n,null),Ef(),Qa());return xn(e,n,l,a),n.child}function zo(e,n){return e!==null&&e.tag===22||n.stateNode!==null||(n.stateNode={_visibility:1,_pendingMarkers:null,_retryCache:null,_transitions:null}),n.sibling}function a0(e,n,a,s,l){var u=_f();return u=u===null?null:{parent:hn._currentValue,pool:u},n.memoizedState={baseLanes:a,cachePool:u},e!==null&&Zl(n,null),Ef(),og(n),e!==null&&Cs(e,n,s,!0),n.childLanes=l,null}function dc(e,n){return n=hc({mode:n.mode,children:n.children},e.mode),n.ref=e.ref,e.child=n,n.return=e,n}function s0(e,n,a){return Ps(n,e.child,null,a),e=dc(n,n.pendingProps),e.flags|=2,ni(n),n.memoizedState=null,e}function hy(e,n,a){var s=n.pendingProps,l=(n.flags&128)!==0;if(n.flags&=-129,e===null){if(xt){if(s.mode==="hidden")return e=dc(n,s),n.lanes=536870912,e.memoizedState={baseLanes:0,cachePool:null},zo(null,e);if(Tf(n),(e=Jt)?(e=D_(e,xi),e=e!==null&&e.data==="&"?e:null,e!==null&&(n.memoizedState={dehydrated:e,treeContext:Ga!==null?{id:Vi,overflow:ki}:null,retryLane:536870912,hydrationErrors:null},a=Vm(e),a.return=n,n.child=a,Mn=n,Jt=null)):e=null,e===null)throw ka(n);return n.lanes=536870912,null}return dc(n,s)}var u=e.memoizedState;if(u!==null){var p=u.dehydrated;if(Tf(n),l)if(n.flags&256)n.flags&=-257,n=s0(e,n,a);else if(n.memoizedState!==null)n.child=e.child,n.flags|=128,n=null;else throw Error(r(558));else if(mn||Cs(e,n,a,!1),l=(a&e.childLanes)!==0,mn||l){if(Za.current===null){if(s=Kt,s!==null&&(p=ho(s,a),p!==0&&p!==u.retryLane))throw u.retryLane=p,Ts(e,p),jn(s,e,p),qf;Uc()}n=s0(e,n,a)}else e=u.treeContext,Jt=yi(p.nextSibling),Mn=n,xt=!0,Va=null,xi=!1,e!==null&&Wm(n,e),n=dc(n,s),n.flags|=134221824;return n}return e=ga(e.child,{mode:s.mode,children:s.children}),e.ref=n.ref,n.child=e,e.return=n,e}function Sr(e,n){var a=n.ref;if(a===null)e!==null&&e.ref!==null&&(n.flags|=4194816);else{if(typeof a!="function"&&typeof a!="object")throw Error(r(284));(e===null||e.ref!==a)&&(n.flags|=4194816)}}function Yf(e,n,a,s,l){return Ns(n),a=wf(e,n,a,s,void 0,l),s=Cf(),e!==null&&!mn?(Nf(e,n,l),Ma(e,n,l)):(xt&&s&&Xl(n),n.flags|=1,xn(e,n,a,l),n.child)}function r0(e,n,a,s,l,u){return Ns(n),n.updateQueue=null,a=cg(n,s,a,l),lg(e),s=Cf(),e!==null&&!mn?(Nf(e,n,u),Ma(e,n,u)):(xt&&s&&Xl(n),n.flags|=1,xn(e,n,a,u),n.child)}function o0(e,n,a,s,l){if(Ns(n),n.stateNode===null){var u=ur,p=a.contextType;typeof p=="object"&&p!==null&&(u=An(p)),u=new a(s,u),n.memoizedState=u.state!==null&&u.state!==void 0?u.state:null,u.updater=Xf,n.stateNode=u,u._reactInternals=n,u=n.stateNode,u.props=s,u.state=n.memoizedState,u.refs={},xf(n),p=a.contextType,u.context=typeof p=="object"&&p!==null?An(p):ur,u.state=n.memoizedState,p=a.getDerivedStateFromProps,typeof p=="function"&&(kf(n,a,p,s),u.state=n.memoizedState),typeof a.getDerivedStateFromProps=="function"||typeof u.getSnapshotBeforeUpdate=="function"||typeof u.UNSAFE_componentWillMount!="function"&&typeof u.componentWillMount!="function"||(p=u.state,typeof u.componentWillMount=="function"&&u.componentWillMount(),typeof u.UNSAFE_componentWillMount=="function"&&u.UNSAFE_componentWillMount(),p!==u.state&&Xf.enqueueReplaceState(u,u.state,null),Lo(n,s,u,l),Uo(),u.state=n.memoizedState),typeof u.componentDidMount=="function"&&(n.flags|=4194308),s=!0}else if(e===null){u=n.stateNode;var A=n.memoizedProps,B=Bs(a,A);u.props=B;var te=u.context,fe=a.contextType;p=ur,typeof fe=="object"&&fe!==null&&(p=An(fe));var ye=a.getDerivedStateFromProps;fe=typeof ye=="function"||typeof u.getSnapshotBeforeUpdate=="function",A=n.pendingProps!==A,fe||typeof u.UNSAFE_componentWillReceiveProps!="function"&&typeof u.componentWillReceiveProps!="function"||(A||te!==p)&&Yg(n,u,s,p),qa=!1;var J=n.memoizedState;u.state=J,Lo(n,s,u,l),Uo(),te=n.memoizedState,A||J!==te||qa?(typeof ye=="function"&&(kf(n,a,ye,s),te=n.memoizedState),(B=qa||qg(n,a,B,s,J,te,p))?(fe||typeof u.UNSAFE_componentWillMount!="function"&&typeof u.componentWillMount!="function"||(typeof u.componentWillMount=="function"&&u.componentWillMount(),typeof u.UNSAFE_componentWillMount=="function"&&u.UNSAFE_componentWillMount()),typeof u.componentDidMount=="function"&&(n.flags|=4194308)):(typeof u.componentDidMount=="function"&&(n.flags|=4194308),n.memoizedProps=s,n.memoizedState=te),u.props=s,u.state=te,u.context=p,s=B):(typeof u.componentDidMount=="function"&&(n.flags|=4194308),s=!1)}else{u=n.stateNode,Sf(e,n),p=n.memoizedProps,fe=Bs(a,p),u.props=fe,ye=n.pendingProps,J=u.context,te=a.contextType,B=ur,typeof te=="object"&&te!==null&&(B=An(te)),A=a.getDerivedStateFromProps,(te=typeof A=="function"||typeof u.getSnapshotBeforeUpdate=="function")||typeof u.UNSAFE_componentWillReceiveProps!="function"&&typeof u.componentWillReceiveProps!="function"||(p!==ye||J!==B)&&Yg(n,u,s,B),qa=!1,J=n.memoizedState,u.state=J,Lo(n,s,u,l),Uo();var oe=n.memoizedState;p!==ye||J!==oe||qa||e!==null&&e.dependencies!==null&&Yl(e.dependencies)?(typeof A=="function"&&(kf(n,a,A,s),oe=n.memoizedState),(fe=qa||qg(n,a,fe,s,J,oe,B)||e!==null&&e.dependencies!==null&&Yl(e.dependencies))?(te||typeof u.UNSAFE_componentWillUpdate!="function"&&typeof u.componentWillUpdate!="function"||(typeof u.componentWillUpdate=="function"&&u.componentWillUpdate(s,oe,B),typeof u.UNSAFE_componentWillUpdate=="function"&&u.UNSAFE_componentWillUpdate(s,oe,B)),typeof u.componentDidUpdate=="function"&&(n.flags|=4),typeof u.getSnapshotBeforeUpdate=="function"&&(n.flags|=1024)):(typeof u.componentDidUpdate!="function"||p===e.memoizedProps&&J===e.memoizedState||(n.flags|=4),typeof u.getSnapshotBeforeUpdate!="function"||p===e.memoizedProps&&J===e.memoizedState||(n.flags|=1024),n.memoizedProps=s,n.memoizedState=oe),u.props=s,u.state=oe,u.context=B,s=fe):(typeof u.componentDidUpdate!="function"||p===e.memoizedProps&&J===e.memoizedState||(n.flags|=4),typeof u.getSnapshotBeforeUpdate!="function"||p===e.memoizedProps&&J===e.memoizedState||(n.flags|=1024),s=!1)}return u=s,Sr(e,n),s=(n.flags&128)!==0,u||s?(u=n.stateNode,a=s&&typeof a.getDerivedStateFromError!="function"?null:u.render(),n.flags|=1,e!==null&&s?(n.child=Ps(n,e.child,null,l),n.child=Ps(n,null,a,l)):xn(e,n,a,l),n.memoizedState=u.state,e=n.child):e=Ma(e,n,l),e}function l0(e,n,a,s){return Rs(),n.flags|=256,xn(e,n,a,s),n.child}var jf={dehydrated:null,treeContext:null,retryLane:0,hydrationErrors:null};function Zf(e){return{baseLanes:e,cachePool:Qm()}}function Kf(e,n,a){return e=e!==null?e.childLanes&~a:0,n&&(e|=si),e}function c0(e,n,a){var s=n.pendingProps,l=!1,u=(n.flags&128)!==0,p;if((p=u)||(p=e!==null&&e.memoizedState===null?!1:(wn.current&2)!==0),p&&(l=!0,n.flags&=-129),p=(n.flags&32)!==0,n.flags&=-33,e===null){if(xt){if(l?Ka(n):Qa(),(e=Jt)?(e=D_(e,xi),e=e!==null&&e.data!=="&"?e:null,e!==null&&(n.memoizedState={dehydrated:e,treeContext:Ga!==null?{id:Vi,overflow:ki}:null,retryLane:536870912,hydrationErrors:null},a=Vm(e),a.return=n,n.child=a,Mn=n,Jt=null)):e=null,e===null)throw ka(n);return Zd(e)?n.lanes=32:n.lanes=536870912,null}return u=s.children,s=s.fallback,l?(Qa(),l=n.mode,u=hc({mode:"hidden",children:u},l),s=As(s,l,a,null),u.return=n,s.return=n,u.sibling=s,n.child=u,s=n.child,s.memoizedState=Zf(a),s.childLanes=Kf(e,p,a),n.memoizedState=jf,zo(null,s)):(Ka(n),Qf(n,u))}var A=e.memoizedState;if(A!==null){var B=A.dehydrated;if(B!==null)return py(e,n,u,p,s,B,A,a)}return l?(Qa(),l=s.fallback,u=n.mode,A=e.child,B=A.sibling,s=ga(A,{mode:"hidden",children:s.children}),s.subtreeFlags=A.subtreeFlags&1206910976,B!==null?l=ga(B,l):(l=As(l,u,a,null),l.flags|=2),l.return=n,s.return=n,s.sibling=l,n.child=s,zo(null,s),s=n.child,l=e.child.memoizedState,l===null?l=Zf(a):(u=l.cachePool,u!==null?(A=hn._currentValue,u=u.parent!==A?{parent:A,pool:A}:u):u=Qm(),l={baseLanes:l.baseLanes|a,cachePool:u}),s.memoizedState=l,s.childLanes=Kf(e,p,a),n.memoizedState=jf,zo(e.child,s)):(Ka(n),a=e.child,e=a.sibling,a=ga(a,{mode:"visible",children:s.children}),a.return=n,a.sibling=null,e!==null&&(p=n.deletions,p===null?(n.deletions=[e],n.flags|=16):p.push(e)),n.child=a,n.memoizedState=null,a)}function Qf(e,n){return n=hc({mode:"visible",children:n},e.mode),n.return=e,e.child=n}function hc(e,n){return e=Xn(22,e,null,n),e.lanes=0,e}function pc(e,n,a){return Ps(n,e.child,null,a),e=Qf(n,n.pendingProps.children),e.flags|=2,n.memoizedState=null,e}function py(e,n,a,s,l,u,p,A){if(a)return n.flags&256?(Ka(n),n.flags&=-257,pc(e,n,A)):n.memoizedState!==null?(Qa(),n.child=e.child,n.flags|=128,null):(Qa(),u=l.fallback,p=n.mode,l=hc({mode:"visible",children:l.children},p),u=As(u,p,A,null),u.flags|=2,l.return=n,u.return=n,l.sibling=u,n.child=l,Ps(n,e.child,null,A),l=n.child,l.memoizedState=Zf(A),l.childLanes=Kf(e,s,A),n.memoizedState=jf,zo(null,l));if(Ka(n),Zd(u)){if(s=u.nextSibling&&u.nextSibling.dataset,s)var B=s.dgst;return s=B,s!==""&&(l=Error(r(419)),l.stack="",l.digest=s,Ao({value:l,source:null,stack:null})),pc(e,n,A)}if(mn||Cs(e,n,A,!1),s=(A&e.childLanes)!==0,mn||s){if(Za.current!==null)return pc(e,n,A);if(s=Kt,s!==null&&(l=ho(s,A),l!==0&&l!==p.retryLane))throw p.retryLane=l,Ts(e,l),jn(s,e,l),qf;return jd(u)||Uc(),pc(e,n,A)}return jd(u)?(n.flags|=192,n.child=e.child,null):(e=p.treeContext,Jt=yi(u.nextSibling),Mn=n,xt=!0,Va=null,xi=!1,e!==null&&Wm(n,e),n=Qf(n,l.children),n.flags|=134221824,n)}function u0(e,n,a){e.lanes|=n;var s=e.alternate;s!==null&&(s.lanes|=n),ql(e.return,n,a)}function f0(e){for(var n=null;e!==null;){var a=e.alternate;a!==null&&tc(a)===null&&(n=e),e=e.sibling}return n}function mc(e,n,a,s,l,u){var p=e.memoizedState;p===null?e.memoizedState={isBackwards:n,rendering:null,renderingStartTime:0,last:s,tail:a,tailMode:l,treeForkCount:u}:(p.isBackwards=n,p.rendering=null,p.renderingStartTime=0,p.last=s,p.tail=a,p.tailMode=l,p.treeForkCount=u)}function Jf(e){var n=e.child;for(e.child=null;n!==null;){var a=n.sibling;n.sibling=e.child,e.child=n,n=a}}function $f(e,n,a){var s=n.pendingProps,l=s.revealOrder,u=s.tail;s=s.children;var p=wn.current;if(n.flags&128)return Oo(n,p),null;var A=(p&2)!==0;if(A?(p=p&1|2,n.flags|=128):p&=1,Oo(n,p),l==="backwards"&&e!==null?(Jf(e),xn(e,n,s,a),Jf(e)):xn(e,n,s,a),s=xt?To:0,!A&&e!==null&&(e.flags&128)!==0)e:for(e=n.child;e!==null;){if(e.tag===13)e.memoizedState!==null&&u0(e,a,n);else if(e.tag===19)u0(e,a,n);else if(e.child!==null){e.child.return=e,e=e.child;continue}if(e===n)break e;for(;e.sibling===null;){if(e.return===null||e.return===n)break e;e=e.return}e.sibling.return=e.return,e=e.sibling}switch(l){case"backwards":a=f0(n.child),a===null?(l=n.child,n.child=null):(l=a.sibling,a.sibling=null,Jf(n)),mc(n,!0,l,null,u,s);break;case"unstable_legacy-backwards":for(a=null,l=n.child,n.child=null;l!==null;){if(e=l.alternate,e!==null&&tc(e)===null){n.child=l;break}e=l.sibling,l.sibling=a,a=l,l=e}mc(n,!0,a,null,u,s);break;case"together":mc(n,!1,null,null,void 0,s);break;case"independent":n.memoizedState=null;break;default:a=f0(n.child),a===null?(l=n.child,n.child=null):(l=a.sibling,a.sibling=null),mc(n,!1,l,a,u,s)}return n.child}function d0(e,n,a){var s=n.pendingProps;return Xa(n,n.type,s.value),xn(e,n,s.children,a),n.child}function Ma(e,n,a){if(e!==null&&(n.dependencies=e.dependencies),ts|=n.lanes,(a&n.childLanes)===0)if(e!==null){if(Cs(e,n,a,!1),(a&n.childLanes)===0)return null}else return null;if(e!==null&&n.child!==e.child)throw Error(r(153));if(n.child!==null){for(e=n.child,a=ga(e,e.pendingProps),n.child=a,a.return=n;e.sibling!==null;)e=e.sibling,a=a.sibling=ga(e,e.pendingProps),a.return=n;a.sibling=null}return n.child}function ed(e,n){return(e.lanes&n)!==0?!0:(e=e.dependencies,!!(e!==null&&Yl(e)))}function my(e,n,a){switch(n.tag){case 3:K(n,n.stateNode.containerInfo),Xa(n,hn,e.memoizedState.cache),Rs();break;case 27:case 5:Ot(n);break;case 4:K(n,n.stateNode.containerInfo);break;case 10:Xa(n,n.type,n.memoizedProps.value);break;case 31:if(n.memoizedState!==null)return n.flags|=128,Tf(n),null;break;case 13:var s=n.memoizedState;if(s!==null){if(s.dehydrated!==null)return Ka(n),n.flags|=128,null;s=Cs(e,n,a,!1);var l=n.child.childLanes;return s||(a&l)!==0?c0(e,n,a):(Ka(n),e=Ma(e,n,a),e!==null?e.sibling:null)}Ka(n);break;case 19:if(n.flags&128)return $f(e,n,a);if(l=(e.flags&128)!==0,s=(a&n.childLanes)!==0,s||(Cs(e,n,a,!1),s=(a&n.childLanes)!==0),l){if(s)return $f(e,n,a);n.flags|=128}if(l=n.memoizedState,l!==null&&(l.rendering=null,l.tail=null,l.lastEffect=null),Oo(n,wn.current),s)break;return null;case 22:return n.lanes=0,i0(e,n,a,n.pendingProps);case 24:Xa(n,hn,e.memoizedState.cache)}return Ma(e,n,a)}function h0(e,n,a){if(e!==null)if(e.memoizedProps!==n.pendingProps)mn=!0;else{if(!ed(e,a)&&(n.flags&128)===0)return mn=!1,my(e,n,a);mn=(e.flags&131072)!==0}else mn=!1,xt&&(n.flags&1048576)!==0&&Xm(n,To,n.index);switch(n.lanes=0,n.tag){case 16:e:{var s=n.pendingProps;if(e=Ls(n.elementType),n.type=e,typeof e=="function")of(e)?(s=Bs(e,s),n.tag=1,n=o0(null,n,e,s,a)):(n.tag=0,n=Yf(null,n,e,s,a));else{if(e!=null){var l=e.$$typeof;if(l===j){n.tag=11,n=e0(null,n,e,s,a);break e}else if(l===ce){n.tag=14,n=t0(null,n,e,s,a);break e}else if(l===ne){n.tag=10,n.type=e,n=d0(null,n,a);break e}}throw n=Ae(e)||e,Error(r(306,n,""))}}return n;case 0:return Yf(e,n,n.type,n.pendingProps,a);case 1:return s=n.type,l=Bs(s,n.pendingProps),o0(e,n,s,l,a);case 3:e:{if(K(n,n.stateNode.containerInfo),e===null)throw Error(r(387));s=n.pendingProps;var u=n.memoizedState;l=u.element,Sf(e,n),Lo(n,s,null,a);var p=n.memoizedState;if(s=p.cache,Xa(n,hn,s),s!==u.cache&&pf(n,[hn],a,!0),Uo(),s=p.element,u.isDehydrated)if(u={element:s,isDehydrated:!1,cache:p.cache},n.updateQueue.baseState=u,n.memoizedState=u,n.flags&256){n=l0(e,n,s,a);break e}else if(s!==l){l=gi(Error(r(424)),n),Ao(l),n=l0(e,n,s,a);break e}else{switch(e=n.stateNode.containerInfo,e.nodeType){case 9:e=e.body;break;default:e=e.nodeName==="HTML"?e.ownerDocument.body:e}for(Jt=yi(e.firstChild),Mn=n,xt=!0,Va=null,xi=!0,a=ig(n,null,s,a),n.child=a;a;)a.flags=a.flags&-3|134221824,a=a.sibling}else{if(Rs(),s===l){n=Ma(e,n,a);break e}xn(e,n,s,a)}n=n.child}return n;case 26:return Sr(e,n),e===null?(a=z_(n.type,null,n.pendingProps,null))?n.memoizedState=a:xt||(n.stateNode=__(n.type,n.pendingProps,Pt.current,n)):n.memoizedState=z_(n.type,e.memoizedProps,n.pendingProps,e.memoizedState),null;case 27:return Ot(n),e===null&&xt&&(s=n.stateNode=O_(n.type,n.pendingProps,Pt.current),Mn=n,xi=!0,l=Jt,ss(n.type)?(Kd=l,Jt=yi(s.firstChild)):Jt=l),xn(e,n,n.pendingProps.children,a),Sr(e,n),e===null&&(n.flags|=4194304),n.child;case 5:return e===null&&xt&&((l=s=Jt)&&(s=cM(s,n.type,n.pendingProps,xi),s!==null?(n.stateNode=s,Mn=n,Jt=yi(s.firstChild),xi=!1,l=!0):l=!1),l||ka(n)),Ot(n),l=n.type,u=n.pendingProps,p=e!==null?e.memoizedProps:null,s=u.children,Gd(l,u)?s=null:p!==null&&Gd(l,p)&&(n.flags|=32),n.memoizedState!==null&&(l=wf(e,n,ay,null,null,a),zr._currentValue=l),Sr(e,n),xn(e,n,s,a),n.child;case 6:return e===null&&xt&&((e=a=Jt)&&(a=uM(a,n.pendingProps,xi),a!==null?(n.stateNode=a,Mn=n,Jt=null,e=!0):e=!1),e||ka(n)),null;case 13:return c0(e,n,a);case 4:return K(n,n.stateNode.containerInfo),s=n.pendingProps,e===null?n.child=Ps(n,null,s,a):xn(e,n,s,a),n.child;case 11:return e0(e,n,n.type,n.pendingProps,a);case 7:return s=n.pendingProps,Sr(e,n),xn(e,n,s,a),n.child;case 8:return xn(e,n,n.pendingProps.children,a),n.child;case 12:return xn(e,n,n.pendingProps.children,a),n.child;case 10:return d0(e,n,a);case 9:return l=n.type._context,s=n.pendingProps.children,Ns(n),l=An(l),s=s(l),n.flags|=1,xn(e,n,s,a),n.child;case 14:return t0(e,n,n.type,n.pendingProps,a);case 15:return n0(e,n,n.type,n.pendingProps,a);case 19:return $f(e,n,a);case 31:return hy(e,n,a);case 22:return i0(e,n,a,n.pendingProps);case 24:return Ns(n),s=An(hn),e===null?(l=_f(),l===null&&(l=Kt,u=mf(),l.pooledCache=u,u.refCount++,u!==null&&(l.pooledCacheLanes|=a),l=u),n.memoizedState={parent:s,cache:l},xf(n),Xa(n,hn,l)):((e.lanes&a)!==0&&(Sf(e,n),Lo(n,null,null,a),Uo()),l=e.memoizedState,u=n.memoizedState,l.parent!==s?(l={parent:s,cache:s},n.memoizedState=l,n.lanes===0&&(n.memoizedState=n.updateQueue.baseState=l),Xa(n,hn,s)):(s=u.cache,Xa(n,hn,s),s!==l.cache&&pf(n,[hn],a,!0))),xn(e,n,n.pendingProps.children,a),n.child;case 30:return n.stateNode===null&&(n.stateNode={autoName:null,paired:null,clones:null,ref:null}),s=n.pendingProps,s.name!=null&&s.name!=="auto"?n.flags|=e===null?18882560:18874368:xt&&Xl(n),e!==null&&e.memoizedProps.name!==s.name?n.flags|=4194816:Sr(e,n),xn(e,n,s.children,a),n.child;case 29:throw n.pendingProps}throw Error(r(156,n.tag))}function Ea(e){e.flags|=4}function td(e,n,a,s,l){var u;if((u=(e.mode&32)!==0)&&(u=a===null?V_(n,s):V_(n,s)&&(s.src!==a.src||s.srcSet!==a.srcSet)),u){if(e.flags|=16777216,(l&335544128)===l)if(e.stateNode.complete)e.flags|=8192;else if(j0())e.flags|=8192;else throw Os=Ql,vf}else e.flags&=-16777217}function p0(e,n){if(n.type!=="stylesheet"||(n.state.loading&4)!==0)e.flags&=-16777217;else if(e.flags|=16777216,!k_(n))if(j0())e.flags|=8192;else throw Os=Ql,vf}function gc(e,n){n!==null&&(e.flags|=4),e.flags&16384&&(n=e.tag!==22?uo():536870912,e.lanes|=n,Tr|=n)}function Fo(e,n){if(!xt)switch(e.tailMode){case"visible":break;case"collapsed":for(var a=e.tail,s=null;a!==null;)a.alternate!==null&&(s=a),a=a.sibling;s===null?n||e.tail===null?e.tail=null:e.tail.sibling=null:s.sibling=null;break;default:for(n=e.tail,a=null;n!==null;)n.alternate!==null&&(a=n),n=n.sibling;a===null?e.tail=null:a.sibling=null}}function $t(e){var n=e.alternate!==null&&e.alternate.child===e.child,a=0,s=0;if(n)for(var l=e.child;l!==null;)a|=l.lanes|l.childLanes,s|=l.subtreeFlags&1206910976,s|=l.flags&1206910976,l.return=e,l=l.sibling;else for(l=e.child;l!==null;)a|=l.lanes|l.childLanes,s|=l.subtreeFlags,s|=l.flags,l.return=e,l=l.sibling;return e.subtreeFlags|=s,e.childLanes=a,n}function gy(e,n,a){var s=n.pendingProps;switch(uf(n),n.tag){case 16:case 15:case 0:case 11:case 7:case 8:case 12:case 9:case 14:return $t(n),null;case 1:return $t(n),null;case 3:return a=n.stateNode,s=null,e!==null&&(s=e.memoizedState.cache),n.memoizedState.cache!==s&&(n.flags|=2048),xa(hn),nn(),a.pendingContext&&(a.context=a.pendingContext,a.pendingContext=null),(e===null||e.child===null)&&(hr(n)?Ea(n):e===null||e.memoizedState.isDehydrated&&(n.flags&256)===0||(n.flags|=1024,df())),$t(n),null;case 26:var l=n.type,u=n.memoizedState;return e===null?(Ea(n),u!==null?($t(n),p0(n,u)):($t(n),td(n,l,null,s,a))):u?u!==e.memoizedState?(Ea(n),$t(n),p0(n,u)):($t(n),n.flags&=-16777217):(e=e.memoizedProps,e!==s&&Ea(n),$t(n),td(n,l,e,s,a)),null;case 27:if(P(n),a=Pt.current,l=n.type,e!==null&&n.stateNode!=null)e.memoizedProps!==s&&Ea(n);else{if(!s){if(n.stateNode===null)throw Error(r(166));return $t(n),n.subtreeFlags&=-33554433,null}e=kt.current,hr(n)?qm(n):(e=O_(l,s,a),n.stateNode=e,Ea(n))}return $t(n),n.subtreeFlags&=-33554433,null;case 5:if(P(n),l=n.type,e!==null&&n.stateNode!=null)e.memoizedProps!==s&&Ea(n);else{if(!s){if(n.stateNode===null)throw Error(r(166));return $t(n),n.subtreeFlags&=-33554433,null}if(u=kt.current,hr(n))qm(n);else{var p=Ko(Pt.current);switch(u){case 1:u=p.createElementNS("http://www.w3.org/2000/svg",l);break;case 2:u=p.createElementNS("http://www.w3.org/1998/Math/MathML",l);break;default:switch(l){case"svg":u=p.createElementNS("http://www.w3.org/2000/svg",l);break;case"math":u=p.createElementNS("http://www.w3.org/1998/Math/MathML",l);break;case"script":u=p.createElement("div"),u.innerHTML="<script><\/script>",u=u.removeChild(u.firstChild);break;case"select":u=typeof s.is=="string"?p.createElement("select",{is:s.is}):p.createElement("select"),s.multiple?u.multiple=!0:s.size&&(u.size=s.size);break;default:u=typeof s.is=="string"?p.createElement(l,{is:s.is}):p.createElement(l)}}u[T]=n,u[G]=s;e:for(p=n.child;p!==null;){if(p.tag===5||p.tag===6)u.appendChild(p.stateNode);else if(p.tag!==4&&p.tag!==27&&p.child!==null){p.child.return=p,p=p.child;continue}if(p===n)break e;for(;p.sibling===null;){if(p.return===null||p.return===n)break e;p=p.return}p.sibling.return=p.return,p=p.sibling}n.stateNode=u;e:switch(Nn(u,l,s),l){case"button":case"input":case"select":case"textarea":s=!!s.autoFocus;break e;case"img":s=!0;break e;default:s=!1}s&&Ea(n)}}return $t(n),n.subtreeFlags&=-33554433,td(n,n.type,e===null?null:e.memoizedProps,n.pendingProps,a),null;case 6:if(e&&n.stateNode!=null)e.memoizedProps!==s&&Ea(n);else{if(typeof s!="string"&&n.stateNode===null)throw Error(r(166));if(e=Pt.current,hr(n)){if(e=n.stateNode,a=n.memoizedProps,s=null,l=Mn,l!==null)switch(l.tag){case 27:case 5:s=l.memoizedProps}e[T]=n,e=!!(e.nodeValue===a||s!==null&&s.suppressHydrationWarning===!0||h_(e.nodeValue,a)),e||ka(n,!0)}else e=Ko(e).createTextNode(s),e[T]=n,n.stateNode=e}return $t(n),null;case 31:if(a=n.memoizedState,e===null||e.memoizedState!==null){if(s=hr(n),a!==null){if(e===null){if(!s)throw Error(r(318));if(e=n.memoizedState,e=e!==null?e.dehydrated:null,!e)throw Error(r(557));e[T]=n}else Rs(),(n.flags&128)===0&&(n.memoizedState=null),n.flags|=4;$t(n),e=!1}else a=df(),e!==null&&e.memoizedState!==null&&(e.memoizedState.hydrationErrors=a),e=!0;if(!e)return n.flags&256?(ni(n),n):(ni(n),null);if((n.flags&128)!==0)throw Error(r(558))}return $t(n),null;case 13:if(s=n.memoizedState,e===null||e.memoizedState!==null&&e.memoizedState.dehydrated!==null){if(l=hr(n),s!==null&&s.dehydrated!==null){if(e===null){if(!l)throw Error(r(318));if(l=n.memoizedState,l=l!==null?l.dehydrated:null,!l)throw Error(r(317));l[T]=n}else Rs(),(n.flags&128)===0&&(n.memoizedState=null),n.flags|=4;$t(n),l=!1}else l=df(),e!==null&&e.memoizedState!==null&&(e.memoizedState.hydrationErrors=l),l=!0;if(!l)return n.flags&256?(ni(n),n):(ni(n),null)}return ni(n),(n.flags&128)!==0?(n.lanes=a,n):(a=s!==null,e=e!==null&&e.memoizedState!==null,a&&(s=n.child,l=null,s.alternate!==null&&s.alternate.memoizedState!==null&&s.alternate.memoizedState.cachePool!==null&&(l=s.alternate.memoizedState.cachePool.pool),u=null,s.memoizedState!==null&&s.memoizedState.cachePool!==null&&(u=s.memoizedState.cachePool.pool),u!==l&&(s.flags|=2048)),a!==e&&a&&(n.child.flags|=8192),gc(n,n.updateQueue),$t(n),null);case 4:return nn(),e===null&&Id(n.stateNode.containerInfo),n.flags|=67108864,$t(n),null;case 10:return xa(n.type),$t(n),null;case 19:if(Af(n),s=n.memoizedState,s===null)return $t(n),null;if(l=(n.flags&128)!==0,u=s.rendering,u===null)if(l)Fo(s,!1);else{if(cn!==0||e!==null&&(e.flags&128)!==0)for(e=n.child;e!==null;){if(u=tc(e),u!==null){for(n.flags|=128,Fo(s,!1),e=u.updateQueue,n.updateQueue=e,gc(n,e),n.subtreeFlags=0,e=a,a=n.child;a!==null;)Gm(a,e),a=a.sibling;return Oo(n,wn.current&1|2),xt&&_a(n,s.treeForkCount),n.child}e=e.sibling}s.tail!==null&&ke()>wc&&(n.flags|=128,l=!0,Fo(s,!1),n.lanes=4194304)}else{if(!l)if(e=tc(u),e!==null){if(n.flags|=128,l=!0,e=e.updateQueue,n.updateQueue=e,gc(n,e),Fo(s,!0),s.tail===null&&s.tailMode!=="collapsed"&&s.tailMode!=="visible"&&!u.alternate&&!xt)return $t(n),null}else 2*ke()-s.renderingStartTime>wc&&a!==536870912&&(n.flags|=128,l=!0,Fo(s,!1),n.lanes=4194304);s.isBackwards?(u.sibling=n.child,n.child=u):(e=s.last,e!==null?e.sibling=u:n.child=u,s.last=u)}if(s.tail!==null){e=s.tail;e:{for(a=e;a!==null;){if(a.alternate!==null){a=!1;break e}a=a.sibling}a=!0}return s.rendering=e,s.tail=e.sibling,s.renderingStartTime=ke(),e.sibling=null,u=wn.current,u=l?u&1|2:u&1,s.tailMode==="visible"||s.tailMode==="collapsed"||!a||xt?Oo(n,u):(a=u,it(Rn,n),it(wn,a),Ln===null&&(Ln=n)),xt&&_a(n,s.treeForkCount),e}return $t(n),null;case 22:case 23:return ni(n),bf(),s=n.memoizedState!==null,e!==null?e.memoizedState!==null!==s&&(n.flags|=8192):s&&(n.flags|=8192),s?(a&536870912)!==0&&(n.flags&128)===0&&($t(n),n.subtreeFlags&6&&(n.flags|=8192)):$t(n),a=n.updateQueue,a!==null&&gc(n,a.retryQueue),a=null,e!==null&&e.memoizedState!==null&&e.memoizedState.cachePool!==null&&(a=e.memoizedState.cachePool.pool),s=null,n.memoizedState!==null&&n.memoizedState.cachePool!==null&&(s=n.memoizedState.cachePool.pool),s!==a&&(n.flags|=2048),e!==null&&tt(Us),null;case 24:return a=null,e!==null&&(a=e.memoizedState.cache),n.memoizedState.cache!==a&&(n.flags|=2048),xa(hn),$t(n),null;case 25:return null;case 30:return n.flags|=33554432,$t(n),null}throw Error(r(156,n.tag))}function _y(e,n){switch(uf(n),n.tag){case 1:return e=n.flags,e&65536?(n.flags=e&-65537|128,n):null;case 3:return xa(hn),nn(),e=n.flags,(e&65536)!==0&&(e&128)===0?(n.flags=e&-65537|128,n):null;case 26:case 27:case 5:return P(n),null;case 31:if(n.memoizedState!==null){if(ni(n),n.alternate===null)throw Error(r(340));Rs()}return e=n.flags,e&65536?(n.flags=e&-65537|128,n):null;case 13:if(ni(n),e=n.memoizedState,e!==null&&e.dehydrated!==null){if(n.alternate===null)throw Error(r(340));Rs()}return e=n.flags,e&65536?(n.flags=e&-65537|128,n):null;case 19:return Af(n),e=n.flags,e&65536?(n.flags=e&-65537|128,e=n.memoizedState,e!==null&&(e.rendering=null,e.tail=null),n.flags|=4,n):null;case 4:return nn(),null;case 10:return xa(n.type),null;case 22:case 23:return ni(n),bf(),e!==null&&tt(Us),e=n.flags,e&65536?(n.flags=e&-65537|128,n):null;case 24:return xa(hn),null;case 25:return null;default:return null}}function m0(e,n){switch(uf(n),n.tag){case 3:xa(hn),nn();break;case 26:case 27:case 5:P(n);break;case 4:nn();break;case 31:n.memoizedState!==null&&ni(n);break;case 13:ni(n);break;case 19:Af(n);break;case 10:xa(n.type);break;case 22:case 23:ni(n),bf(),e!==null&&tt(Us);break;case 24:xa(hn)}}function Ho(e,n){try{var a=n.updateQueue,s=a!==null?a.lastEffect:null;if(s!==null){var l=s.next;a=l;do{if((a.tag&e)===e){s=void 0;var u=a.create,p=a.inst;s=u(),p.destroy=s}a=a.next}while(a!==l)}}catch(A){Wt(n,n.return,A)}}function Ja(e,n,a){try{var s=n.updateQueue,l=s!==null?s.lastEffect:null;if(l!==null){var u=l.next;s=u;do{if((s.tag&e)===e){var p=s.inst,A=p.destroy;if(A!==void 0){p.destroy=void 0,l=n;var B=a,te=A;try{te()}catch(fe){Wt(l,B,fe)}}}s=s.next}while(s!==u)}}catch(fe){Wt(n,n.return,fe)}}function g0(e){var n=e.updateQueue;if(n!==null){var a=e.stateNode;try{sg(n,a)}catch(s){Wt(e,e.return,s)}}}function _0(e,n,a){a.props=Bs(e.type,e.memoizedProps),a.state=e.memoizedState;try{a.componentWillUnmount()}catch(s){Wt(e,n,s)}}function Xi(e,n){try{var a=e.ref;if(a!==null){switch(e.tag){case 26:case 27:case 5:var s=e.stateNode;break;case 30:var l=e.stateNode,u=pa(e.memoizedProps,l);(l.ref===null||l.ref.name!==u)&&(l.ref=b_(u)),s=l.ref;break;case 7:if(e.stateNode===null){var p=new oi(e);_(e.child,!1,oM,p,void 0,void 0),e.stateNode=p}s=e.stateNode;break;default:s=e.stateNode}typeof a=="function"?e.refCleanup=a(s):a.current=s}}catch(A){Wt(e,n,A)}}function Cn(e,n){var a=e.ref,s=e.refCleanup;if(a!==null)if(typeof s=="function")try{s()}catch(l){Wt(e,n,l)}finally{e.refCleanup=null,e=e.alternate,e!=null&&(e.refCleanup=null)}else if(typeof a=="function")try{a(null)}catch(l){Wt(e,n,l)}else a.current=null}function _c(e,n){if((e.tag===5||e.tag===27||e.tag===6)&&e.alternate===null&&n!==null)for(var a=0;a<n.length;a++)N_(e.stateNode,n[a])}function v0(e){for(var n=e.return;n!==null&&(id(n)&&N_(e.stateNode,n.stateNode),!nd(n));)n=n.return}function Go(e){for(var n=e.return;n!==null&&(id(n)&&lM(e.stateNode,n.stateNode),!nd(n));)n=n.return}function nd(e){return e.tag===5||e.tag===3||e.tag===27}function id(e){return e&&e.tag===7&&e.stateNode!==null}function ad(e){var n=e.type,a=e.memoizedProps,s=e.stateNode;try{e:switch(n){case"button":case"input":case"select":case"textarea":a.autoFocus&&s.focus();break e;case"img":a.src?s.src=a.src:a.srcSet&&(s.srcset=a.srcSet)}}catch(l){Wt(e,e.return,l)}}function sd(e,n,a){try{var s=e.stateNode;ky(s,e.type,a,n),s[G]=n}catch(l){Wt(e,e.return,l)}}function x0(e){return e.tag===5||e.tag===3||e.tag===26||e.tag===27&&ss(e.type)||e.tag===4}function rd(e){e:for(;;){for(;e.sibling===null;){if(e.return===null||x0(e.return))return null;e=e.return}for(e.sibling.return=e.return,e=e.sibling;e.tag!==5&&e.tag!==6&&e.tag!==18;){if(e.tag===27&&ss(e.type)||e.flags&2||e.child===null||e.tag===4)continue e;e.child.return=e,e=e.child}if(!(e.flags&2))return e.stateNode}}function od(e,n,a,s){var l=e.tag;if(l===5||l===6)l=e.stateNode,n?(a.nodeType===9?a.body:a.nodeName==="HTML"?a.ownerDocument.body:a).insertBefore(l,n):(n=a.nodeType===9?a.body:a.nodeName==="HTML"?a.ownerDocument.body:a,n.appendChild(l),a=a._reactRootContainer,a!=null||n.onclick!==null||(n.onclick=Gi)),_c(e,s),vt=!0;else if(l!==4&&(l===27&&(_c(e,s),s=null,ss(e.type)&&(a=e.stateNode,n=null)),e=e.child,e!==null))for(od(e,n,a,s),e=e.sibling;e!==null;)od(e,n,a,s),e=e.sibling}function vc(e,n,a,s){var l=e.tag;if(l===5||l===6)l=e.stateNode,n?a.insertBefore(l,n):a.appendChild(l),_c(e,s),vt=!0;else if(l!==4&&(l===27&&(_c(e,s),s=null,ss(e.type)&&(a=e.stateNode)),e=e.child,e!==null))for(vc(e,n,a,s),e=e.sibling;e!==null;)vc(e,n,a,s),e=e.sibling}function S0(e){var n=e.stateNode,a=e.memoizedProps;try{for(var s=e.type,l=n.attributes;l.length;)n.removeAttributeNode(l[0]);Nn(n,s,a),n[T]=e,n[G]=a}catch(u){Wt(e,e.return,u)}}var xc=!1,ii=null;function y0(e){(e.tag===30||(e.subtreeFlags&33554432)!==0)&&(xc=!0)}var Wi=null;function M0(){var e=Wi;return Wi=null,e}var Wn=0;function yr(e,n,a,s,l){return Wn=0,E0(e.child,n,a,s,l)}function E0(e,n,a,s,l){for(var u=!1;e!==null;){if(e.tag===5){var p=e.stateNode;if(s!==null){var A=Xd(p);s.push(A),A.view&&(u=!0)}else u||Xd(p).view&&(u=!0);xc=!0,M_(p,Wn===0?n:n+"_"+Wn,a),Wn++}else(e.tag!==22||e.memoizedState===null)&&(e.tag===30&&l||E0(e.child,n,a,s,l)&&(u=!0));e=e.sibling}return u}function qi(e,n){for(;e!==null;)e.tag===5?E_(e.stateNode,e.memoizedProps):(e.tag!==22||e.memoizedState===null)&&(e.tag===30&&n||qi(e.child,n)),e=e.sibling}function Sc(e){if((e.subtreeFlags&18874368)!==0)for(e=e.child;e!==null;){if((e.tag!==22||e.memoizedState===null)&&(Sc(e),e.tag===30&&(e.flags&18874368)!==0&&e.stateNode.paired)){var n=e.memoizedProps;if(n.name==null||n.name==="auto")throw Error(r(544));var a=n.name;n=ma(n.default,n.share),n!=="none"&&(yr(e,a,n,null,!1)||qi(e.child,!1))}e=e.sibling}}function ld(e,n){if(e.tag===30){var a=e.stateNode,s=e.memoizedProps,l=pa(s,a),u=ma(s.default,a.paired?s.share:s.enter);u!=="none"?yr(e,l,u,null,!1)?(Sc(e),a.paired||n||Cr(e,s.onEnter)):qi(e.child,!1):Sc(e)}else if((e.subtreeFlags&33554432)!==0)for(e=e.child;e!==null;)ld(e,n),e=e.sibling;else Sc(e)}function cd(e){if(ii!==null&&ii.size!==0){var n=ii;if((e.subtreeFlags&18874368)!==0)for(e=e.child;e!==null;){if(e.tag!==22||e.memoizedState===null){if(e.tag===30&&(e.flags&18874368)!==0){var a=e.memoizedProps,s=a.name;if(s!=null&&s!=="auto"){var l=n.get(s);if(l!==void 0){var u=ma(a.default,a.share);if(u!=="none"&&(yr(e,s,u,null,!1)?(u=e.stateNode,l.paired=u,u.paired=l,Cr(e,a.onShare)):qi(e.child,!1)),n.delete(s),n.size===0)break}}}cd(e)}e=e.sibling}}}function ud(e){if(e.tag===30){var n=e.memoizedProps,a=pa(n,e.stateNode),s=ii!==null?ii.get(a):void 0,l=ma(n.default,s!==void 0?n.share:n.exit);l!=="none"&&(yr(e,a,l,null,!1)?s!==void 0?(l=e.stateNode,s.paired=l,l.paired=s,ii.delete(a),Cr(e,n.onShare)):Cr(e,n.onExit):qi(e.child,!1)),ii!==null&&cd(e)}else if((e.subtreeFlags&33554432)!==0)for(e=e.child;e!==null;)ud(e),e=e.sibling;else ii!==null&&cd(e)}function b0(e){for(e=e.child;e!==null;){if(e.tag===30){var n=e.memoizedProps,a=pa(n,e.stateNode);n=ma(n.default,n.update),e.flags&=-5,n!=="none"&&yr(e,a,n,e.memoizedState=[],!1)}else(e.subtreeFlags&33554432)!==0&&b0(e);e=e.sibling}}function fd(e){if((e.subtreeFlags&18874368)!==0)for(e=e.child;e!==null;){if(e.tag!==22||e.memoizedState===null){if(e.tag===30&&(e.flags&18874368)!==0){var n=e.stateNode;n.paired!==null&&(n.paired=null,qi(e.child,!1))}fd(e)}e=e.sibling}}function yc(e){if(e.tag===30)e.stateNode.paired=null,qi(e.child,!1),fd(e);else if((e.subtreeFlags&33554432)!==0)for(e=e.child;e!==null;)yc(e),e=e.sibling;else fd(e)}function T0(e){for(e=e.child;e!==null;)e.tag===30?qi(e.child,!1):(e.subtreeFlags&33554432)!==0&&T0(e),e=e.sibling}function dd(e,n,a,s,l,u,p){for(var A=!1;n!==null;){if(n.tag===5){var B=n.stateNode;if(u!==null&&Wn<u.length){var te=u[Wn],fe=Xd(B);(te.view||fe.view)&&(A=!0);var ye;if(ye=(e.flags&4)===0)if(fe.clip)ye=!0;else{ye=te.rect;var J=fe.rect;ye=ye.y!==J.y||ye.x!==J.x||ye.height!==J.height||ye.width!==J.width}ye&&(e.flags|=4),fe.abs?fe=!te.abs:(te=te.rect,fe=fe.rect,fe=te.height!==fe.height||te.width!==fe.width),fe&&(e.flags|=32)}else e.flags|=32;(e.flags&4)!==0&&M_(B,Wn===0?a:a+"_"+Wn,l),A&&(e.flags&4)!==0||(Wi===null&&(Wi=[]),Wi.push(B,Wn===0?s:s+"_"+Wn,n.memoizedProps)),Wn++}else(n.tag!==22||n.memoizedState===null)&&(n.tag===30&&p?e.flags|=n.flags&32:dd(e,n.child,a,s,l,u,p)&&(A=!0));n=n.sibling}return A}function A0(e,n){for(e=e.child;e!==null;){if(e.tag===30){var a=e.memoizedProps,s=e.stateNode,l=pa(a,s),u=ma(a.default,a.update),p;p=e.memoizedState,e.memoizedState=null,s=e;var A=e.child;Wn=0,l=dd(s,A,l,l,u,p,!1),(e.flags&4)!==0&&l&&Cr(e,a.onUpdate)}else(e.subtreeFlags&33554432)!==0&&A0(e);e=e.sibling}}var En=!1,Ht=!1,Yi=!1,hd=!1,R0=typeof WeakSet=="function"?WeakSet:Set,bn=null,ji=!1,Vo=!1,Mc=!1,pd=!1;function vy(e,n,a){if(e=e.containerInfo,Fd=Fr,e=Dm(e),$u(e)){if("selectionStart"in e)var s={start:e.selectionStart,end:e.selectionEnd};else e:{s=(s=e.ownerDocument)&&s.defaultView||window;var l=s.getSelection&&s.getSelection();if(l&&l.rangeCount!==0){s=l.anchorNode;var u=l.anchorOffset,p=l.focusNode;l=l.focusOffset;try{s.nodeType,p.nodeType}catch{s=null;break e}var A=0,B=-1,te=-1,fe=0,ye=0,J=e,oe=null;t:for(;;){for(var Oe;J!==s||u!==0&&J.nodeType!==3||(B=A+u),J!==p||l!==0&&J.nodeType!==3||(te=A+l),J.nodeType===3&&(A+=J.nodeValue.length),(Oe=J.firstChild)!==null;)oe=J,J=Oe;for(;;){if(J===e)break t;if(oe===s&&++fe===u&&(B=A),oe===p&&++ye===l&&(te=A),(Oe=J.nextSibling)!==null)break;J=oe,oe=J.parentNode}J=Oe}s=B===-1||te===-1?null:{start:B,end:te}}else s=null}s=s||{start:0,end:0}}else s=null;for(Hd={focusedElem:e,selectionRange:s},Fr=!1,a=(a&335544064)===a,bn=n,n=a?9270:1024;bn!==null;){if(e=bn,a&&(s=e.deletions,s!==null))for(u=0;u<s.length;u++)a&&ud(s[u]);if(e.alternate===null&&(e.flags&2)!==0)a&&y0(e),Ec(a);else{if(e.tag===22){if(s=e.alternate,e.memoizedState!==null){s!==null&&s.memoizedState===null&&a&&ud(s),Ec(a);continue}else if(s!==null&&s.memoizedState!==null){a&&y0(e),Ec(a);continue}}s=e.child,(e.subtreeFlags&n)!==0&&s!==null?(s.return=e,bn=s):(a&&b0(e),Ec(a))}}ii=null}function Ec(e){for(;bn!==null;){var n=bn,a=e,s=n.alternate,l=n.flags;switch(n.tag){case 0:case 11:case 15:break;case 1:if((l&1024)!==0&&s!==null){a=void 0,l=s.memoizedProps,s=s.memoizedState;var u=n.stateNode;try{var p=Bs(n.type,l);a=u.getSnapshotBeforeUpdate(p,s),u.__reactInternalSnapshotBeforeUpdate=a}catch(A){Wt(n,n.return,A)}}break;case 3:if((l&1024)!==0){if(s=n.stateNode.containerInfo,a=s.nodeType,a===9)Yd(s);else if(a===1)switch(s.nodeName){case"HEAD":case"HTML":case"BODY":Yd(s);break;default:s.textContent=""}}break;case 5:case 26:case 27:case 6:case 4:case 17:break;case 30:a&&s!==null&&(a=pa(s.memoizedProps,s.stateNode),l=n.memoizedProps,l=ma(l.default,l.update),l!=="none"&&yr(s,a,l,s.memoizedState=[],!0));break;default:if((l&1024)!==0)throw Error(r(163))}if(s=n.sibling,s!==null){s.return=n.return,bn=s;break}bn=n.return}}function w0(e,n,a){var s=a.flags;switch(a.tag){case 0:case 11:case 15:Zi(e,a),s&4&&Ho(5,a);break;case 1:if(Zi(e,a),s&4)if(e=a.stateNode,n===null)try{e.componentDidMount()}catch(p){Wt(a,a.return,p)}else{var l=Bs(a.type,n.memoizedProps);n=n.memoizedState;try{e.componentDidUpdate(l,n,e.__reactInternalSnapshotBeforeUpdate)}catch(p){Wt(a,a.return,p)}}s&64&&g0(a),s&512&&Xi(a,a.return);break;case 3:if(Zi(e,a),s&64&&(e=a.updateQueue,e!==null)){if(n=null,a.child!==null)switch(a.child.tag){case 27:case 5:n=a.child.stateNode;break;case 1:n=a.child.stateNode}try{sg(e,n)}catch(p){Wt(a,a.return,p)}}break;case 27:n===null&&s&4&&S0(a);case 26:case 5:Zi(e,a),n===null&&s&4&&ad(a),s&512&&Xi(a,a.return);break;case 12:Zi(e,a);break;case 31:Zi(e,a),s&4&&U0(e,a);break;case 13:Zi(e,a),s&4&&L0(e,a),s&64&&(e=a.memoizedState,e!==null&&(e=e.dehydrated,e!==null&&(a=Ny.bind(null,a),fM(e,a))));break;case 22:if(s=a.memoizedState!==null||En,!s){var u=n!==null&&n.memoizedState!==null||Ht;n=En,l=Ht,En=s,(Ht=u)&&!l?(s=2,(a.subtreeFlags&8772)!==0&&(s|=1),Ci(e,a,s)):Zi(e,a),En=n,Ht=l}break;case 30:Zi(e,a),s&512&&Xi(a,a.return);break;case 7:s&512&&Xi(a,a.return);default:Zi(e,a)}}function md(e,n){for(e=e.child;e!==null;)C0(e,n),e=e.sibling}function C0(e,n){switch(e.tag){case 5:case 26:try{var a=e.stateNode;if(n){var s=a.style;typeof s.setProperty=="function"?s.setProperty("display","none","important"):s.display="none"}else{var l=e.stateNode,u=e.memoizedProps.style,p=u!=null&&u.hasOwnProperty("display")?u.display:null;l.style.display=p==null||typeof p=="boolean"?"":(""+p).trim()}}catch(B){Wt(e,e.return,B)}gd(e,n);break;case 6:try{e.stateNode.nodeValue=n?"":e.memoizedProps,vt=!0}catch(B){Wt(e,e.return,B)}break;case 18:try{var A=e.stateNode;n?y_(A,!0):y_(e.stateNode,!1)}catch(B){Wt(e,e.return,B)}break;case 22:case 23:e.memoizedState===null&&md(e,n);break;default:md(e,n)}}function gd(e,n){if(e.subtreeFlags&67108864)for(e=e.child;e!==null;){e:{var a=e,s=n;switch(a.tag){case 4:C0(a,s);break e;case 22:a.memoizedState===null&&gd(a,s);break e;default:gd(a,s)}}e=e.sibling}}function N0(e){var n=e.alternate;n!==null&&(e.alternate=null,N0(n)),e.child=null,e.deletions=null,e.sibling=null,e.tag===5&&(n=e.stateNode,n!==null&&Ke(n)),e.stateNode=null,e.return=null,e.dependencies=null,e.memoizedProps=null,e.memoizedState=null,e.pendingProps=null,e.stateNode=null,e.updateQueue=null}var tn=null,qn=!1;function Ri(e,n,a){for(a=a.child;a!==null;)D0(e,n,a),a=a.sibling}function D0(e,n,a){if(Ge&&typeof Ge.onCommitFiberUnmount=="function")try{Ge.onCommitFiberUnmount(Je,a)}catch{}switch(a.tag){case 26:Ht||Cn(a,n),Ri(e,n,a),a.memoizedState?a.memoizedState.count--:a.stateNode&&!Ht&&(a=a.stateNode,a.parentNode.removeChild(a));break;case 27:Ht||Cn(a,n),Go(a);var s=tn,l=qn;ss(a.type)&&(tn=a.stateNode,qn=!1),Ri(e,n,a),P_(a.stateNode,a.type,a.memoizedProps),tn=s,qn=l;break;case 5:Ht||Cn(a,n),Go(a);case 6:if(a.tag===6&&Go(a),s=tn,l=qn,tn=null,Ri(e,n,a),tn=s,qn=l,tn!==null)if(qn)try{(tn.nodeType===9?tn.body:tn.nodeName==="HTML"?tn.ownerDocument.body:tn).removeChild(a.stateNode),vt=!0}catch(u){Wt(a,n,u)}else try{tn.removeChild(a.stateNode),vt=!0}catch(u){Wt(a,n,u)}break;case 18:tn!==null&&(qn?(e=tn,S_(e.nodeType===9?e.body:e.nodeName==="HTML"?e.ownerDocument.body:e,a.stateNode),Hr(e)):S_(tn,a.stateNode));break;case 4:s=tn,l=qn,tn=a.stateNode.containerInfo,qn=!0,Ri(e,n,a),tn=s,qn=l;break;case 0:case 11:case 14:case 15:Ja(2,a,n),Ht||Ja(4,a,n),Ri(e,n,a);break;case 1:Ht||(Cn(a,n),s=a.stateNode,typeof s.componentWillUnmount=="function"&&_0(a,n,s)),Ri(e,n,a);break;case 21:Ri(e,n,a);break;case 22:Ht=(s=Ht)||a.memoizedState!==null,Ri(e,n,a),Ht=s;break;case 30:Cn(a,n),Ri(e,n,a);break;case 7:Ht||Cn(a,n),Ri(e,n,a);break;default:Ri(e,n,a)}}function U0(e,n){if(n.memoizedState===null&&(e=n.alternate,e!==null&&(e=e.memoizedState,e!==null))){e=e.dehydrated;try{Hr(e)}catch(a){Wt(n,n.return,a)}}}function L0(e,n){if(n.memoizedState===null&&(e=n.alternate,e!==null&&(e=e.memoizedState,e!==null&&(e=e.dehydrated,e!==null))))try{Hr(e)}catch(a){Wt(n,n.return,a)}}function xy(e){switch(e.tag){case 31:case 13:case 19:var n=e.stateNode;return n===null&&(n=e.stateNode=new R0),n;case 22:return e=e.stateNode,n=e._retryCache,n===null&&(n=e._retryCache=new R0),n;default:throw Error(r(435,e.tag))}}function bc(e,n){var a=xy(e);n.forEach(function(s){if(!a.has(s)){a.add(s);var l=Dy.bind(null,e,s);s.then(l,l)}})}function Hn(e,n,a){var s=n.deletions;if(s!==null)for(var l=0;l<s.length;l++){var u=s[l],p=e,A=n,B=A;e:for(;B!==null;){switch(B.tag){case 27:if(ss(B.type)){tn=B.stateNode,qn=!1;break e}break;case 5:tn=B.stateNode,qn=!1;break e;case 3:case 4:tn=B.stateNode.containerInfo,qn=!0;break e}B=B.return}if(tn===null)throw Error(r(160));D0(p,A,u),tn=null,qn=!1,p=u.alternate,p!==null&&(p.return=null),u.return=null}if(n.subtreeFlags&13886)for(n=n.child;n!==null;)O0(n,e,a),n=n.sibling}var wi=null;function O0(e,n,a){var s=e.alternate,l=e.flags;switch(e.tag){case 0:case 11:case 14:case 15:if(l&4&&(s=e.updateQueue,s=s!==null?s.events:null,s!==null))for(var u=0;u<s.length;u++){var p=s[u];p.ref.impl=p.nextImpl}Hn(n,e,a),Gn(e),l&4&&(Ja(3,e,e.return),Ho(3,e),Ja(5,e,e.return));break;case 1:Hn(n,e,a),Gn(e),l&512&&(Ht||s===null||Cn(s,s.return)),l&64&&En&&(e=e.updateQueue,e!==null&&(n=e.callbacks,n!==null&&(a=e.shared.hiddenCallbacks,e.shared.hiddenCallbacks=a===null?n:a.concat(n))));break;case 26:if(u=wi,Hn(n,e,a),Gn(e),l&512&&(Ht||s===null||Cn(s,s.return)),l&4)if(l=s!==null?s.memoizedState:null,a=e.memoizedState,s===null)if(a===null)if(e.stateNode===null)if(En)e.stateNode=__(e.type,e.memoizedProps,n.containerInfo,e);else{e:{n=e.type,a=e.memoizedProps,l=u.ownerDocument||u;t:switch(n){case"title":s=l.getElementsByTagName("title")[0],(!s||s[Le]||s[T]||s.namespaceURI==="http://www.w3.org/2000/svg"||s.hasAttribute("itemprop"))&&(s=l.createElement(n),l.head.insertBefore(s,l.querySelector("head > title"))),Nn(s,n,a),s[T]=e,_t(s),n=s;break e;case"link":if(u=G_("link","href",l).get(n+(a.href||""))){for(p=0;p<u.length;p++)if(s=u[p],s.getAttribute("href")===(a.href==null||a.href===""?null:a.href)&&s.getAttribute("rel")===(a.rel==null?null:a.rel)&&s.getAttribute("title")===(a.title==null?null:a.title)&&s.getAttribute("crossorigin")===(a.crossOrigin==null?null:a.crossOrigin)){u.splice(p,1);break t}}s=l.createElement(n),Nn(s,n,a),l.head.appendChild(s);break;case"meta":if(u=G_("meta","content",l).get(n+(a.content||""))){for(p=0;p<u.length;p++)if(s=u[p],s.getAttribute("content")===(a.content==null?null:""+a.content)&&s.getAttribute("name")===(a.name==null?null:a.name)&&s.getAttribute("property")===(a.property==null?null:a.property)&&s.getAttribute("http-equiv")===(a.httpEquiv==null?null:a.httpEquiv)&&s.getAttribute("charset")===(a.charSet==null?null:a.charSet)){u.splice(p,1);break t}}s=l.createElement(n),Nn(s,n,a),l.head.appendChild(s);break;default:throw Error(r(468,n))}s[T]=e,_t(s),n=s}e.stateNode=n}else En||eh(u,e.type,e.stateNode);else e.stateNode=H_(u,a,e.memoizedProps);else l!==a?(l===null?(n=s.stateNode,n===null||Ht||n.parentNode.removeChild(n)):l.count--,a===null?En||eh(u,e.type,e.stateNode):H_(u,a,e.memoizedProps)):a===null&&e.stateNode!==null&&sd(e,e.memoizedProps,s.memoizedProps);break;case 27:Hn(n,e,a),Gn(e),l&512&&(Ht||s===null||Cn(s,s.return)),s!==null&&l&4&&sd(e,e.memoizedProps,s.memoizedProps);break;case 5:if(u=Yi,Yi=!1,Hn(n,e,a),Yi=u,Gn(e),l&512&&(Ht||s===null||Cn(s,s.return)),e.flags&32){n=e.stateNode;try{ir(n,""),vt=!0}catch(fe){Wt(e,e.return,fe)}}l&4&&e.stateNode!=null&&(n=e.memoizedProps,sd(e,n,s!==null?s.memoizedProps:n)),l&1024&&(hd=!0);break;case 6:if(Hn(n,e,a),Gn(e),l&4){if(e.stateNode===null)throw Error(r(162));n=e.memoizedProps,a=e.stateNode;try{a.nodeValue=n,vt=!0}catch(fe){Wt(e,e.return,fe)}}break;case 3:if(vt=!1,Fc=null,u=wi,wi=Qo(n.containerInfo),Hn(n,e,a),wi=u,Gn(e),l&4&&s!==null&&s.memoizedState.isDehydrated)try{Hr(n.containerInfo)}catch(fe){Wt(e,e.return,fe)}hd&&(hd=!1,P0(e)),vt=!1;break;case 4:l=Yi,Yi=En,s=Bt(),u=wi,wi=Qo(e.stateNode.containerInfo),Hn(n,e,a),Gn(e),wi=u,vt&&Vo&&(Mc=!0),vt=s,Yi=l;break;case 12:Hn(n,e,a),Gn(e);break;case 31:Hn(n,e,a),Gn(e),l&4&&(n=e.updateQueue,n!==null&&(e.updateQueue=null,bc(e,n)));break;case 13:Hn(n,e,a),Gn(e),e.child.flags&8192&&e.memoizedState!==null!=(s!==null&&s.memoizedState!==null)&&(Rc=ke()),l&4&&(n=e.updateQueue,n!==null&&(e.updateQueue=null,bc(e,n)));break;case 22:u=e.memoizedState!==null,p=s!==null&&s.memoizedState!==null;var A=En,B=Ht,te=Yi;En=A||u,Yi=te||u,Ht=B||p,Hn(n,e,a),Ht=B,Yi=te,En=A,Gn(e),l&8192&&(n=e.stateNode,n._visibility=u?n._visibility&-2:n._visibility|1,!u||s===null||p||En||Ht||(n=p||Ht,a=En,s=Ht,En=u||En,Ht=n,$a(e,2),En=a,Ht=s),!u&&Yi||md(e,u)),l&4&&(n=e.updateQueue,n!==null&&(a=n.retryQueue,a!==null&&(n.retryQueue=null,bc(e,a))));break;case 19:Hn(n,e,a),Gn(e),l&4&&(n=e.updateQueue,n!==null&&(e.updateQueue=null,bc(e,n)));break;case 30:l&512&&(Ht||s===null||Cn(s,s.return)),l=Bt(),u=Vo,p=(a&335544064)===a,A=e.memoizedProps,Vo=p&&ma(A.default,A.update)!=="none",Hn(n,e,a),Gn(e),p&&s!==null&&vt&&(e.flags|=4),Vo=u,vt=l;break;case 21:break;case 7:l&512&&(Ht||s===null||Cn(s,s.return)),s&&s.stateNode!==null&&(s.stateNode._fragmentFiber=e);default:Hn(n,e,a),Gn(e)}}function Gn(e){var n=e.flags;if(n&2){try{for(var a,s=e.return;s!==null;){if(x0(s)){a=s;break}s=s.return}s=null;for(var l=e.return;l!==null;){if(id(l)){var u=l.stateNode;s===null?s=[u]:s.push(u)}if(nd(l))break;l=l.return}var p=s;if(a==null)throw Error(r(160));switch(a.tag){case 27:var A=a.stateNode,B=rd(e);vc(e,B,A,p);break;case 5:var te=a.stateNode;a.flags&32&&(ir(te,""),a.flags&=-33);var fe=rd(e);vc(e,fe,te,p);break;case 3:case 4:var ye=a.stateNode.containerInfo,J=rd(e);od(e,J,ye,p);break;default:throw Error(r(161))}}catch(oe){Wt(e,e.return,oe)}e.flags&=-3}n&4096&&(e.flags&=-4097)}function P0(e){if(e.subtreeFlags&1024)for(e=e.child;e!==null;){var n=e;P0(n),n.tag===5&&n.flags&1024&&(n=n.stateNode,Fr=!0,n.reset(),Fr=!1),e=e.sibling}}function Mr(e,n){if(n.subtreeFlags&9270)for(n=n.child;n!==null;)I0(n,e),n=n.sibling;else A0(n)}function I0(e,n){var a=e.alternate;if(a===null)ld(e,!1);else switch(e.tag){case 3:if(pd=ji=!1,M0(),Mr(n,e),!ji&&!Mc){if(e=Wi,e!==null)for(var s=0;s<e.length;s+=3){a=e[s];var l=e[s+1];E_(a,e[s+2]),a=a.ownerDocument.documentElement,a!==null&&a.animate({opacity:[0,0],pointerEvents:["none","none"]},{duration:0,fill:"forwards",pseudoElement:"::view-transition-group("+l+")"})}e=n.containerInfo,e=e.nodeType===9?e.documentElement:e.ownerDocument.documentElement,e!==null&&e.style.viewTransitionName===""&&(e.style.viewTransitionName="none",e.animate({opacity:[0,0],pointerEvents:["none","none"]},{duration:0,fill:"forwards",pseudoElement:"::view-transition-group(root)"}),e.animate({width:[0,0],height:[0,0]},{duration:0,fill:"forwards",pseudoElement:"::view-transition"})),pd=!0}Wi=null;break;case 5:Mr(n,e);break;case 4:s=ji,ji=!1,Mr(n,e),ji&&(Mc=!0),ji=s;break;case 22:e.memoizedState===null&&(a.memoizedState!==null?ld(e,!1):Mr(n,e));break;case 30:s=ji,l=M0(),ji=!1,Mr(n,e),ji&&(e.flags|=4);var u=e.memoizedProps,p=e.stateNode;n=pa(u,p),p=pa(a.memoizedProps,p);var A=ma(u.default,u.update);A==="none"?n=!1:(u=a.memoizedState,a.memoizedState=null,a=e.child,Wn=0,n=dd(e,a,n,p,A,u,!0),Wn!==(u===null?0:u.length)&&(e.flags|=32)),(e.flags&4)!==0&&n?(Cr(e,e.memoizedProps.onUpdate),Wi=l):l!==null&&(l.push.apply(l,Wi),Wi=l),ji=(e.flags&32)!==0?!0:s;break;default:Mr(n,e)}}function Zi(e,n){if(n.subtreeFlags&8772)for(n=n.child;n!==null;)w0(e,n.alternate,n),n=n.sibling}function $a(e,n){for(e=e.child;e!==null;){var a=e,s=n;switch(a.tag){case 0:case 11:case 14:case 15:Ja(4,a,a.return),$a(a,s);break;case 1:Cn(a,a.return);var l=a.stateNode;typeof l.componentWillUnmount=="function"&&_0(a,a.return,l),$a(a,s);break;case 27:(s&2)!==0&&P_(a.stateNode,a.type,a.memoizedProps);case 5:Cn(a,a.return),a.tag!==5&&a.tag!==27||Go(a),$a(a,s);break;case 6:Go(a);break;case 26:Cn(a,a.return),l=a.stateNode,a.memoizedState!==null||l===null||Ht||l.parentNode.removeChild(l),$a(a,s);break;case 22:a.memoizedState===null&&$a(a,s);break;case 30:Cn(a,a.return),$a(a,s);break;case 7:Cn(a,a.return);default:$a(a,s)}e=e.sibling}}function Ci(e,n,a){for(a=(n.subtreeFlags&8772)!==0?a:a&-2,n=n.child;n!==null;){var s=n.alternate,l=e,u=n,p=u.flags,A=(a&1)!==0;switch(u.tag){case 0:case 11:case 15:Ci(l,u,a),Ho(4,u);break;case 1:if(Ci(l,u,a),s=u,l=s.stateNode,typeof l.componentDidMount=="function")try{l.componentDidMount()}catch(fe){Wt(s,s.return,fe)}if(s=u,l=s.updateQueue,l!==null){var B=s.stateNode;try{var te=l.shared.hiddenCallbacks;if(te!==null)for(l.shared.hiddenCallbacks=null,l=0;l<te.length;l++)ag(te[l],B)}catch(fe){Wt(s,s.return,fe)}}A&&p&64&&g0(u),Xi(u,u.return);break;case 27:(a&2)!==0&&S0(u);case 5:u.tag!==5&&u.tag!==27||v0(u),Ci(l,u,a),A&&s===null&&p&4&&ad(u),Xi(u,u.return);break;case 6:v0(u);break;case 26:B=u.stateNode,u.memoizedState!==null||B===null||En||eh(Qo(B.ownerDocument),u.type,B),Ci(l,u,a),A&&s===null&&p&4&&ad(u),Xi(u,u.return);break;case 12:Ci(l,u,a);break;case 31:Ci(l,u,a),A&&p&4&&U0(l,u);break;case 13:Ci(l,u,a),A&&p&4&&L0(l,u);break;case 22:u.memoizedState===null&&Ci(l,u,a),Xi(u,u.return);break;case 30:Ci(l,u,a),Xi(u,u.return);break;case 7:Xi(u,u.return);default:Ci(l,u,a)}n=n.sibling}}function _d(e,n){var a=null;e!==null&&e.memoizedState!==null&&e.memoizedState.cachePool!==null&&(a=e.memoizedState.cachePool.pool),e=null,n.memoizedState!==null&&n.memoizedState.cachePool!==null&&(e=n.memoizedState.cachePool.pool),e!==a&&(e!=null&&e.refCount++,a!=null&&Ro(a))}function vd(e,n){e=null,n.alternate!==null&&(e=n.alternate.memoizedState.cache),n=n.memoizedState.cache,n!==e&&(n.refCount++,e!=null&&Ro(e))}function Si(e,n,a,s){var l=(a&335544064)===a;if(n.subtreeFlags&(l?10262:10256))for(n=n.child;n!==null;)B0(e,n,a,s),n=n.sibling;else l&&T0(n)}function B0(e,n,a,s){var l=(a&335544064)===a;l&&n.alternate===null&&n.return!==null&&n.return.alternate!==null&&yc(n);var u=n.flags;switch(n.tag){case 0:case 11:case 15:Si(e,n,a,s),u&2048&&Ho(9,n);break;case 1:Si(e,n,a,s);break;case 3:Si(e,n,a,s),l&&pd&&(e=e.containerInfo,e=e.nodeType===9?e.body:e.nodeName==="HTML"?e.ownerDocument.body:e,e.style.viewTransitionName==="root"&&(e.style.viewTransitionName=""),e=e.ownerDocument.documentElement,e!==null&&e.style.viewTransitionName==="none"&&(e.style.viewTransitionName="")),u&2048&&(u=null,n.alternate!==null&&(u=n.alternate.memoizedState.cache),n=n.memoizedState.cache,n!==u&&(n.refCount++,u!=null&&Ro(u)));break;case 12:if(u&2048){Si(e,n,a,s),u=n.stateNode;try{var p=n.memoizedProps,A=p.id,B=p.onPostCommit;typeof B=="function"&&B(A,n.alternate===null?"mount":"update",u.passiveEffectDuration,-0)}catch(te){Wt(n,n.return,te)}}else Si(e,n,a,s);break;case 31:Si(e,n,a,s);break;case 13:Si(e,n,a,s);break;case 23:break;case 22:p=n.stateNode,A=n.alternate,n.memoizedState!==null?(l&&A!==null&&A.memoizedState===null&&yc(A),p._visibility&2?Si(e,n,a,s):ko(e,n)):(l&&A!==null&&A.memoizedState!==null&&yc(n),p._visibility&2?Si(e,n,a,s):(p._visibility|=2,Er(e,n,a,s,(n.subtreeFlags&10256)!==0||!1))),u&2048&&_d(A,n);break;case 24:Si(e,n,a,s),u&2048&&vd(n.alternate,n);break;case 30:l&&(u=n.alternate,u!==null&&(qi(u.child,!0),qi(n.child,!0))),Si(e,n,a,s);break;default:Si(e,n,a,s)}}function Er(e,n,a,s,l){for(l=l&&((n.subtreeFlags&10256)!==0||!1),n=n.child;n!==null;){var u=e,p=n,A=a,B=s,te=p.flags;switch(p.tag){case 0:case 11:case 15:Er(u,p,A,B,l),Ho(8,p);break;case 23:break;case 22:var fe=p.stateNode;p.memoizedState!==null?fe._visibility&2?Er(u,p,A,B,l):ko(u,p):(fe._visibility|=2,Er(u,p,A,B,l)),l&&te&2048&&_d(p.alternate,p);break;case 24:Er(u,p,A,B,l),l&&te&2048&&vd(p.alternate,p);break;default:Er(u,p,A,B,l)}n=n.sibling}}function ko(e,n){if(n.subtreeFlags&10256)for(n=n.child;n!==null;){var a=e,s=n,l=s.flags;switch(s.tag){case 22:ko(a,s),l&2048&&_d(s.alternate,s);break;case 24:ko(a,s),l&2048&&vd(s.alternate,s);break;default:ko(a,s)}n=n.sibling}}var zs=8192;function Fs(e,n,a){if(e.subtreeFlags&zs)for(e=e.child;e!==null;)z0(e,n,a),e=e.sibling}function z0(e,n,a){switch(e.tag){case 26:Fs(e,n,a),e.flags&zs&&(e.memoizedState!==null?TM(a,wi,e.memoizedState,e.memoizedProps):(e=e.stateNode,(n&335544128)===n&&W_(a,e)));break;case 5:Fs(e,n,a),e.flags&zs&&(e=e.stateNode,(n&335544128)===n&&W_(a,e));break;case 3:case 4:var s=wi;wi=Qo(e.stateNode.containerInfo),Fs(e,n,a),wi=s;break;case 22:e.memoizedState===null&&(s=e.alternate,s!==null&&s.memoizedState!==null?(s=zs,zs=16777216,Fs(e,n,a),zs=s):Fs(e,n,a));break;case 30:if((e.flags&zs)!==0&&(s=e.memoizedProps.name,s!=null&&s!=="auto")){var l=e.stateNode;l.paired=null,ii===null&&(ii=new Map),ii.set(s,l)}Fs(e,n,a);break;default:Fs(e,n,a)}}function F0(e){var n=e.alternate;if(n!==null&&(e=n.child,e!==null)){n.child=null;do n=e.sibling,e.sibling=null,e=n;while(e!==null)}}function Xo(e){var n=e.deletions;if((e.flags&16)!==0){if(n!==null)for(var a=0;a<n.length;a++){var s=n[a];bn=s,G0(s,e)}F0(e)}if(e.subtreeFlags&10256)for(e=e.child;e!==null;)H0(e),e=e.sibling}function H0(e){switch(e.tag){case 0:case 11:case 15:Xo(e),e.flags&2048&&Ja(9,e,e.return);break;case 3:Xo(e);break;case 12:Xo(e);break;case 22:var n=e.stateNode;e.memoizedState!==null&&n._visibility&2&&(e.return===null||e.return.tag!==13)?(n._visibility&=-3,Tc(e)):Xo(e);break;default:Xo(e)}}function Tc(e){var n=e.deletions;if((e.flags&16)!==0){if(n!==null)for(var a=0;a<n.length;a++){var s=n[a];bn=s,G0(s,e)}F0(e)}for(e=e.child;e!==null;){switch(n=e,n.tag){case 0:case 11:case 15:Ja(8,n,n.return),Tc(n);break;case 22:a=n.stateNode,a._visibility&2&&(a._visibility&=-3,Tc(n));break;default:Tc(n)}e=e.sibling}}function G0(e,n){for(;bn!==null;){var a=bn;switch(a.tag){case 0:case 11:case 15:Ja(8,a,n);break;case 23:case 22:if(a.memoizedState!==null&&a.memoizedState.cachePool!==null){var s=a.memoizedState.cachePool.pool;s!=null&&s.refCount++}break;case 24:Ro(a.memoizedState.cache)}if(s=a.child,s!==null)s.return=a,bn=s;else e:for(a=e;bn!==null;){s=bn;var l=s.sibling,u=s.return;if(N0(s),s===a){bn=null;break e}if(l!==null){l.return=u,bn=l;break e}bn=u}}}var Sy={getCacheForType:function(e){var n=An(hn),a=n.data.get(e);return a===void 0&&(a=e(),n.data.set(e,a)),a},cacheSignal:function(){return An(hn).controller.signal}},yy=typeof WeakMap=="function"?WeakMap:Map,zt=0,Kt=null,Mt=null,Tt=0,Xt=0,ai=null,es=!1,br=!1,xd=!1,ba=0,cn=0,ts=0,Hs=0,Ac=0,si=0,Tr=0,Wo=null,Yn=null,Sd=!1,Rc=0,V0=0,wc=1/0,Cc=null,ns=null,an=0,Ni=null,Gs=null,Ki=0,yd=0,Md=null,k0=null,Ar=null,Rr=null,wr=null,qo=0,Nc=null;function ri(){return(zt&2)!==0&&Tt!==0?Tt&-Tt:xe.T!==null?Ud():wl()}function X0(){if(si===0)if((Tt&536870912)===0||xt){var e=Ss;Ss<<=1,(Ss&3932160)===0&&(Ss=262144),si=e}else si=536870912;return e=Rn.current,e!==null&&(e.flags|=32),si}function Cr(e,n){if(n!=null){var a=e.stateNode,s=a.ref;s===null&&(s=a.ref=b_(pa(e.memoizedProps,a))),Rr===null&&(Rr=[]),Rr.push(n.bind(null,s))}}function jn(e,n,a){(e===Kt&&(Xt===2||Xt===9)||e.cancelPendingCommit!==null)&&(Nr(e,0),is(e,Tt,si,!1)),Fi(e,a),((zt&2)===0||e!==Kt)&&(e===Kt&&((zt&2)===0&&(Hs|=a),cn===4&&is(e,Tt,si,!1)),Qi(e))}function W0(e,n,a){if((zt&6)!==0)throw Error(r(327));var s=!a&&(n&127)===0&&(n&e.expiredLanes)===0||za(e,n),l=s?by(e,n):bd(e,n,!0),u=s;do{if(l===0){br&&!s&&is(e,n,0,!1);break}else{if(a=e.current.alternate,u&&!My(a)){l=bd(e,n,!1),u=!1;continue}if(l===2){if(u=n,e.errorRecoveryDisabledLanes&u)var p=0;else p=e.pendingLanes&-536870913,p=p!==0?p:p&536870912?536870912:0;if(p!==0){n=p;e:{var A=e;l=Wo;var B=A.current.memoizedState.isDehydrated;if(B&&(Nr(A,p).flags|=256),p=bd(A,p,!1),p!==2&&p!==6){if(xd&&!B){A.errorRecoveryDisabledLanes|=u,Hs|=u,l=4;break e}u=Yn,Yn=l,u!==null&&(Yn===null?Yn=u:Yn.push.apply(Yn,u))}l=p}if(u=!1,l!==2)continue}}if(l===1){Nr(e,0),is(e,n,0,!0);break}e:{switch(s=e,u=l,u){case 0:case 1:throw Error(r(345));case 4:if((n&4194048)!==n&&(n&62914560)!==n)break;case 6:is(s,n,si,!es);break e;case 2:Yn=null;break;case 3:case 5:break;default:throw Error(r(329))}if((n&62914560)===n&&(l=Rc+300-ke(),10<l)){if(is(s,n,si,!es),ys(s,0,!0)!==0)break e;Ki=n,s.timeoutHandle=kd(q0.bind(null,s,a,Yn,Cc,Sd,n,si,Hs,Tr,es,u,"Throttled",-0,0),l);break e}q0(s,a,Yn,Cc,Sd,n,si,Hs,Tr,es,u,null,-0,0)}}break}while(!0);Qi(e)}function q0(e,n,a,s,l,u,p,A,B,te,fe,ye,J,oe){e.timeoutHandle=-1;var Oe=n.subtreeFlags,Qe=(u&335544064)===u;if(ye=null,(Qe||Oe&8192||(Oe&16785408)===16785408)&&(ye={stylesheets:null,count:0,imgCount:0,imgBytes:0,suspenseyImages:[],waitingForImages:!0,waitingForViewTransition:!1,unsuspend:Gi},ii=null,z0(n,u,ye),Qe&&(Oe=ye,Qe=e.containerInfo,Qe=(Qe.nodeType===9?Qe:Qe.ownerDocument).__reactViewTransition,Qe!=null&&(Oe.count++,Oe.waitingForViewTransition=!0,Oe=el.bind(Oe),Qe.finished.then(Oe,Oe))),Oe=(u&62914560)===u?Rc-ke():(u&4194048)===u?V0-ke():0,Oe=AM(ye,Oe),Oe!==null)){Ki=u,e.cancelPendingCommit=Oe(e_.bind(null,e,n,u,a,s,l,p,A,B,te,fe,ye,null,J,oe)),is(e,u,p,!te);return}e_(e,n,u,a,s,l,p,A,B,te,fe,ye)}function My(e){for(var n=e;;){var a=n.tag;if((a===0||a===11||a===15)&&n.flags&16384&&(a=n.updateQueue,a!==null&&(a=a.stores,a!==null)))for(var s=0;s<a.length;s++){var l=a[s],u=l.getSnapshot;l=l.value;try{if(!ti(u(),l))return!1}catch{return!1}}if(a=n.child,n.subtreeFlags&16384&&a!==null)a.return=n,n=a;else{if(n===e)break;for(;n.sibling===null;){if(n.return===null||n.return===e)return!0;n=n.return}n.sibling.return=n.return,n=n.sibling}}return!0}function is(e,n,a,s){n=zi(e,n),n&=~Ac,n&=~Hs,e.suspendedLanes|=n,e.pingedLanes&=~n,s&&(e.warmLanes|=n),s=e.expirationTimes;for(var l=n;0<l;){var u=31-ct(l),p=1<<u;s[u]=-1,l&=~p}a!==0&&Ms(e,a,n)}function Dc(){return(zt&6)===0?(Yo(0),!1):!0}function Ed(){if(Mt!==null){if(Xt===0)var e=Mt.return;else e=Mt,va=ws=null,Df(e),gr=null,No=0,e=Mt;for(;e!==null;)m0(e.alternate,e),e=e.return;Mt=null}}function Nr(e,n){var a=e.timeoutHandle;return a!==-1&&(e.timeoutHandle=-1,qy(a)),a=e.cancelPendingCommit,a!==null&&(e.cancelPendingCommit=null,a()),Ki=0,Ed(),Kt=e,Mt=a=ga(e.current,null),Tt=n,Xt=0,ai=null,es=!1,br=za(e,n),xd=!1,Tr=si=Ac=Hs=ts=cn=0,Yn=Wo=null,Sd=!1,ba=zi(e,n),Fl(),a}function Y0(e,n){dt=null,xe.H=uc,n===mr||n===Kl?(n=eg(),Xt=3):n===vf?(n=eg(),Xt=4):Xt=n===qf?8:n!==null&&typeof n=="object"&&typeof n.then=="function"?6:1,ai=n,Mt===null&&(cn=1,fc(e,gi(n,e.current)))}function j0(){var e=Rn.current;return e===null?!0:(Tt&4194048)===Tt?Ln===null:(Tt&62914560)===Tt||(Tt&536870912)!==0?e===Ln:!1}function Z0(){var e=xe.H;return xe.H=uc,e===null?uc:e}function K0(){var e=xe.A;return xe.A=Sy,e}function Uc(){cn=4,es||(Tt&4194048)!==Tt&&Rn.current!==null||(br=!0),(ts&134217727)===0&&(Hs&134217727)===0||Kt===null||is(Kt,Tt,si,!1)}function bd(e,n,a){var s=zt;zt|=2;var l=Z0(),u=K0();(Kt!==e||Tt!==n)&&(Cc=null,Nr(e,n)),n=!1;var p=cn;e:do try{if(Xt!==0&&Mt!==null){var A=Mt,B=ai;switch(Xt){case 8:Ed(),p=6;break e;case 3:case 2:case 9:case 6:Rn.current===null&&(n=!0);var te=Xt;if(Xt=0,ai=null,Dr(e,A,B,te),a&&br){p=0;break e}break;default:te=Xt,Xt=0,ai=null,Dr(e,A,B,te)}}Ey(),p=cn;break}catch(fe){Y0(e,fe)}while(!0);return n&&e.shellSuspendCounter++,va=ws=null,zt=s,xe.H=l,xe.A=u,Mt===null&&(Kt=null,Tt=0,Fl()),p}function Ey(){for(;Mt!==null;)Q0(Mt)}function by(e,n){var a=zt;zt|=2;var s=Z0(),l=K0();Kt!==e||Tt!==n?(Cc=null,wc=ke()+500,Nr(e,n)):br=za(e,n);e:do try{if(Xt!==0&&Mt!==null){n=Mt;var u=ai;t:switch(Xt){case 1:Xt=0,ai=null,Dr(e,n,u,1);break;case 2:case 9:if(Jm(u)){Xt=0,ai=null,J0(n);break}n=function(){Xt!==2&&Xt!==9||Kt!==e||(Xt=7),Qi(e)},u.then(n,n);break e;case 3:Xt=7;break e;case 4:Xt=5;break e;case 7:Jm(u)?(Xt=0,ai=null,J0(n)):(Xt=0,ai=null,Dr(e,n,u,7));break;case 5:var p=null;switch(Mt.tag){case 26:p=Mt.memoizedState;case 5:case 27:var A=Mt;if(p?k_(p):A.stateNode.complete){Xt=0,ai=null;var B=A.sibling;if(B!==null)Mt=B;else{var te=A.return;te!==null?(Mt=te,Lc(te)):Mt=null}break t}}Xt=0,ai=null,Dr(e,n,u,5);break;case 6:Xt=0,ai=null,Dr(e,n,u,6);break;case 8:Ed(),cn=6;break e;default:throw Error(r(462))}}Ty();break}catch(fe){Y0(e,fe)}while(!0);return va=ws=null,xe.H=s,xe.A=l,zt=a,Mt!==null?0:(Kt=null,Tt=0,Fl(),cn)}function Ty(){for(;Mt!==null&&!Be();)Q0(Mt)}function Q0(e){var n=h0(e.alternate,e,ba);e.memoizedProps=e.pendingProps,n===null?Lc(e):Mt=n}function J0(e){var n=e,a=n.alternate;switch(n.tag){case 15:case 0:n=r0(a,n,n.pendingProps,n.type,void 0,Tt);break;case 11:n=r0(a,n,n.pendingProps,n.type.render,n.ref,Tt);break;case 5:Df(n);var s=n;s===Mn&&(xt?(Wl(s),s.tag===5&&s.stateNode!=null&&(Jt=s.stateNode)):(Wl(s),xt=!0));default:m0(a,n),n=Mt=Gm(n,ba),n=h0(a,n,ba)}e.memoizedProps=e.pendingProps,n===null?Lc(e):Mt=n}function Dr(e,n,a,s){va=ws=null,Df(n),gr=null,No=0;var l=n.return;try{if(dy(e,l,n,a,Tt)){cn=1,fc(e,gi(a,e.current)),Mt=null;return}}catch(u){if(l!==null)throw Mt=l,u;cn=1,fc(e,gi(a,e.current)),Mt=null;return}n.flags&32768?(xt||s===1?e=!0:br||(Tt&536870912)!==0?e=!1:(es=e=!0,(s===2||s===9||s===3||s===6)&&(s=Rn.current,s!==null&&s.tag===13&&(s.flags|=16384))),$0(n,e)):Lc(n)}function Lc(e){var n=e;do{if((n.flags&32768)!==0){$0(n,es);return}e=n.return;var a=gy(n.alternate,n,ba);if(a!==null){Mt=a;return}if(n=n.sibling,n!==null){Mt=n;return}Mt=n=e}while(n!==null);cn===0&&(cn=5)}function $0(e,n){do{var a=_y(e.alternate,e);if(a!==null){a.flags&=32767,Mt=a;return}if(a=e.return,a!==null&&(a.flags|=32768,a.subtreeFlags=0,a.deletions=null),!n&&(e=e.sibling,e!==null)){Mt=e;return}Mt=e=a}while(e!==null);cn=6,Mt=null}function e_(e,n,a,s,l,u,p,A,B,te,fe,ye){e.cancelPendingCommit=null;do Oc();while(an!==0);if((zt&6)!==0)throw Error(r(327));if(n!==null){if(n===e.current)throw Error(r(177));e===Kt&&(Mt=Kt=null,Tt=0),Gs=n,Ni=e,Ki=a,Md=l,k0=s,Ay(e,n,a,p,A,B,ye)}}function Ay(e,n,a,s,l,u,p){var A=n.lanes|n.childLanes;if(yd=A,A|=sf,Rl(e,a,A,s,l,u),Rr=null,(a&335544064)===a?(wr=ey(e),s=10262):(wr=null,s=10256),(n.subtreeFlags&s)!==0||(n.flags&s)!==0?(e.callbackNode=null,e.callbackPriority=0,Uy(Ce,function(){return wd(),null})):(e.callbackNode=null,e.callbackPriority=0),xc=!1,s=(n.flags&13878)!==0,(n.subtreeFlags&13878)!==0||s){s=xe.T,xe.T=null,l=we.p,we.p=2,u=zt,zt|=4;try{vy(e,n,a)}finally{zt=u,we.p=l,xe.T=s}}an=1,xc?Ar=Jy(p,e.containerInfo,wr,Td,Ad,wy,Rd,wd,Ry):(Td(),Ad(),Rd())}function Ry(e){if(an!==0){var n=Ni.onRecoverableError;n(e,{componentStack:null})}}function wy(){an===3&&(an=0,I0(Gs,Ni),an=4)}function Td(){if(an===1){an=0;var e=Ni,n=Gs,a=Ki,s=(n.flags&13878)!==0;if((n.subtreeFlags&13878)!==0||s){s=xe.T,xe.T=null;var l=we.p;we.p=2;var u=zt;zt|=4;try{Vo=Mc=!1,O0(n,e,a),a=Hd;var p=Dm(e.containerInfo),A=a.focusedElem,B=a.selectionRange;if(p!==A&&A&&A.ownerDocument&&Nm(A.ownerDocument.documentElement,A)){if(B!==null&&$u(A)){var te=B.start,fe=B.end;if(fe===void 0&&(fe=te),"selectionStart"in A)A.selectionStart=te,A.selectionEnd=Math.min(fe,A.value.length);else{var ye=A.ownerDocument||document,J=ye&&ye.defaultView||window;if(J.getSelection){var oe=J.getSelection(),Oe=A.textContent.length,Qe=Math.min(B.start,Oe),ht=B.end===void 0?Qe:Math.min(B.end,Oe);!oe.extend&&Qe>ht&&(p=ht,ht=Qe,Qe=p);var ee=Cm(A,Qe),V=Cm(A,ht);if(ee&&V&&(oe.rangeCount!==1||oe.anchorNode!==ee.node||oe.anchorOffset!==ee.offset||oe.focusNode!==V.node||oe.focusOffset!==V.offset)){var ae=ye.createRange();ae.setStart(ee.node,ee.offset),oe.removeAllRanges(),Qe>ht?(oe.addRange(ae),oe.extend(V.node,V.offset)):(ae.setEnd(V.node,V.offset),oe.addRange(ae))}}}}for(ye=[],oe=A;oe=oe.parentNode;)oe.nodeType===1&&ye.push({element:oe,left:oe.scrollLeft,top:oe.scrollTop});for(typeof A.focus=="function"&&A.focus(),A=0;A<ye.length;A++){var Se=ye[A];Se.element.scrollLeft=Se.left,Se.element.scrollTop=Se.top}}Fr=!!Fd,Hd=Fd=null}finally{zt=u,we.p=l,xe.T=s}}e.current=n,an=2}}function Ad(){if(an===2){an=0;var e=Ni,n=Gs,a=(n.flags&8772)!==0;if((n.subtreeFlags&8772)!==0||a){a=xe.T,xe.T=null;var s=we.p;we.p=2;var l=zt;zt|=4;try{w0(e,n.alternate,n)}finally{zt=l,we.p=s,xe.T=a}}an=3}}function Rd(){if(an===4||an===3){an=0;var e=Ar;Ar=null,Pe();var n=Ni,a=Gs,s=Ki,l=k0,u=(s&335544064)===s?10262:10256;if((a.subtreeFlags&u)!==0||(a.flags&u)!==0?an=5:(an=0,Gs=Ni=null,t_(n,n.pendingLanes)),u=n.pendingLanes,u===0&&(ns=null),mo(s),a=a.stateNode,Ge&&typeof Ge.onCommitFiberRoot=="function")try{Ge.onCommitFiberRoot(Je,a,void 0,(a.current.flags&128)===128)}catch{}if(l!==null){a=xe.T,u=we.p,we.p=2,xe.T=null;try{for(var p=n.onRecoverableError,A=0;A<l.length;A++){var B=l[A];p(B.value,{componentStack:B.stack})}}finally{xe.T=a,we.p=u}}if(l=Rr,p=wr,wr=null,l!==null&&(Rr=null,p===null&&(p=[]),e!==null))for(B=0;B<l.length;B++)a=(0,l[B])(p),a!==void 0&&e.finished.finally(a);(Ki&3)!==0&&Oc(),Qi(n),u=n.pendingLanes,(s&261930)!==0&&(u&42)!==0?n===Nc?qo++:(qo=0,Nc=n):(qo=0,Nc=null),Yo(0)}}function t_(e,n){(e.pooledCacheLanes&=n)===0&&(n=e.pooledCache,n!=null&&(e.pooledCache=null,Ro(n)))}function Oc(){return Ar!==null&&(Ar.skipTransition(),Ar=null),Td(),Ad(),Rd(),wd()}function wd(){if(an!==5)return!1;var e=Ni,n=yd;yd=0;var a=mo(Ki),s=xe.T,l=we.p;try{we.p=32>a?32:a,xe.T=null,a=Md,Md=null;var u=Ni,p=Ki;if(an=0,Gs=Ni=null,Ki=0,(zt&6)!==0)throw Error(r(331));var A=zt;if(zt|=4,H0(u.current),B0(u,u.current,p,a),zt=A,Yo(0,!1),Ge&&typeof Ge.onPostCommitFiberRoot=="function")try{Ge.onPostCommitFiberRoot(Je,u)}catch{}return!0}finally{we.p=l,xe.T=s,t_(e,n)}}function n_(e,n,a){n=gi(a,n),n=Wf(e.stateNode,n,2),e=ja(e,n,2),e!==null&&(Fi(e,2),Qi(e))}function Wt(e,n,a){if(e.tag===3)n_(e,e,a);else for(;n!==null;){if(n.tag===3){n_(n,e,a);break}else if(n.tag===1){var s=n.stateNode;if(typeof n.type.getDerivedStateFromError=="function"||typeof s.componentDidCatch=="function"&&(ns===null||!ns.has(s))){e=gi(a,e),a=Jg(2),s=ja(n,a,2),s!==null&&($g(a,s,n,e),Fi(s,2),Qi(s));break}}n=n.return}}function Cd(e,n,a){var s=e.pingCache;if(s===null){s=e.pingCache=new yy;var l=new Set;s.set(n,l)}else l=s.get(n),l===void 0&&(l=new Set,s.set(n,l));l.has(a)||(xd=!0,l.add(a),e=Cy.bind(null,e,n,a),n.then(e,e))}function Cy(e,n,a){var s=e.pingCache;s!==null&&s.delete(n),e.pingedLanes|=e.suspendedLanes&a,e.warmLanes&=~a,Kt===e&&(Tt&a)===a&&((cn===4||cn===3&&(Tt&62914560)===Tt&&300>ke()-Rc)&&(zt&2)===0?Nr(e,0):Ac|=a,Tr===Tt&&(Tr=0)),Qi(e)}function i_(e,n){n===0&&(n=uo()),e=Ts(e,n),e!==null&&(Fi(e,n),Qi(e))}function Ny(e){var n=e.memoizedState,a=0;n!==null&&(a=n.retryLane),i_(e,a)}function Dy(e,n){var a=0;switch(e.tag){case 31:case 13:var s=e.stateNode,l=e.memoizedState;l!==null&&(a=l.retryLane);break;case 19:s=e.stateNode;break;case 22:s=e.stateNode._retryCache;break;default:throw Error(r(314))}s!==null&&s.delete(n),i_(e,a)}function Uy(e,n){return Ne(e,n)}var Ur=null,Lr=null,Nd=!1,Pc=!1,Dd=!1,as=0;function Qi(e){e!==Lr&&e.next===null&&(Lr===null?Ur=Lr=e:Lr=Lr.next=e),Pc=!0,Nd||(Nd=!0,Oy())}function Yo(e,n){if(!Dd&&Pc){Dd=!0;do for(var a=!1,s=Ur;s!==null;){if(e!==0){var l=s.pendingLanes;if(l===0)var u=0;else{var p=s.suspendedLanes,A=s.pingedLanes;u=(1<<31-ct(42|e)+1)-1,u&=l&~(p&~A),u=u&201326741?u&201326741|1:u?u|2:0}u!==0&&(a=!0,o_(s,u))}else u=Tt,u=ys(s,s===Kt?u:0,s.cancelPendingCommit!==null||s.timeoutHandle!==-1),(u&3)===0||za(s,u)||(a=!0,o_(s,u));s=s.next}while(a);Dd=!1}}function Ly(){a_()}function a_(){Pc=Nd=!1;var e=0;as!==0&&Wy()&&(e=as);for(var n=ke(),a=null,s=Ur;s!==null;){var l=s.next,u=s_(s,n);u===0?(s.next=null,a===null?Ur=l:a.next=l,l===null&&(Lr=a)):(a=s,(e!==0||(u&3)!==0)&&(Pc=!0)),s=l}an!==0&&an!==5||Yo(e),as!==0&&(as=0)}function s_(e,n){for(var a=e.suspendedLanes,s=e.pingedLanes,l=e.expirationTimes,u=e.pendingLanes&-62914561;0<u;){var p=31-ct(u),A=1<<p,B=l[p];B===-1?((A&a)===0||(A&s)!==0)&&(l[p]=co(A,n)):B<=n&&(e.expiredLanes|=A),u&=~A}if(n=Kt,a=Tt,a=ys(e,e===n?a:0,e.cancelPendingCommit!==null||e.timeoutHandle!==-1),s=e.callbackNode,a===0||e===n&&(Xt===2||Xt===9)||e.cancelPendingCommit!==null)return s!==null&&s!==null&&$e(s),e.callbackNode=null,e.callbackPriority=0;if((a&3)===0||za(e,a)){if(n=a&-a,n===e.callbackPriority)return n;switch(s!==null&&$e(s),mo(a)){case 2:case 8:a=Z;break;case 32:a=Ce;break;case 268435456:a=Ue;break;default:a=Ce}return s=r_.bind(null,e),a=Ne(a,s),e.callbackPriority=n,e.callbackNode=a,n}return s!==null&&s!==null&&$e(s),e.callbackPriority=2,e.callbackNode=null,2}function r_(e,n){if(an!==0&&an!==5)return e.callbackNode=null,e.callbackPriority=0,null;var a=e.callbackNode;if(Oc()&&e.callbackNode!==a)return null;var s=Tt;return s=ys(e,e===Kt?s:0,e.cancelPendingCommit!==null||e.timeoutHandle!==-1),s===0?null:(W0(e,s,n),s_(e,ke()),e.callbackNode!=null&&e.callbackNode===a?r_.bind(null,e):null)}function o_(e,n){if(Oc())return null;W0(e,n,!0)}function Oy(){Yy(function(){(zt&6)!==0?Ne(lt,Ly):a_()})}function Ud(){if(as===0){var e=Ds;e===0&&(e=er,er<<=1,(er&261888)===0&&(er=256)),as=e}return as}function l_(e){return e==null||typeof e=="symbol"||typeof e=="boolean"?null:typeof e=="function"?e:Dl(e)}function Py(e,n,a,s,l){if(n==="submit"&&a&&a.stateNode===l){var u=l_((l[G]||null).action),p=s.submitter;p&&(n=(n=p[G]||null)?l_(n.formAction):p.getAttribute("formAction"),n!==null&&(u=n,p=null));var A=new Pl("action","action",null,s,l);e.push({event:A,listeners:[{instance:null,listener:function(){if(s.defaultPrevented){if(as!==0){var B=new FormData(l,p);Hf(a,{pending:!0,data:B,method:l.method,action:u},null,B)}}else typeof u=="function"&&(A.preventDefault(),B=new FormData(l,p),Hf(a,{pending:!0,data:B,method:l.method,action:u},u,B))},currentTarget:l}]})}}for(var Ld=0;Ld<af.length;Ld++){var Od=af[Ld],Iy=Od.toLowerCase(),By=Od[0].toUpperCase()+Od.slice(1);Ai(Iy,"on"+By)}Ai(Om,"onAnimationEnd"),Ai(Pm,"onAnimationIteration"),Ai(Im,"onAnimationStart"),Ai("dblclick","onDoubleClick"),Ai("focusin","onFocus"),Ai("focusout","onBlur"),Ai(qS,"onTransitionRun"),Ai(YS,"onTransitionStart"),Ai(jS,"onTransitionCancel"),Ai(Bm,"onTransitionEnd"),rn("onMouseEnter",["mouseout","mouseover"]),rn("onMouseLeave",["mouseout","mouseover"]),rn("onPointerEnter",["pointerout","pointerover"]),rn("onPointerLeave",["pointerout","pointerover"]),Fe("onChange","change click focusin focusout input keydown keyup selectionchange".split(" ")),Fe("onSelect","focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange".split(" ")),Fe("onBeforeInput",["compositionend","keypress","textInput","paste"]),Fe("onCompositionEnd","compositionend focusout keydown keypress keyup mousedown".split(" ")),Fe("onCompositionStart","compositionstart focusout keydown keypress keyup mousedown".split(" ")),Fe("onCompositionUpdate","compositionupdate focusout keydown keypress keyup mousedown".split(" "));var jo="abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting".split(" "),zy=new Set("beforetoggle cancel close invalid load scroll scrollend toggle".split(" ").concat(jo));function c_(e,n){n=(n&4)!==0;for(var a=0;a<e.length;a++){var s=e[a],l=s.event;s=s.listeners;e:{var u=void 0;if(n)for(var p=s.length-1;0<=p;p--){var A=s[p],B=A.instance,te=A.currentTarget;if(A=A.listener,B!==u&&l.isPropagationStopped())break e;u=A,l.currentTarget=te;try{u(l)}catch(fe){zl(fe)}l.currentTarget=null,u=B}else for(p=0;p<s.length;p++){if(A=s[p],B=A.instance,te=A.currentTarget,A=A.listener,B!==u&&l.isPropagationStopped())break e;u=A,l.currentTarget=te;try{u(l)}catch(fe){zl(fe)}l.currentTarget=null,u=B}}}}function Et(e,n){var a=n[se];a===void 0&&(a=n[se]=new Set);var s=e+"__bubble";a.has(s)||(u_(n,e,2,!1),a.add(s))}function Pd(e,n,a){var s=0;n&&(s|=4),u_(a,e,s,n)}var Ic="_reactListening"+Math.random().toString(36).slice(2);function Id(e){if(!e[Ic]){e[Ic]=!0,Ft.forEach(function(a){a!=="selectionchange"&&(zy.has(a)||Pd(a,!1,e),Pd(a,!0,e))});var n=e.nodeType===9?e:e.ownerDocument;n===null||n[Ic]||(n[Ic]=!0,Pd("selectionchange",!1,n))}}function u_(e,n,a,s){switch($_(n)){case 2:var l=NM;break;case 8:l=DM;break;default:l=nh}a=l.bind(null,n,a,e),l=void 0,!ku||n!=="touchstart"&&n!=="touchmove"&&n!=="wheel"||(l=!0),s?l!==void 0?e.addEventListener(n,a,{capture:!0,passive:l}):e.addEventListener(n,a,!0):l!==void 0?e.addEventListener(n,a,{passive:l}):e.addEventListener(n,a,!1)}function Bd(e,n,a,s,l){var u=s;if((n&1)===0&&(n&2)===0&&s!==null)e:for(;;){if(s===null)return;var p=s.tag;if(p===3||p===4){var A=s.stateNode.containerInfo;if(A===l)break;if(p===4)for(p=s.return;p!==null;){var B=p.tag;if((B===3||B===4)&&p.stateNode.containerInfo===l)return;p=p.return}for(;A!==null;){if(p=st(A),p===null)return;if(B=p.tag,B===5||B===6||B===26||B===27){s=u=p;continue e}A=A.parentNode}}s=s.return}um(function(){var te=u,fe=Gu(a),ye=[];e:{var J=zm.get(e);if(J!==void 0){var oe=Pl,Oe=e;switch(e){case"keypress":if(Ll(a)===0)break e;case"keydown":case"keyup":oe=MS;break;case"focusin":Oe="focus",oe=Yu;break;case"focusout":Oe="blur",oe=Yu;break;case"beforeblur":case"afterblur":oe=Yu;break;case"click":if(a.button===2)break e;case"auxclick":case"dblclick":case"mousedown":case"mousemove":case"mouseup":case"mouseout":case"mouseover":case"contextmenu":oe=hm;break;case"drag":case"dragend":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"dragstart":case"drop":oe=uS;break;case"touchcancel":case"touchend":case"touchmove":case"touchstart":oe=RS;break;case Om:case Pm:case Im:oe=hS;break;case Bm:oe=CS;break;case"scroll":case"scrollend":oe=lS;break;case"wheel":oe=DS;break;case"copy":case"cut":case"paste":oe=mS;break;case"gotpointercapture":case"lostpointercapture":case"pointercancel":case"pointerdown":case"pointermove":case"pointerout":case"pointerover":case"pointerup":oe=mm;break;case"submit":oe=TS;break;case"toggle":case"beforetoggle":oe=LS}var Qe=(n&4)!==0,ht=!Qe&&(e==="scroll"||e==="scrollend"),ee=Qe?J!==null?J+"Capture":null:J;Qe=[];for(var V=te,ae;V!==null;){var Se=V;if(ae=Se.stateNode,Se=Se.tag,Se!==5&&Se!==26&&Se!==27||ae===null||ee===null||(Se=go(V,ee),Se!=null&&Qe.push(Zo(V,Se,ae))),ht)break;V=V.return}0<Qe.length&&(J=new oe(J,Oe,null,a,fe),ye.push({event:J,listeners:Qe}))}}if((n&7)===0){e:{if(oe=e==="mouseover"||e==="pointerover",J=e==="mouseout"||e==="pointerout",oe&&a!==Hu&&(Oe=a.relatedTarget||a.fromElement)&&(st(Oe)||Oe[de]))break e;(J||oe)&&(Oe=fe.window===fe?fe:(oe=fe.ownerDocument)?oe.defaultView||oe.parentWindow:window,J?(oe=a.relatedTarget||a.toElement,J=te,oe=oe?st(oe):null,oe!==null&&(ht=f(oe),Qe=oe.tag,oe!==ht||Qe!==5&&Qe!==27&&Qe!==6)&&(oe=null)):(J=null,oe=te),J!==oe&&(Qe=hm,Se="onMouseLeave",ee="onMouseEnter",V="mouse",(e==="pointerout"||e==="pointerover")&&(Qe=mm,Se="onPointerLeave",ee="onPointerEnter",V="pointer"),ht=J==null?Oe:We(J),ae=oe==null?Oe:We(oe),Oe=new Qe(Se,V+"leave",J,a,fe),Oe.target=ht,Oe.relatedTarget=ae,Se=null,st(fe)===te&&(Qe=new Qe(ee,V+"enter",oe,a,fe),Qe.target=ae,Qe.relatedTarget=ht,Se=Qe),ht=Se,Qe=J&&oe?U(J,oe,Fy):null,J!==null&&f_(ye,Oe,J,Qe,!1),oe!==null&&ht!==null&&f_(ye,ht,oe,Qe,!0)))}e:{if(J=te?We(te):window,oe=J.nodeName&&J.nodeName.toLowerCase(),oe==="select"||oe==="input"&&J.type==="file")var qe=Em;else if(ym(J))if(bm)qe=kS;else{qe=GS;var At=HS}else oe=J.nodeName,!oe||oe.toLowerCase()!=="input"||J.type!=="checkbox"&&J.type!=="radio"?te&&Fu(te.elementType)&&(qe=Em):qe=VS;if(qe&&(qe=qe(e,te))){Mm(ye,qe,a,fe);break e}At&&At(e,J,te)}switch(At=te?We(te):window,e){case"focusin":(ym(At)||At.contentEditable==="true")&&(or=At,ef=te,bo=null);break;case"focusout":bo=ef=or=null;break;case"mousedown":tf=!0;break;case"contextmenu":case"mouseup":case"dragend":tf=!1,Um(ye,a,fe);break;case"selectionchange":if(WS)break;case"keydown":case"keyup":Um(ye,a,fe)}var et;if(Zu)e:{switch(e){case"compositionstart":var at="onCompositionStart";break e;case"compositionend":at="onCompositionEnd";break e;case"compositionupdate":at="onCompositionUpdate";break e}at=void 0}else rr?xm(e,a)&&(at="onCompositionEnd"):e==="keydown"&&a.keyCode===229&&(at="onCompositionStart");at&&(gm&&a.locale!=="ko"&&(rr||at!=="onCompositionStart"?at==="onCompositionEnd"&&rr&&(et=fm()):(Fa=fe,Xu="value"in Fa?Fa.value:Fa.textContent,rr=!0)),At=Bc(te,at),0<At.length&&(at=new pm(at,e,null,a,fe),ye.push({event:at,listeners:At}),et?at.data=et:(et=Sm(a),et!==null&&(at.data=et)))),(et=PS?IS(e,a):BS(e,a))&&(at=Bc(te,"onBeforeInput"),0<at.length&&(At=new pm("onBeforeInput","beforeinput",null,a,fe),ye.push({event:At,listeners:at}),At.data=et)),Py(ye,e,te,a,fe)}c_(ye,n)})}function Zo(e,n,a){return{instance:e,listener:n,currentTarget:a}}function Bc(e,n){for(var a=n+"Capture",s=[];e!==null;){var l=e,u=l.stateNode;if(l=l.tag,l!==5&&l!==26&&l!==27||u===null||(l=go(e,a),l!=null&&s.unshift(Zo(e,l,u)),l=go(e,n),l!=null&&s.push(Zo(e,l,u))),e.tag===3)return s;e=e.return}return[]}function Fy(e){if(e===null)return null;do e=e.return;while(e&&e.tag!==5&&e.tag!==27);return e||null}function f_(e,n,a,s,l){for(var u=n._reactName,p=[];a!==null&&a!==s;){var A=a,B=A.alternate,te=A.stateNode;if(A=A.tag,B!==null&&B===s)break;A!==5&&A!==26&&A!==27||te===null||(B=te,l?(te=go(a,u),te!=null&&p.unshift(Zo(a,te,B))):l||(te=go(a,u),te!=null&&p.push(Zo(a,te,B)))),a=a.return}p.length!==0&&e.push({event:n,listeners:p})}var Hy=/\r\n?/g,Gy=/\u0000|\uFFFD/g;function d_(e){return(typeof e=="string"?e:""+e).replace(Hy,`
`).replace(Gy,"")}function h_(e,n){return n=d_(n),d_(e)===n}function qt(e,n,a,s,l,u){switch(a){case"children":if(typeof s=="string")n==="body"||n==="textarea"&&s===""||ir(e,s);else if(typeof s=="number"||typeof s=="bigint")n!=="body"&&ir(e,""+s);else return;break;case"className":ei(e,"class",s);break;case"tabIndex":ei(e,"tabindex",s);break;case"dir":case"role":case"viewBox":case"width":case"height":ei(e,a,s);break;case"style":lm(e,s,u);return;case"data":if(n!=="object"){ei(e,"data",s);break}case"src":case"href":if(s===""&&(n!=="a"||a!=="href")){e.removeAttribute(a);break}if(s==null||typeof s=="function"||typeof s=="symbol"||typeof s=="boolean"){e.removeAttribute(a);break}s=Dl(s),e.setAttribute(a,s);break;case"action":case"formAction":if(typeof s=="function"){e.setAttribute(a,"javascript:throw new Error('A React form was unexpectedly submitted. If you called form.submit() manually, consider using form.requestSubmit() instead. If you\\'re trying to use event.stopPropagation() in a submit event handler, consider also calling event.preventDefault().')");break}else typeof u=="function"&&(a==="formAction"?(n!=="input"&&qt(e,n,"name",l.name,l,null),qt(e,n,"formEncType",l.formEncType,l,null),qt(e,n,"formMethod",l.formMethod,l,null),qt(e,n,"formTarget",l.formTarget,l,null)):(qt(e,n,"encType",l.encType,l,null),qt(e,n,"method",l.method,l,null),qt(e,n,"target",l.target,l,null)));if(s==null||typeof s=="symbol"||typeof s=="boolean"){e.removeAttribute(a);break}s=Dl(s),e.setAttribute(a,s);break;case"onClick":s!=null&&(e.onclick=Gi);return;case"onScroll":s!=null&&Et("scroll",e);return;case"onScrollEnd":s!=null&&Et("scrollend",e);return;case"dangerouslySetInnerHTML":if(s!=null){if(typeof s!="object"||!("__html"in s))throw Error(r(61));if(a=s.__html,a!=null){if(l.children!=null)throw Error(r(60));(u!=null?u.__html:void 0)!==a&&(e.innerHTML=a)}}break;case"multiple":e.multiple=s&&typeof s!="function"&&typeof s!="symbol";break;case"muted":e.muted=s&&typeof s!="function"&&typeof s!="symbol";break;case"suppressContentEditableWarning":case"suppressHydrationWarning":case"defaultValue":case"defaultChecked":case"innerHTML":case"ref":break;case"autoFocus":break;case"xlinkHref":if(s==null||typeof s=="function"||typeof s=="boolean"||typeof s=="symbol"){e.removeAttribute("xlink:href");break}a=Dl(s),e.setAttributeNS("http://www.w3.org/1999/xlink","xlink:href",a);break;case"contentEditable":case"spellCheck":case"draggable":case"value":case"autoReverse":case"externalResourcesRequired":case"focusable":case"preserveAlpha":s!=null&&typeof s!="function"&&typeof s!="symbol"?e.setAttribute(a,s):e.removeAttribute(a);break;case"inert":case"allowFullScreen":case"async":case"autoPlay":case"controls":case"credentialless":case"default":case"defer":case"disabled":case"disablePictureInPicture":case"disableRemotePlayback":case"formNoValidate":case"hidden":case"loop":case"noModule":case"noValidate":case"open":case"playsInline":case"readOnly":case"required":case"reversed":case"scoped":case"seamless":case"itemScope":s&&typeof s!="function"&&typeof s!="symbol"?e.setAttribute(a,""):e.removeAttribute(a);break;case"capture":case"download":s===!0?e.setAttribute(a,""):s!==!1&&s!=null&&typeof s!="function"&&typeof s!="symbol"?e.setAttribute(a,s):e.removeAttribute(a);break;case"cols":case"rows":case"size":case"span":s!=null&&typeof s!="function"&&typeof s!="symbol"&&!isNaN(s)&&1<=s?e.setAttribute(a,s):e.removeAttribute(a);break;case"rowSpan":case"start":s==null||typeof s=="function"||typeof s=="symbol"||isNaN(s)?e.removeAttribute(a):e.setAttribute(a,s);break;case"popover":Et("beforetoggle",e),Et("toggle",e),Qt(e,"popover",s);break;case"xlinkActuate":bt(e,"http://www.w3.org/1999/xlink","xlink:actuate",s);break;case"xlinkArcrole":bt(e,"http://www.w3.org/1999/xlink","xlink:arcrole",s);break;case"xlinkRole":bt(e,"http://www.w3.org/1999/xlink","xlink:role",s);break;case"xlinkShow":bt(e,"http://www.w3.org/1999/xlink","xlink:show",s);break;case"xlinkTitle":bt(e,"http://www.w3.org/1999/xlink","xlink:title",s);break;case"xlinkType":bt(e,"http://www.w3.org/1999/xlink","xlink:type",s);break;case"xmlBase":bt(e,"http://www.w3.org/XML/1998/namespace","xml:base",s);break;case"xmlLang":bt(e,"http://www.w3.org/XML/1998/namespace","xml:lang",s);break;case"xmlSpace":bt(e,"http://www.w3.org/XML/1998/namespace","xml:space",s);break;case"is":Qt(e,"is",s);break;case"innerText":case"textContent":return;default:if(!(2<a.length)||a[0]!=="o"&&a[0]!=="O"||a[1]!=="n"&&a[1]!=="N")a=rS.get(a)||a,Qt(e,a,s);else return}vt=!0}function zd(e,n,a,s,l,u){switch(a){case"style":lm(e,s,u);return;case"dangerouslySetInnerHTML":if(s!=null){if(typeof s!="object"||!("__html"in s))throw Error(r(61));if(a=s.__html,a!=null){if(l.children!=null)throw Error(r(60));(u!=null?u.__html:void 0)!==a&&(e.innerHTML=a)}}break;case"children":if(typeof s=="string")ir(e,s);else if(typeof s=="number"||typeof s=="bigint")ir(e,""+s);else return;break;case"onScroll":s!=null&&Et("scroll",e);return;case"onScrollEnd":s!=null&&Et("scrollend",e);return;case"onClick":s!=null&&(e.onclick=Gi);return;case"suppressContentEditableWarning":case"suppressHydrationWarning":case"innerHTML":case"ref":return;case"innerText":case"textContent":return;default:if(!vn.hasOwnProperty(a))e:{if(a[0]==="o"&&a[1]==="n"&&(l=a.endsWith("Capture"),u=a.slice(2,l?a.length-7:void 0),n=e[G]||null,n=n!=null?n[a]:null,typeof n=="function"&&e.removeEventListener(u,n,l),typeof s=="function")){typeof n!="function"&&n!==null&&(a in e?e[a]=null:e.hasAttribute(a)&&e.removeAttribute(a)),e.addEventListener(u,s,l);break e}vt=!0,a in e?e[a]=s:s===!0?e.setAttribute(a,""):Qt(e,a,s)}return}vt=!0}function Nn(e,n,a){switch(n){case"div":case"span":case"svg":case"path":case"a":case"g":case"p":case"li":break;case"img":Et("error",e),Et("load",e);var s=!1,l=!1,u;for(u in a)if(a.hasOwnProperty(u)){var p=a[u];if(p!=null)switch(u){case"src":s=!0;break;case"srcSet":l=!0;break;case"children":case"dangerouslySetInnerHTML":throw Error(r(137,n));default:qt(e,n,u,p,a,null)}}l&&qt(e,n,"srcSet",a.srcSet,a,null),s&&qt(e,n,"src",a.src,a,null);return;case"input":Et("invalid",e);var A=u=p=l=null,B=null,te=null;for(s in a)if(a.hasOwnProperty(s)){var fe=a[s];if(fe!=null)switch(s){case"name":l=fe;break;case"type":p=fe;break;case"checked":B=fe;break;case"defaultChecked":te=fe;break;case"value":u=fe;break;case"defaultValue":A=fe;break;case"children":case"dangerouslySetInnerHTML":if(fe!=null)throw Error(r(137,n));break;default:qt(e,n,s,fe,a,null)}}am(e,u,A,B,te,p,l,!1);return;case"select":Et("invalid",e),s=p=u=null;for(l in a)if(a.hasOwnProperty(l)&&(A=a[l],A!=null))switch(l){case"value":u=A;break;case"defaultValue":p=A;break;case"multiple":s=A;default:qt(e,n,l,A,a,null)}n=u,a=p,e.multiple=!!s,n!=null?nr(e,!!s,n,!1):a!=null&&nr(e,!!s,a,!0);return;case"textarea":Et("invalid",e),u=l=s=null;for(p in a)if(a.hasOwnProperty(p)&&(A=a[p],A!=null))switch(p){case"value":s=A;break;case"defaultValue":l=A;break;case"children":u=A;break;case"dangerouslySetInnerHTML":if(A!=null)throw Error(r(91));break;default:qt(e,n,p,A,a,null)}rm(e,s,l,u);return;case"option":for(B in a)if(a.hasOwnProperty(B)&&(s=a[B],s!=null))switch(B){case"selected":e.selected=s&&typeof s!="function"&&typeof s!="symbol";break;default:qt(e,n,B,s,a,null)}return;case"dialog":Et("beforetoggle",e),Et("toggle",e),Et("cancel",e),Et("close",e);break;case"iframe":case"object":Et("load",e);break;case"video":case"audio":for(s=0;s<jo.length;s++)Et(jo[s],e);break;case"image":Et("error",e),Et("load",e);break;case"details":Et("toggle",e);break;case"embed":case"source":case"link":Et("error",e),Et("load",e);case"area":case"base":case"br":case"col":case"hr":case"keygen":case"meta":case"param":case"track":case"wbr":case"menuitem":for(te in a)if(a.hasOwnProperty(te)&&(s=a[te],s!=null))switch(te){case"children":case"dangerouslySetInnerHTML":throw Error(r(137,n));default:qt(e,n,te,s,a,null)}return;default:if(Fu(n)){for(fe in a)a.hasOwnProperty(fe)&&(s=a[fe],s!==void 0&&zd(e,n,fe,s,a,void 0));return}}for(A in a)a.hasOwnProperty(A)&&(s=a[A],s!=null&&qt(e,n,A,s,a,null))}var Vy={};function ky(e,n,a,s){switch(n){case"div":case"span":case"svg":case"path":case"a":case"g":case"p":case"li":break;case"input":var l=null,u=null,p=null,A=null,B=null,te=null,fe=null;for(oe in a){var ye=a[oe];if(a.hasOwnProperty(oe)&&ye!=null)switch(oe){case"checked":break;case"value":break;case"defaultValue":B=ye;default:s.hasOwnProperty(oe)||qt(e,n,oe,null,s,ye)}}for(var J in s){var oe=s[J];if(ye=a[J],s.hasOwnProperty(J)&&(oe!=null||ye!=null))switch(J){case"type":oe!==ye&&(vt=!0),u=oe;break;case"name":oe!==ye&&(vt=!0),l=oe;break;case"checked":oe!==ye&&(vt=!0),te=oe;break;case"defaultChecked":oe!==ye&&(vt=!0),fe=oe;break;case"value":oe!==ye&&(vt=!0),p=oe;break;case"defaultValue":oe!==ye&&(vt=!0),A=oe;break;case"children":case"dangerouslySetInnerHTML":if(oe!=null)throw Error(r(137,n));break;default:oe!==ye&&qt(e,n,J,oe,s,ye)}}Bu(e,p,A,B,te,fe,u,l);return;case"select":oe=p=A=J=null;for(u in a)if(B=a[u],a.hasOwnProperty(u)&&B!=null)switch(u){case"value":break;case"multiple":oe=B;default:s.hasOwnProperty(u)||qt(e,n,u,null,s,B)}for(l in s)if(u=s[l],B=a[l],s.hasOwnProperty(l)&&(u!=null||B!=null))switch(l){case"value":u!==B&&(vt=!0),J=u;break;case"defaultValue":u!==B&&(vt=!0),A=u;break;case"multiple":u!==B&&(vt=!0),p=u;default:u!==B&&qt(e,n,l,u,s,B)}n=A,a=p,s=oe,J!=null?nr(e,!!a,J,!1):!!s!=!!a&&(n!=null?nr(e,!!a,n,!0):nr(e,!!a,a?[]:"",!1));return;case"textarea":oe=J=null;for(A in a)if(l=a[A],a.hasOwnProperty(A)&&l!=null&&!s.hasOwnProperty(A))switch(A){case"value":break;case"children":break;default:qt(e,n,A,null,s,l)}for(p in s)if(l=s[p],u=a[p],s.hasOwnProperty(p)&&(l!=null||u!=null))switch(p){case"value":l!==u&&(vt=!0),J=l;break;case"defaultValue":l!==u&&(vt=!0),oe=l;break;case"children":break;case"dangerouslySetInnerHTML":if(l!=null)throw Error(r(91));break;default:l!==u&&qt(e,n,p,l,s,u)}sm(e,J,oe);return;case"option":for(var Oe in a)if(J=a[Oe],a.hasOwnProperty(Oe)&&J!=null&&!s.hasOwnProperty(Oe))switch(Oe){case"selected":e.selected=!1;break;default:qt(e,n,Oe,null,s,J)}for(B in s)if(J=s[B],oe=a[B],s.hasOwnProperty(B)&&J!==oe&&(J!=null||oe!=null))switch(B){case"selected":J!==oe&&(vt=!0),e.selected=J&&typeof J!="function"&&typeof J!="symbol";break;default:qt(e,n,B,J,s,oe)}return;case"img":case"link":case"area":case"base":case"br":case"col":case"embed":case"hr":case"keygen":case"meta":case"param":case"source":case"track":case"wbr":case"menuitem":for(var Qe in a)J=a[Qe],a.hasOwnProperty(Qe)&&J!=null&&!s.hasOwnProperty(Qe)&&qt(e,n,Qe,null,s,J);for(te in s)if(J=s[te],oe=a[te],s.hasOwnProperty(te)&&J!==oe&&(J!=null||oe!=null))switch(te){case"children":case"dangerouslySetInnerHTML":if(J!=null)throw Error(r(137,n));break;default:qt(e,n,te,J,s,oe)}return;default:if(Fu(n)){for(var ht in a)J=a[ht],a.hasOwnProperty(ht)&&J!==void 0&&!s.hasOwnProperty(ht)&&zd(e,n,ht,void 0,s,J);for(fe in s)J=s[fe],oe=a[fe],!s.hasOwnProperty(fe)||J===oe||J===void 0&&oe===void 0||zd(e,n,fe,J,s,oe);return}}for(var ee in a)J=a[ee],a.hasOwnProperty(ee)&&J!=null&&!s.hasOwnProperty(ee)&&qt(e,n,ee,null,s,J);for(ye in s)J=s[ye],oe=a[ye],!s.hasOwnProperty(ye)||J===oe||J==null&&oe==null||qt(e,n,ye,J,s,oe)}function p_(e){switch(e){case"css":case"script":case"font":case"img":case"image":case"input":case"link":return!0;default:return!1}}function Xy(){if(typeof performance.getEntriesByType=="function"){for(var e=0,n=0,a=performance.getEntriesByType("resource"),s=0;s<a.length;s++){var l=a[s],u=l.transferSize,p=l.initiatorType,A=l.duration;if(u&&A&&p_(p)){for(p=0,A=l.responseEnd,s+=1;s<a.length;s++){var B=a[s],te=B.startTime;if(te>A)break;var fe=B.transferSize,ye=B.initiatorType;fe&&p_(ye)&&(B=B.responseEnd,p+=fe*(B<A?1:(A-te)/(B-te)))}if(--s,n+=8*(u+p)/(l.duration/1e3),e++,10<e)break}}if(0<e)return n/e/1e6}return navigator.connection&&(e=navigator.connection.downlink,typeof e=="number")?e:5}var Fd=null,Hd=null;function Ko(e){return e.nodeType===9?e:e.ownerDocument}function m_(e){switch(e){case"http://www.w3.org/2000/svg":return 1;case"http://www.w3.org/1998/Math/MathML":return 2;default:return 0}}function g_(e,n){if(e===0)switch(n){case"svg":return 1;case"math":return 2;default:return 0}return e===1&&n==="foreignObject"?0:e}function __(e,n,a,s){return a=Ko(a).createElement(e),a[T]=s,a[G]=n,Nn(a,e,n),_t(a),a}function Gd(e,n){return e==="textarea"||e==="noscript"||typeof n.children=="string"||typeof n.children=="number"||typeof n.children=="bigint"||typeof n.dangerouslySetInnerHTML=="object"&&n.dangerouslySetInnerHTML!==null&&n.dangerouslySetInnerHTML.__html!=null}var Vd=null;function Wy(){var e=window.event;return e&&e.type==="popstate"?e===Vd?!1:(Vd=e,!0):(Vd=null,!1)}var kd=typeof setTimeout=="function"?setTimeout:void 0,qy=typeof clearTimeout=="function"?clearTimeout:void 0,v_=typeof Promise=="function"?Promise:void 0,x_=typeof requestAnimationFrame=="function"?requestAnimationFrame:kd,Yy=typeof queueMicrotask=="function"?queueMicrotask:typeof v_<"u"?function(e){return v_.resolve(null).then(e).catch(jy)}:kd;function jy(e){setTimeout(function(){throw e})}function ss(e){return e==="head"}function S_(e,n){var a=n,s=0;do{var l=a.nextSibling;if(e.removeChild(a),l&&l.nodeType===8)if(a=l.data,a==="/$"||a==="/&"){if(s===0){e.removeChild(l),Hr(n);return}s--}else if(a==="$"||a==="$?"||a==="$~"||a==="$!"||a==="&")s++;else if(a==="html")Qd(e.ownerDocument.documentElement);else if(a==="head"){a=e.ownerDocument.head,Qd(a);for(var u=a.firstChild;u;){var p=u.nextSibling,A=u.nodeName;u[Le]||A==="SCRIPT"||A==="STYLE"||A==="LINK"&&u.rel.toLowerCase()==="stylesheet"||a.removeChild(u),u=p}}else a==="body"&&Qd(e.ownerDocument.body);a=l}while(a);Hr(n)}function y_(e,n){var a=e;e=0;do{var s=a.nextSibling;if(a.nodeType===1?n?(a._stashedDisplay=a.style.display,a.style.display="none"):(a.style.display=a._stashedDisplay||"",a.getAttribute("style")===""&&a.removeAttribute("style")):a.nodeType===3&&(n?(a._stashedText=a.nodeValue,a.nodeValue=""):a.nodeValue=a._stashedText||""),s&&s.nodeType===8)if(a=s.data,a==="/$"){if(e===0)break;e--}else a!=="$"&&a!=="$?"&&a!=="$~"&&a!=="$!"||e++;a=s}while(a)}function M_(e,n,a){if(n=CSS.escape(n)!==n?"r-"+btoa(n).replace(/=/g,""):n,e.style.viewTransitionName=n,a!=null&&(e.style.viewTransitionClass=a),a=getComputedStyle(e),a.display==="inline"){if(n=e.getClientRects(),n.length===1)var s=1;else for(var l=s=0;l<n.length;l++){var u=n[l];0<u.width&&0<u.height&&s++}s===1&&(e=e.style,e.display=n.length===1?"inline-block":"block",e.marginTop="-"+a.paddingTop,e.marginBottom="-"+a.paddingBottom)}}function E_(e,n){e=e.style,n=n.style;var a=n!=null?n.hasOwnProperty("viewTransitionName")?n.viewTransitionName:n.hasOwnProperty("view-transition-name")?n["view-transition-name"]:null:null;e.viewTransitionName=a==null||typeof a=="boolean"?"":(""+a).trim(),a=n!=null?n.hasOwnProperty("viewTransitionClass")?n.viewTransitionClass:n.hasOwnProperty("view-transition-class")?n["view-transition-class"]:null:null,e.viewTransitionClass=a==null||typeof a=="boolean"?"":(""+a).trim(),e.display==="inline-block"&&(n==null?e.display=e.margin="":(a=n.display,e.display=a==null||typeof a=="boolean"?"":a,a=n.margin,a!=null?e.margin=a:(a=n.hasOwnProperty("marginTop")?n.marginTop:n["margin-top"],e.marginTop=a==null||typeof a=="boolean"?"":a,n=n.hasOwnProperty("marginBottom")?n.marginBottom:n["margin-bottom"],e.marginBottom=n==null||typeof n=="boolean"?"":n)))}function Zy(e,n,a){return a=a.ownerDocument.defaultView,{rect:e,abs:n.position==="absolute"||n.position==="fixed",clip:n.clipPath!=="none"||n.overflow!=="visible"||n.filter!=="none"||n.mask!=="none"||n.mask!=="none"||n.borderRadius!=="0px",view:0<=e.bottom&&0<=e.right&&e.top<=a.innerHeight&&e.left<=a.innerWidth}}function Xd(e){var n=e.getBoundingClientRect(),a=getComputedStyle(e);return Zy(n,a,e)}function Ky(e){return e.documentElement.clientHeight}function Qy(e){this.addEventListener("load",e),this.addEventListener("error",e)}function Jy(e,n,a,s,l,u,p,A,B){var te=n.nodeType===9?n:n.ownerDocument;try{var fe=te.startViewTransition({update:function(){var J=te.defaultView,oe=J.navigation&&J.navigation.transition,Oe=te.fonts.status;s();var Qe=[];if(Oe==="loaded"&&(Ky(te),te.fonts.status==="loading"&&Qe.push(te.fonts.ready)),Oe=Qe.length,e!==null)for(var ht=e.suspenseyImages,ee=0,V=0;V<ht.length;V++){var ae=ht[V];if(!ae.complete){var Se=ae.getBoundingClientRect();if(0<Se.bottom&&0<Se.right&&Se.top<J.innerHeight&&Se.left<J.innerWidth){if(ee+=X_(ae),ee>Hc){Qe.length=Oe;break}ae=new Promise(Qy.bind(ae)),Qe.push(ae)}}}if(0<Qe.length)return J=Promise.race([Promise.all(Qe),new Promise(function(qe){return setTimeout(qe,500)})]).then(l,l),(oe?Promise.allSettled([oe.finished,J]):J).then(u,u);if(l(),oe)return oe.finished.then(u,u);u()},types:a});te.__reactViewTransition=fe;var ye=[];return fe.ready.then(function(){for(var J=te.documentElement.getAnimations({subtree:!0}),oe=0;oe<J.length;oe++){var Oe=J[oe],Qe=Oe.effect,ht=Qe.pseudoElement;if(ht!=null&&ht.startsWith("::view-transition")){ye.push(Oe),Oe=Qe.getKeyframes();for(var ee=ht=void 0,V=!0,ae=0;ae<Oe.length;ae++){var Se=Oe[ae],qe=Se.width;if(ht===void 0)ht=qe;else if(ht!==qe){V=!1;break}if(qe=Se.height,ee===void 0)ee=qe;else if(ee!==qe){V=!1;break}delete Se.width,delete Se.height,Se.transform==="none"&&delete Se.transform}V&&ht!==void 0&&ee!==void 0&&(Qe.setKeyframes(Oe),V=getComputedStyle(Qe.target,Qe.pseudoElement),V.width!==ht||V.height!==ee)&&(V=Oe[0],V.width=ht,V.height=ee,V=Oe[Oe.length-1],V.width=ht,V.height=ee,Qe.setKeyframes(Oe))}}p()},function(J){te.__reactViewTransition===fe&&(te.__reactViewTransition=null);try{if(typeof J=="object"&&J!==null)switch(J.name){case"InvalidStateError":(J.message==="View transition was skipped because document visibility state is hidden."||J.message==="Skipping view transition because document visibility state has become hidden."||J.message==="Skipping view transition because viewport size changed."||J.message==="Transition was aborted because of invalid state")&&(J=null)}J!==null&&B(J)}finally{s(),l(),p()}}),fe.finished.finally(function(){for(var J=0;J<ye.length;J++)ye[J].cancel();te.__reactViewTransition===fe&&(te.__reactViewTransition=null),A()}),fe}catch{return s(),l(),p(),null}}function Vs(e,n){this._scope=document.documentElement,this._selector="::view-transition-"+e+"("+n+")"}Vs.prototype.animate=function(e,n){return n=typeof n=="number"?{duration:n}:I({},n),n.pseudoElement=this._selector,this._scope.animate(e,n)},Vs.prototype.getAnimations=function(){for(var e=this._scope,n=this._selector,a=e.getAnimations({subtree:!0}),s=[],l=0;l<a.length;l++){var u=a[l].effect;u!==null&&u.target===e&&u.pseudoElement===n&&s.push(a[l])}return s},Vs.prototype.getComputedStyle=function(){return getComputedStyle(this._scope,this._selector)};function b_(e){return{name:e,group:new Vs("group",e),imagePair:new Vs("image-pair",e),old:new Vs("old",e),new:new Vs("new",e)}}function oi(e){this._fragmentFiber=e,this._observers=this._eventListeners=null}oi.prototype.addEventListener=function(e,n,a){var s=null,l=null;if(!(a!=null&&typeof a!="boolean"&&(s=a.signal||null,s!==null&&s.aborted))){this._eventListeners===null&&(this._eventListeners=[]);var u=this._eventListeners;if(A_(u,e,n,a)===-1){var p=this,A=n;a!=null&&typeof a!="boolean"&&a.once===!0&&(A=function(B){p.removeEventListener(e,n,a),typeof n=="function"?n.call(this,B):n.handleEvent(B)}),s!==null&&(l=p.removeEventListener.bind(p,e,n,a),s.addEventListener("abort",l,{once:!0}),l=s.removeEventListener.bind(s,"abort",l)),s=Or(a),u.push({type:e,listener:n,optionsOrUseCapture:a,attachedListener:A,cleanup:l}),_(this._fragmentFiber.child,!1,$y,e,A,s)}this._eventListeners=u}};function $y(e,n,a,s){return y(e).addEventListener(n,a,s),!1}oi.prototype.removeEventListener=function(e,n,a){var s=this._eventListeners;if(s!==null&&(n=A_(s,e,n,a),n!==-1)){var l=s[n];a=l.attachedListener;var u=l.cleanup;l=Or(l.optionsOrUseCapture),_(this._fragmentFiber.child,!1,eM,e,a,l),s.splice(n,1),u!==null&&u()}};function eM(e,n,a,s){return y(e).removeEventListener(n,a,s),!1}function Or(e){return e!=null&&typeof e!="boolean"&&(e.once===!0||e.signal instanceof AbortSignal)?{capture:e.capture,passive:e.passive}:e}function T_(e){return e==null?"c=0":typeof e=="boolean"?"c="+(e?"1":"0"):"c="+(e.capture?"1":"0")}function A_(e,n,a,s){if(e.length===0)return-1;s=T_(s);for(var l=0;l<e.length;l++){var u=e[l];if(u.type===n&&u.listener===a&&T_(u.optionsOrUseCapture)===s)return l}return-1}oi.prototype.dispatchEvent=function(e){var n=v(this._fragmentFiber);if(n===null)return!0;n=y(n);var a=this._eventListeners;if(a!==null&&0<a.length||!e.bubbles){var s=n.nodeType===9?n.createComment(""):document.createTextNode("");if(a)for(var l=0;l<a.length;l++){var u=a[l];s.addEventListener(u.type,u.attachedListener,Or(u.optionsOrUseCapture))}if(n.appendChild(s),e=s.dispatchEvent(e),a)for(l=0;l<a.length;l++)u=a[l],s.removeEventListener(u.type,u.attachedListener,Or(u.optionsOrUseCapture));return n.removeChild(s),e}return n.dispatchEvent(e)},oi.prototype.focus=function(e){_(this._fragmentFiber.child,!0,R_,e,void 0,void 0)};function R_(e,n){return e.tag===6?!1:(e=y(e),dM(e,n))}oi.prototype.focusLast=function(e){var n=[];_(this._fragmentFiber.child,!0,Wd,n,void 0,void 0);for(var a=n.length-1;0<=a&&!R_(n[a],e);a--);};function Wd(e,n){return n.push(e),!1}oi.prototype.blur=function(){var e=v(this._fragmentFiber);e!==null&&(e=y(e),e=Ko(e).activeElement,e!==null&&_(this._fragmentFiber.child,!1,tM,e,void 0,void 0))};function tM(e,n){return e.tag===6?!1:(e=y(e),e===n||e.contains(n)?(n.blur(),!0):!1)}oi.prototype.observeUsing=function(e){this._observers===null&&(this._observers=new Set),this._observers.add(e),_(this._fragmentFiber.child,!1,nM,e,void 0,void 0)};function nM(e,n){return e.tag===6||(e=y(e),n.observe(e)),!1}oi.prototype.unobserveUsing=function(e){var n=this._observers;if(n!==null&&n.has(e)){n.delete(e),_(this._fragmentFiber.child,!1,iM,e,void 0,void 0);for(var a=n=0;a<Di.length;a++){var s=Di[a];s.fragmentInstance===this&&s.observer===e?e.unobserve(s.instance):Di[n++]=s}Di.length=n}};function iM(e,n){return e.tag===6||(e=y(e),n.unobserve(e)),!1}var Di=[],qd=!1;function aM(e,n,a){Di.push({fragmentInstance:e,observer:n,instance:a}),qd||(qd=!0,hM(function(){qd=!1;var s=Di;Di=[];for(var l=0;l<s.length;l++){var u=s[l];u.observer.unobserve(u.instance)}}))}oi.prototype.getClientRects=function(){var e=[];return _(this._fragmentFiber.child,!1,sM,e,void 0,void 0),e};function sM(e,n){if(e.tag===6){e=e.stateNode;var a=e.ownerDocument.createRange();a.selectNodeContents(e),n.push.apply(n,a.getClientRects())}else e=y(e),n.push.apply(n,e.getClientRects());return!1}oi.prototype.getRootNode=function(e){var n=v(this._fragmentFiber);return n===null?this:y(n).getRootNode(e)},oi.prototype.compareDocumentPosition=function(e){var n=v(this._fragmentFiber);if(n===null)return Node.DOCUMENT_POSITION_DISCONNECTED;var a=[];_(this._fragmentFiber.child,!1,Wd,a,void 0,void 0);var s=y(n);if(a.length===0){if(a=s,E(this._fragmentFiber)){e:{for(n=this._fragmentFiber.return;n!==null;){if(n.tag===4){n=n.stateNode.containerInfo;break e}if(n.tag===3||n.tag===5||n.tag===27)break;n=n.return}n=null}n!=null&&(a=n)}n=this._fragmentFiber;var l=s=a.compareDocumentPosition(e);return a===e?l=Node.DOCUMENT_POSITION_CONTAINS:s&Node.DOCUMENT_POSITION_CONTAINED_BY&&(a=R(n)[1],a===null?l=Node.DOCUMENT_POSITION_PRECEDING:(e=y(a).compareDocumentPosition(e),l=e===0||e&Node.DOCUMENT_POSITION_FOLLOWING?Node.DOCUMENT_POSITION_FOLLOWING:Node.DOCUMENT_POSITION_PRECEDING)),l|=Node.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC}n=y(a[0]),l=y(a[a.length-1]);var u=E(this._fragmentFiber)?n.parentElement:s;if(u==null)return Node.DOCUMENT_POSITION_DISCONNECTED;s=u.compareDocumentPosition(n)&Node.DOCUMENT_POSITION_CONTAINED_BY,u=u.compareDocumentPosition(l)&Node.DOCUMENT_POSITION_CONTAINED_BY;var p=n.compareDocumentPosition(e),A=l.compareDocumentPosition(e),B=p&Node.DOCUMENT_POSITION_CONTAINED_BY||A&Node.DOCUMENT_POSITION_CONTAINED_BY;return A=s&&u&&p&Node.DOCUMENT_POSITION_FOLLOWING&&A&Node.DOCUMENT_POSITION_PRECEDING,n=s&&n===e||u&&l===e||B||A?Node.DOCUMENT_POSITION_CONTAINED_BY:!s&&n===e||!u&&l===e?Node.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC:p,n&Node.DOCUMENT_POSITION_DISCONNECTED||n&Node.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC||rM(n,this._fragmentFiber,a[0],a[a.length-1],e)?n:Node.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC};function rM(e,n,a,s,l){var u=st(l);if(e&Node.DOCUMENT_POSITION_CONTAINED_BY){if(a=!!u)e:{for(;u!==null;){if(u.tag===7&&(u===n||u.alternate===n)){a=!0;break e}u=u.return}a=!1}return a}if(e&Node.DOCUMENT_POSITION_CONTAINS){if(u===null)return u=l.ownerDocument,l===u||l===u.documentElement||l===u.body;e:{for(u=n,n=v(n);u!==null;){if(!(u.tag!==5&&u.tag!==3&&u.tag!==27||u!==n&&u.alternate!==n)){u=!0;break e}u=u.return}u=!1}return u}return e&Node.DOCUMENT_POSITION_PRECEDING?((n=!!u)&&!(n=u===a)&&(n=U(a,u,D),n===null?n=!1:(_(n,!0,W,u,a),u=S,S=null,n=u!==null)),n):e&Node.DOCUMENT_POSITION_FOLLOWING?((n=!!u)&&!(n=u===s)&&(n=U(s,u,D),n===null?n=!1:(_(n,!0,C,u,s),u=S,z=S=null,n=u!==null)),n):!1}function w_(e,n){var a=e.ownerDocument.createRange();a.selectNodeContents(e),e=a.getBoundingClientRect(),window.scrollTo(window.scrollX+e.left,n?window.scrollY+e.top:window.scrollY+e.bottom-window.innerHeight)}oi.prototype.scrollIntoView=function(e){if(typeof e=="object")throw Error(r(566));var n=[];_(this._fragmentFiber.child,!1,Wd,n,void 0,void 0);var a=e!==!1;if(n.length===0){var s=R(this._fragmentFiber);if(s=a?s[1]||s[0]||v(this._fragmentFiber):s[0]||s[1],s===null)return;if(s.tag===6){e=y(s),w_(e,a);return}if(s=y(s),s.nodeType!==9){if(s.nodeType===11){a="host"in s?s.host:null,a!==null&&a.scrollIntoView(e);return}s.scrollIntoView(e)}}for(s=a?n.length-1:0;s!==(a?-1:n.length);){var l=n[s];l.tag===6?(l=y(l),w_(l,a)):y(l).scrollIntoView(e),s+=a?-1:1}};function oM(e,n){return e=y(e),C_(e,n),!1}function C_(e,n){e.reactFragments==null&&(e.reactFragments=new Set),e.reactFragments.add(n)}function N_(e,n){var a=n._eventListeners;if(a!==null)for(var s=0;s<a.length;s++){var l=a[s];e.addEventListener(l.type,l.attachedListener,Or(l.optionsOrUseCapture))}e.nodeType!==3&&(a=n._observers,a!==null&&a.forEach(function(u){for(var p=0,A=0;A<Di.length;A++){var B=Di[A];(B.fragmentInstance!==n||B.observer!==u||B.instance!==e)&&(Di[p++]=B)}Di.length=p,u.observe(e)}),C_(e,n))}function lM(e,n){var a=n._eventListeners;if(a!==null)for(var s=0;s<a.length;s++){var l=a[s];e.removeEventListener(l.type,l.attachedListener,Or(l.optionsOrUseCapture))}e.nodeType!==3&&(a=n._observers,a!==null&&a.forEach(function(u){typeof u.rootMargin=="string"?aM(n,u,e):u.unobserve(e)}),e.reactFragments!=null&&e.reactFragments.delete(n))}function Yd(e){var n=e.firstChild;for(n&&n.nodeType===10&&(n=n.nextSibling);n;){var a=n;switch(n=n.nextSibling,a.nodeName){case"HTML":case"HEAD":case"BODY":Yd(a),Ke(a);continue;case"SCRIPT":case"STYLE":continue;case"LINK":if(a.rel.toLowerCase()==="stylesheet")continue}e.removeChild(a)}}function cM(e,n,a,s){for(;e.nodeType===1;){var l=a;if(e.nodeName.toLowerCase()!==n.toLowerCase()){if(!s&&(e.nodeName!=="INPUT"||e.type!=="hidden"))break}else if(s){if(!e[Le])switch(n){case"meta":if(!e.hasAttribute("itemprop"))break;return e;case"link":if(u=e.getAttribute("rel"),u==="stylesheet"&&e.hasAttribute("data-precedence"))break;if(u!==l.rel||e.getAttribute("href")!==(l.href==null||l.href===""?null:l.href)||e.getAttribute("crossorigin")!==(l.crossOrigin==null?null:l.crossOrigin)||e.getAttribute("title")!==(l.title==null?null:l.title))break;return e;case"style":if(e.hasAttribute("data-precedence"))break;return e;case"script":if(u=e.getAttribute("src"),(u!==(l.src==null?null:l.src)||e.getAttribute("type")!==(l.type==null?null:l.type)||e.getAttribute("crossorigin")!==(l.crossOrigin==null?null:l.crossOrigin))&&u&&e.hasAttribute("async")&&!e.hasAttribute("itemprop"))break;return e;default:return e}}else if(n==="input"&&e.type==="hidden"){var u=l.name==null?null:""+l.name;if(l.type==="hidden"&&e.getAttribute("name")===u)return e}else return e;if(e=yi(e.nextSibling),e===null)break}return null}function uM(e,n,a){if(n==="")return null;for(;e.nodeType!==3;)if((e.nodeType!==1||e.nodeName!=="INPUT"||e.type!=="hidden")&&!a||(e=yi(e.nextSibling),e===null))return null;return e}function D_(e,n){for(;e.nodeType!==8;)if((e.nodeType!==1||e.nodeName!=="INPUT"||e.type!=="hidden")&&!n||(e=yi(e.nextSibling),e===null))return null;return e}function jd(e){return e.data==="$?"||e.data==="$~"}function Zd(e){return e.data==="$!"||e.data==="$?"&&e.ownerDocument.readyState!=="loading"}function fM(e,n){var a=e.ownerDocument;if(e.data==="$~")e._reactRetry=n;else if(e.data!=="$?"||a.readyState!=="loading")n();else{var s=function(){n(),a.removeEventListener("DOMContentLoaded",s)};a.addEventListener("DOMContentLoaded",s),e._reactRetry=s}}function yi(e){for(;e!=null;e=e.nextSibling){var n=e.nodeType;if(n===1||n===3)break;if(n===8){if(n=e.data,n==="$"||n==="$!"||n==="$?"||n==="$~"||n==="&"||n==="F!"||n==="F")break;if(n==="/$"||n==="/&")return null}}return e}var Kd=null;function U_(e){e=e.nextSibling;for(var n=0;e;){if(e.nodeType===8){var a=e.data;if(a==="/$"||a==="/&"){if(n===0)return yi(e.nextSibling);n--}else a!=="$"&&a!=="$!"&&a!=="$?"&&a!=="$~"&&a!=="&"||n++}e=e.nextSibling}return null}function L_(e){e=e.previousSibling;for(var n=0;e;){if(e.nodeType===8){var a=e.data;if(a==="$"||a==="$!"||a==="$?"||a==="$~"||a==="&"){if(n===0)return e;n--}else a!=="/$"&&a!=="/&"||n++}e=e.previousSibling}return null}function dM(e,n){function a(){s=!0}if(e.ownerDocument.activeElement===e)return!0;var s=!1;try{e.ownerDocument.addEventListener("focus",a,!0),(e.focus||HTMLElement.prototype.focus).call(e,n)}finally{e.ownerDocument.removeEventListener("focus",a,!0)}return s}function hM(e){x_(function(){x_(function(n){return e(n)})})}function O_(e,n,a){switch(n=Ko(a),e){case"html":if(e=n.documentElement,!e)throw Error(r(452));return e;case"head":if(e=n.head,!e)throw Error(r(453));return e;case"body":if(e=n.body,!e)throw Error(r(454));return e;default:throw Error(r(451))}}function P_(e,n,a){for(var s in a){var l=a[s];a.hasOwnProperty(s)&&l!=null&&qt(e,n,s,null,Vy,l)}a.dangerouslySetInnerHTML!=null&&(e.textContent=""),e.onclick===Gi&&(e.onclick=null),Ke(e)}function Qd(e){for(var n=e.attributes;n.length;)e.removeAttributeNode(n[0]);Ke(e)}var Mi=new Map,I_=new Set;function Qo(e){if(typeof e.getRootNode=="function"){var n=e.getRootNode();if(n.nodeType===9||n.nodeType===11)return n}return e.nodeType===9?e:e.ownerDocument}var Ta=we.d;we.d={f:pM,r:mM,D:gM,C:_M,L:vM,m:xM,X:yM,S:SM,M:MM};function pM(){var e=Ta.f(),n=Dc();return e||n}function mM(e){var n=ut(e);n!==null&&n.tag===5&&n.type==="form"?zg(n):Ta.r(e)}var Pr=typeof document>"u"?null:document;function B_(e,n,a){var s=Pr;if(s&&typeof n=="string"&&n){var l=pi(n);l='link[rel="'+e+'"][href="'+l+'"]',typeof a=="string"&&(l+='[crossorigin="'+a+'"]'),I_.has(l)||(I_.add(l),e={rel:e,crossOrigin:a,href:n},s.querySelector(l)===null&&(n=s.createElement("link"),Nn(n,"link",e),_t(n),s.head.appendChild(n)))}}function gM(e){Ta.D(e),B_("dns-prefetch",e,null)}function _M(e,n){Ta.C(e,n),B_("preconnect",e,n)}function vM(e,n,a){Ta.L(e,n,a);var s=Pr;if(s&&e&&n){var l='link[rel="preload"][as="'+pi(n)+'"]';n==="image"&&a&&a.imageSrcSet?(l+='[imagesrcset="'+pi(a.imageSrcSet)+'"]',typeof a.imageSizes=="string"&&(l+='[imagesizes="'+pi(a.imageSizes)+'"]')):l+='[href="'+pi(e)+'"]';var u=l;switch(n){case"style":u=Ir(e);break;case"script":u=Br(e)}if(!(Mi.has(u)||(e=I({rel:"preload",href:n==="image"&&a&&a.imageSrcSet?void 0:e,as:n},a),Mi.set(u,e),s.querySelector(l)!==null||n==="style"&&s.querySelector(Jo(u))||n==="script"&&s.querySelector($o(u))))){var p=s.createElement("link");Nn(p,"link",e),n==="style"&&(p[Ye]=!0,p.onload=p.onerror=function(){jt(p)}),_t(p),s.head.appendChild(p)}}}function xM(e,n){Ta.m(e,n);var a=Pr;if(a&&e){var s=n&&typeof n.as=="string"?n.as:"script",l='link[rel="modulepreload"][as="'+pi(s)+'"][href="'+pi(e)+'"]',u=l;switch(s){case"audioworklet":case"paintworklet":case"serviceworker":case"sharedworker":case"worker":case"script":u=Br(e)}if(!Mi.has(u)&&(e=I({rel:"modulepreload",href:e},n),Mi.set(u,e),a.querySelector(l)===null)){switch(s){case"audioworklet":case"paintworklet":case"serviceworker":case"sharedworker":case"worker":case"script":if(a.querySelector($o(u)))return}s=a.createElement("link"),Nn(s,"link",e),_t(s),a.head.appendChild(s)}}}function SM(e,n,a){Ta.S(e,n,a);var s=Pr;if(s&&e){var l=yt(s).hoistableStyles,u=Ir(e);n=n||"default";var p=l.get(u);if(!p){var A={loading:0,preload:null};if(p=s.querySelector(Jo(u)))A.loading=5;else{e=I({rel:"stylesheet",href:e,"data-precedence":n},a),(a=Mi.get(u))&&Jd(e,a);var B=p=s.createElement("link");_t(B),Nn(B,"link",e),B._p=new Promise(function(te,fe){B.onload=te,B.onerror=fe}),B.addEventListener("load",function(){A.loading|=1}),B.addEventListener("error",function(){A.loading|=2}),A.loading|=4,zc(p,n,s)}p={type:"stylesheet",instance:p,count:1,state:A},l.set(u,p)}}}function yM(e,n){Ta.X(e,n);var a=Pr;if(a&&e){var s=yt(a).hoistableScripts,l=Br(e),u=s.get(l);u||(u=a.querySelector($o(l)),u||(e=I({src:e,async:!0},n),(n=Mi.get(l))&&$d(e,n),u=a.createElement("script"),_t(u),Nn(u,"link",e),a.head.appendChild(u)),u={type:"script",instance:u,count:1,state:null},s.set(l,u))}}function MM(e,n){Ta.M(e,n);var a=Pr;if(a&&e){var s=yt(a).hoistableScripts,l=Br(e),u=s.get(l);u||(u=a.querySelector($o(l)),u||(e=I({src:e,async:!0,type:"module"},n),(n=Mi.get(l))&&$d(e,n),u=a.createElement("script"),_t(u),Nn(u,"link",e),a.head.appendChild(u)),u={type:"script",instance:u,count:1,state:null},s.set(l,u))}}function z_(e,n,a,s){var l=(l=Pt.current)?Qo(l):null;if(!l)throw Error(r(446));switch(e){case"meta":case"title":return null;case"style":return typeof a.precedence=="string"&&typeof a.href=="string"?(a=Ir(a.href),n=yt(l).hoistableStyles,s=n.get(a),s||(s={type:"style",instance:null,count:0,state:null},n.set(a,s)),s):{type:"void",instance:null,count:0,state:null};case"link":if(a.rel==="stylesheet"&&typeof a.href=="string"&&typeof a.precedence=="string"){e=Ir(a.href);var u=yt(l).hoistableStyles,p=u.get(e);if(p||(l=l.ownerDocument||l,p={type:"stylesheet",instance:null,count:0,state:{loading:0,preload:null}},u.set(e,p),(u=l.querySelector(Jo(e)))?u._p||(p.instance=u,p.state.loading=5):(u=Mi.get(e),u||(u={rel:"preload",as:"style",href:a.href,crossOrigin:a.crossOrigin,integrity:a.integrity,media:a.media,hrefLang:a.hrefLang,referrerPolicy:a.referrerPolicy},Mi.set(e,u)),EM(l,e,u,p.state))),n&&s===null)throw Error(r(528,""));return p}if(n&&s!==null)throw Error(r(529,""));return null;case"script":return n=a.async,a=a.src,typeof a=="string"&&n&&typeof n!="function"&&typeof n!="symbol"?(a=Br(a),n=yt(l).hoistableScripts,s=n.get(a),s||(s={type:"script",instance:null,count:0,state:null},n.set(a,s)),s):{type:"void",instance:null,count:0,state:null};default:throw Error(r(444,e))}}function Ir(e){return'href="'+pi(e)+'"'}function Jo(e){return'link[rel="stylesheet"]['+e+"]"}function F_(e){return I({},e,{"data-precedence":e.precedence,precedence:null})}function EM(e,n,a,s){if(n=e.querySelector('link[rel="preload"][as="style"]['+n+"]")){if(n[Ye]!==!0){s.loading=1;return}}else n=e.createElement("link"),n[Ye]=!0,n.onload=n.onerror=jt.bind(null,n),Nn(n,"link",a),_t(n),e.head.appendChild(n);s.preload=n,n.addEventListener("load",function(){return s.loading|=1}),n.addEventListener("error",function(){return s.loading|=2})}function Br(e){return'[src="'+pi(e)+'"]'}function $o(e){return"script[async]"+e}function H_(e,n,a){if(n.count++,n.instance===null)switch(n.type){case"style":var s=e.querySelector('style[data-href~="'+pi(a.href)+'"]');if(s)return n.instance=s,_t(s),s;var l=I({},a,{"data-href":a.href,"data-precedence":a.precedence,href:null,precedence:null});return s=(e.ownerDocument||e).createElement("style"),_t(s),Nn(s,"style",l),zc(s,a.precedence,e),n.instance=s;case"stylesheet":l=Ir(a.href);var u=e.querySelector(Jo(l));if(u)return n.state.loading|=4,n.instance=u,_t(u),u;s=F_(a),(l=Mi.get(l))&&Jd(s,l),u=(e.ownerDocument||e).createElement("link"),_t(u);var p=u;return p._p=new Promise(function(A,B){p.onload=A,p.onerror=B}),Nn(u,"link",s),n.state.loading|=4,zc(u,a.precedence,e),n.instance=u;case"script":return u=Br(a.src),(l=e.querySelector($o(u)))?(n.instance=l,_t(l),l):(s=a,(l=Mi.get(u))&&(s=I({},a),$d(s,l)),e=e.ownerDocument||e,l=e.createElement("script"),_t(l),Nn(l,"link",s),e.head.appendChild(l),n.instance=l);case"void":return null;default:throw Error(r(443,n.type))}else n.type==="stylesheet"&&(n.state.loading&4)===0&&(s=n.instance,n.state.loading|=4,zc(s,a.precedence,e));return n.instance}function zc(e,n,a){for(var s=a.querySelectorAll('link[rel="stylesheet"][data-precedence],style[data-precedence]'),l=s.length?s[s.length-1]:null,u=l,p=0;p<s.length;p++){var A=s[p];if(A.dataset.precedence===n)u=A;else if(u!==l)break}u?u.parentNode.insertBefore(e,u.nextSibling):(n=a.nodeType===9?a.head:a,n.insertBefore(e,n.firstChild))}function Jd(e,n){e.crossOrigin==null&&(e.crossOrigin=n.crossOrigin),e.referrerPolicy==null&&(e.referrerPolicy=n.referrerPolicy),e.title==null&&(e.title=n.title)}function $d(e,n){e.crossOrigin==null&&(e.crossOrigin=n.crossOrigin),e.referrerPolicy==null&&(e.referrerPolicy=n.referrerPolicy),e.integrity==null&&(e.integrity=n.integrity)}var Fc=null;function G_(e,n,a){if(Fc===null){var s=new Map,l=Fc=new Map;l.set(a,s)}else l=Fc,s=l.get(a),s||(s=new Map,l.set(a,s));if(s.has(e))return s;for(s.set(e,null),a=a.getElementsByTagName(e),l=0;l<a.length;l++){var u=a[l];if(!(u[Le]||u[T]||e==="link"&&u.getAttribute("rel")==="stylesheet")&&u.namespaceURI!=="http://www.w3.org/2000/svg"){var p=u.getAttribute(n)||"";p=e+p;var A=s.get(p);A?A.push(u):s.set(p,[u])}}return s}function eh(e,n,a){e=e.ownerDocument||e,e.head.insertBefore(a,n==="title"?e.querySelector("head > title"):null)}function bM(e,n,a){if(a===1||n.itemProp!=null)return!1;switch(e){case"meta":case"title":return!0;case"style":if(typeof n.precedence!="string"||typeof n.href!="string"||n.href==="")break;return!0;case"link":if(typeof n.rel!="string"||typeof n.href!="string"||n.href===""||n.onLoad||n.onError)break;switch(n.rel){case"stylesheet":return e=n.disabled,typeof n.precedence=="string"&&e==null;default:return!0}case"script":if(n.async&&typeof n.async!="function"&&typeof n.async!="symbol"&&!n.onLoad&&!n.onError&&n.src&&typeof n.src=="string")return!0}return!1}function V_(e,n){return e==="img"&&n.src!=null&&n.src!==""&&n.onLoad==null&&n.loading!=="lazy"}function k_(e){return!(e.type==="stylesheet"&&(e.state.loading&3)===0)}function X_(e){return(e.width||100)*(e.height||100)*(typeof devicePixelRatio=="number"?devicePixelRatio:1)*.25}function W_(e,n){typeof n.decode=="function"&&(e.imgCount++,n.complete||(e.imgBytes+=X_(n),e.suspenseyImages.push(n)),e=RM.bind(e),n.decode().then(e,e))}function TM(e,n,a,s){if(a.type==="stylesheet"&&(typeof s.media!="string"||matchMedia(s.media).matches!==!1)&&(a.state.loading&4)===0){if(a.instance===null){var l=Ir(s.href),u=n.querySelector(Jo(l));if(u){n=u._p,n!==null&&typeof n=="object"&&typeof n.then=="function"&&(e.count++,e=el.bind(e),n.then(e,e)),a.state.loading|=4,a.instance=u,_t(u);return}u=n.ownerDocument||n,s=F_(s),(l=Mi.get(l))&&Jd(s,l),u=u.createElement("link"),_t(u);var p=u;p._p=new Promise(function(A,B){p.onload=A,p.onerror=B}),Nn(u,"link",s),a.instance=u}e.stylesheets===null&&(e.stylesheets=new Map),e.stylesheets.set(a,n),(n=a.state.preload)&&(a.state.loading&3)===0&&(e.count++,a=el.bind(e),n.addEventListener("load",a),n.addEventListener("error",a))}}var Hc=0;function AM(e,n){return e.stylesheets&&e.count===0&&Vc(e,e.stylesheets),0<e.count||0<e.imgCount?function(a){var s=setTimeout(function(){if(e.stylesheets&&Vc(e,e.stylesheets),e.unsuspend){var u=e.unsuspend;e.unsuspend=null,u()}},6e4+n);0<e.imgBytes&&Hc===0&&(Hc=62500*Xy());var l=setTimeout(function(){if(e.waitingForImages=!1,e.count===0&&(e.stylesheets&&Vc(e,e.stylesheets),e.unsuspend)){var u=e.unsuspend;e.unsuspend=null,u()}},(e.imgBytes>Hc?50:800)+n);return e.unsuspend=a,function(){e.unsuspend=null,clearTimeout(s),clearTimeout(l)}}:null}function q_(e){if(e.count===0&&(e.imgCount===0||!e.waitingForImages)){if(e.stylesheets)Vc(e,e.stylesheets);else if(e.unsuspend){var n=e.unsuspend;e.unsuspend=null,n()}}}function el(){this.count--,q_(this)}function RM(){this.imgCount--,q_(this)}var Gc=null;function Vc(e,n){e.stylesheets=null,e.unsuspend!==null&&(e.count++,Gc=new Map,n.forEach(wM,e),Gc=null,el.call(e))}function wM(e,n){if(!(n.state.loading&4)){var a=Gc.get(e);if(a)var s=a.get(null);else{a=new Map,Gc.set(e,a);for(var l=e.querySelectorAll("link[data-precedence],style[data-precedence]"),u=0;u<l.length;u++){var p=l[u];(p.nodeName==="LINK"||p.getAttribute("media")!=="not all")&&(a.set(p.dataset.precedence,p),s=p)}s&&a.set(null,s)}l=n.instance,p=l.getAttribute("data-precedence"),u=a.get(p)||s,u===s&&a.set(null,l),a.set(p,l),this.count++,s=el.bind(this),l.addEventListener("load",s),l.addEventListener("error",s),u?u.parentNode.insertBefore(l,u.nextSibling):(e=e.nodeType===9?e.head:e,e.insertBefore(l,e.firstChild)),n.state.loading|=4}}var zr={$$typeof:ne,Provider:null,Consumer:null,_currentValue:Vt,_currentValue2:Vt,_threadCount:0};function CM(e,n,a,s,l,u,p,A,B){this.tag=1,this.containerInfo=e,this.pingCache=this.current=this.pendingChildren=null,this.timeoutHandle=-1,this.callbackNode=this.next=this.pendingContext=this.context=this.cancelPendingCommit=null,this.callbackPriority=0,this.expirationTimes=tr(-1),this.entangledLanes=this.shellSuspendCounter=this.errorRecoveryDisabledLanes=this.expiredLanes=this.warmLanes=this.pingedLanes=this.suspendedLanes=this.pendingLanes=0,this.entanglements=tr(0),this.hiddenUpdates=tr(null),this.identifierPrefix=s,this.onUncaughtError=l,this.onCaughtError=u,this.onRecoverableError=p,this.pooledCache=null,this.pooledCacheLanes=0,this.formState=B,this.transitionTypes=null,this.incompleteTransitions=new Map}function Y_(e,n,a,s,l,u,p,A,B,te,fe,ye){return e=new CM(e,n,a,p,B,te,fe,ye,A),n=1,u===!0&&(n|=24),u=Xn(3,null,null,n),e.current=u,u.stateNode=e,n=mf(),n.refCount++,e.pooledCache=n,n.refCount++,u.memoizedState={element:s,isDehydrated:a,cache:n},xf(u),e}function j_(e){return e?(e=ur,e):ur}function Z_(e,n,a,s,l,u){l=j_(l),s.context===null?s.context=l:s.pendingContext=l,s=Ya(n),s.payload={element:a},u=u===void 0?null:u,u!==null&&(s.callback=u),a=ja(e,s,n),a!==null&&(jn(a,e,n),Do(a,e,n))}function K_(e,n){if(e=e.memoizedState,e!==null&&e.dehydrated!==null){var a=e.retryLane;e.retryLane=a!==0&&a<n?a:n}}function th(e,n){K_(e,n),(e=e.alternate)&&K_(e,n)}function Q_(e){if(e.tag===13||e.tag===31){var n=Ts(e,67108864);n!==null&&jn(n,e,67108864),th(e,67108864)}}function J_(e){if(e.tag===13||e.tag===31){var n=ri();n=po(n);var a=Ts(e,n);a!==null&&jn(a,e,n),th(e,n)}}var Fr=!0;function NM(e,n,a,s){var l=xe.T;xe.T=null;var u=we.p;try{we.p=2,nh(e,n,a,s)}finally{we.p=u,xe.T=l}}function DM(e,n,a,s){var l=xe.T;xe.T=null;var u=we.p;try{we.p=8,nh(e,n,a,s)}finally{we.p=u,xe.T=l}}function nh(e,n,a,s){if(Fr){var l=ih(s);if(l===null)Bd(e,n,s,kc,a),ev(e,s);else if(LM(l,e,n,a,s))s.stopPropagation();else if(ev(e,s),n&4&&-1<UM.indexOf(e)){for(;l!==null;){var u=ut(l);if(u!==null)switch(u.tag){case 3:if(u=u.stateNode,u.current.memoizedState.isDehydrated){var p=fa(u.pendingLanes);if(p!==0){var A=u;for(A.pendingLanes|=2,A.entangledLanes|=2;p;){var B=1<<31-ct(p);A.entanglements[1]|=B,p&=~B}Qi(u),(zt&6)===0&&(wc=ke()+500,Yo(0))}}break;case 31:case 13:A=Ts(u,2),A!==null&&jn(A,u,2),Dc(),th(u,2)}if(u=ih(s),u===null&&Bd(e,n,s,kc,a),u===l)break;l=u}l!==null&&s.stopPropagation()}else Bd(e,n,s,null,a)}}function ih(e){return e=Gu(e),ah(e)}var kc=null;function ah(e){if(kc=null,e=st(e),e!==null){var n=f(e);if(n===null)e=null;else{var a=n.tag;if(a===13){if(e=d(n),e!==null)return e;e=null}else if(a===31){if(e=h(n),e!==null)return e;e=null}else if(a===3){if(n.stateNode.current.memoizedState.isDehydrated)return n.tag===3?n.stateNode.containerInfo:null;e=null}else n!==e&&(e=null)}}return kc=e,null}function $_(e){switch(e){case"beforetoggle":case"cancel":case"click":case"close":case"contextmenu":case"copy":case"cut":case"auxclick":case"dblclick":case"dragend":case"dragstart":case"drop":case"focusin":case"focusout":case"input":case"invalid":case"keydown":case"keypress":case"keyup":case"mousedown":case"mouseup":case"paste":case"pause":case"play":case"pointercancel":case"pointerdown":case"pointerup":case"ratechange":case"reset":case"seeked":case"submit":case"toggle":case"touchcancel":case"touchend":case"touchstart":case"volumechange":case"change":case"selectionchange":case"textInput":case"compositionstart":case"compositionend":case"compositionupdate":case"beforeblur":case"afterblur":case"beforeinput":case"blur":case"fullscreenchange":case"fullscreenerror":case"focus":case"hashchange":case"popstate":case"select":case"selectstart":return 2;case"drag":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"mousemove":case"mouseout":case"mouseover":case"pointermove":case"pointerout":case"pointerover":case"resize":case"scroll":case"touchmove":case"wheel":case"mouseenter":case"mouseleave":case"pointerenter":case"pointerleave":return 8;case"message":switch(nt()){case lt:return 2;case Z:return 8;case Ce:case Me:return 32;case Ue:return 268435456;default:return 32}default:return 32}}var sh=!1,rs=null,os=null,ls=null,tl=new Map,nl=new Map,cs=[],UM="mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset".split(" ");function ev(e,n){switch(e){case"focusin":case"focusout":rs=null;break;case"dragenter":case"dragleave":os=null;break;case"mouseover":case"mouseout":ls=null;break;case"pointerover":case"pointerout":tl.delete(n.pointerId);break;case"gotpointercapture":case"lostpointercapture":nl.delete(n.pointerId)}}function il(e,n,a,s,l,u){return e===null||e.nativeEvent!==u?(e={blockedOn:n,domEventName:a,eventSystemFlags:s,nativeEvent:u,targetContainers:[l]},n!==null&&(n=ut(n),n!==null&&Q_(n)),e):(e.eventSystemFlags|=s,n=e.targetContainers,l!==null&&n.indexOf(l)===-1&&n.push(l),e)}function LM(e,n,a,s,l){switch(n){case"focusin":return rs=il(rs,e,n,a,s,l),!0;case"dragenter":return os=il(os,e,n,a,s,l),!0;case"mouseover":return ls=il(ls,e,n,a,s,l),!0;case"pointerover":var u=l.pointerId;return tl.set(u,il(tl.get(u)||null,e,n,a,s,l)),!0;case"gotpointercapture":return u=l.pointerId,nl.set(u,il(nl.get(u)||null,e,n,a,s,l)),!0}return!1}function tv(e){var n=st(e.target);if(n!==null){var a=f(n);if(a!==null){if(n=a.tag,n===13){if(n=d(a),n!==null){e.blockedOn=n,Cl(e.priority,function(){J_(a)});return}}else if(n===31){if(n=h(a),n!==null){e.blockedOn=n,Cl(e.priority,function(){J_(a)});return}}else if(n===3&&a.stateNode.current.memoizedState.isDehydrated){e.blockedOn=a.tag===3?a.stateNode.containerInfo:null;return}}}e.blockedOn=null}function Xc(e){if(e.blockedOn!==null)return!1;for(var n=e.targetContainers;0<n.length;){var a=ih(e.nativeEvent);if(a===null){a=e.nativeEvent;var s=new a.constructor(a.type,a);Hu=s,a.target.dispatchEvent(s),Hu=null}else return n=ut(a),n!==null&&Q_(n),e.blockedOn=a,!1;n.shift()}return!0}function nv(e,n,a){Xc(e)&&a.delete(n)}function OM(){sh=!1,rs!==null&&Xc(rs)&&(rs=null),os!==null&&Xc(os)&&(os=null),ls!==null&&Xc(ls)&&(ls=null),tl.forEach(nv),nl.forEach(nv)}function Wc(e,n){e.blockedOn===n&&(e.blockedOn=null,sh||(sh=!0,o.unstable_scheduleCallback(o.unstable_NormalPriority,OM)))}var qc=null;function iv(e){qc!==e&&(qc=e,o.unstable_scheduleCallback(o.unstable_NormalPriority,function(){qc===e&&(qc=null);for(var n=0;n<e.length;n+=3){var a=e[n],s=e[n+1],l=e[n+2];if(typeof s!="function"){if(ah(s||a)===null)continue;break}var u=ut(a);u!==null&&(e.splice(n,3),n-=3,Hf(u,{pending:!0,data:l,method:a.method,action:s},s,l))}}))}function Hr(e){function n(B){return Wc(B,e)}rs!==null&&Wc(rs,e),os!==null&&Wc(os,e),ls!==null&&Wc(ls,e),tl.forEach(n),nl.forEach(n);for(var a=0;a<cs.length;a++){var s=cs[a];s.blockedOn===e&&(s.blockedOn=null)}for(;0<cs.length&&(a=cs[0],a.blockedOn===null);)tv(a),a.blockedOn===null&&cs.shift();if(a=(e.ownerDocument||e).$$reactFormReplay,a!=null)for(s=0;s<a.length;s+=3){var l=a[s],u=a[s+1],p=l[G]||null;if(typeof u=="function")p||iv(a);else if(p){var A=null;if(u&&u.hasAttribute("formAction")){if(l=u,p=u[G]||null)A=p.formAction;else if(ah(l)!==null)continue}else A=p.action;typeof A=="function"?a[s+1]=A:(a.splice(s,3),s-=3),iv(a)}}}function av(){function e(u){u.canIntercept&&u.info==="react-transition"&&u.intercept({handler:function(){return new Promise(function(p){return l=p})},focusReset:"manual",scroll:"manual"})}function n(){l!==null&&(l(),l=null),s||setTimeout(a,20)}function a(){if(!s&&!navigation.transition){var u=navigation.currentEntry;u&&u.url!=null&&navigation.navigate(u.url,{state:u.getState(),info:"react-transition",history:"replace"})}}if(typeof navigation=="object"){var s=!1,l=null;return navigation.addEventListener("navigate",e),navigation.addEventListener("navigatesuccess",n),navigation.addEventListener("navigateerror",n),setTimeout(a,100),function(){s=!0,navigation.removeEventListener("navigate",e),navigation.removeEventListener("navigatesuccess",n),navigation.removeEventListener("navigateerror",n),l!==null&&(l(),l=null)}}}function rh(e){this._internalRoot=e}Yc.prototype.render=rh.prototype.render=function(e){var n=this._internalRoot;if(n===null)throw Error(r(409));var a=n.current,s=ri();Z_(a,s,e,n,null,null)},Yc.prototype.unmount=rh.prototype.unmount=function(){var e=this._internalRoot;if(e!==null){this._internalRoot=null;var n=e.containerInfo;Z_(e.current,2,null,e,null,null),Dc(),n[de]=null}};function Yc(e){this._internalRoot=e}Yc.prototype.unstable_scheduleHydration=function(e){if(e){var n=wl();e={blockedOn:null,target:e,priority:n};for(var a=0;a<cs.length&&n!==0&&n<cs[a].priority;a++);cs.splice(a,0,e),a===0&&tv(e)}};var sv=t.version;if(sv!=="19.3.0")throw Error(r(527,sv,"19.3.0"));we.findDOMNode=function(e){var n=e._reactInternals;if(n===void 0)throw typeof e.render=="function"?Error(r(188)):(e=Object.keys(e).join(","),Error(r(268,e)));return e=g(n),e=e!==null?x(e):null,e=e===null?null:e.stateNode,e};var PM={bundleType:0,version:"19.3.0",rendererPackageName:"react-dom",currentDispatcherRef:xe,reconcilerVersion:"19.3.0"};if(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__<"u"){var jc=__REACT_DEVTOOLS_GLOBAL_HOOK__;if(!jc.isDisabled&&jc.supportsFiber)try{Je=jc.inject(PM),Ge=jc}catch{}}return sl.createRoot=function(e,n){if(!c(e))throw Error(r(299));var a=!1,s="",l=jg,u=Zg,p=Kg;return n!=null&&(n.unstable_strictMode===!0&&(a=!0),n.identifierPrefix!==void 0&&(s=n.identifierPrefix),n.onUncaughtError!==void 0&&(l=n.onUncaughtError),n.onCaughtError!==void 0&&(u=n.onCaughtError),n.onRecoverableError!==void 0&&(p=n.onRecoverableError)),n=Y_(e,1,!1,null,null,a,s,null,l,u,p,av),e[de]=n.current,Id(e),new rh(n)},sl.hydrateRoot=function(e,n,a){if(!c(e))throw Error(r(299));var s=!1,l="",u=jg,p=Zg,A=Kg,B=null;return a!=null&&(a.unstable_strictMode===!0&&(s=!0),a.identifierPrefix!==void 0&&(l=a.identifierPrefix),a.onUncaughtError!==void 0&&(u=a.onUncaughtError),a.onCaughtError!==void 0&&(p=a.onCaughtError),a.onRecoverableError!==void 0&&(A=a.onRecoverableError),a.formState!==void 0&&(B=a.formState)),n=Y_(e,1,!0,n,a??null,s,l,B,u,p,A,av),n.context=j_(null),a=n.current,s=ri(),s=po(s),l=Ya(s),l.callback=null,ja(a,l,s),a=s,n.current.lanes=a,Fi(n,a),Qi(n),e[de]=n.current,Id(e),new Yc(n)},sl.version="19.3.0",sl}var mv;function WM(){if(mv)return ch.exports;mv=1;function o(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(o)}catch(t){console.error(t)}}return o(),ch.exports=XM(),ch.exports}var qM=WM();const gv=["en","zh","ja"],YM={zh:{meta:{title:"Yoyo — 本地智能体工作站",description:"一台本地的智能体工作站。在你指定的文件夹里改代码、审查改动、拍短剧。写之前先问你。新的干活方式要你亲自换成正在用的那一份。"},nav:{workstation:"工作站",video:"视频",model:"模型",trust:"信任",doors:"入口",download:"下载",menu:"菜单",close:"关闭菜单",themeToDark:"切换到深色",themeToLight:"切换到浅色"},hero:{title:`在你选的文件夹里，
跑一轮智能体。`,dek:"它改代码，审查每一处改动，也能把一章文字拍成短剧。写文件和跑命令之前先问你。新的干活方式先接受评测，再由你换成正在使用的那一份。",download:"下载",github:"GitHub",builds:"全部安装包",chips:["Apache-2.0","Windows · macOS · Linux","桌面 · 浏览器 · 命令行","自备模型"]},stage:{ask:"写文件或跑命令之前，问题出现在这一轮里。可以只放行这一次，记住这个对话，记在这台电脑上，或让这一轮停下。",diff:"每一处改动单独勾选。原来的文件如果已经不在，会直接告诉你失败，不会悄悄把仓库倒回去。",drama:"贴上一章，先确认分成几集，再开拍。创作、短剧和画布是同一套界面，目录和代码仓库分开。",trust:"评测通过，只说明这一份可以换。换上去，是你的动作。密钥和更新程序不在它能碰到的范围里。"},turn:{title:"一轮里，事情按这个顺序发生。",dek:"先点名文件，再决定能不能写。改动留在你看得见的地方。花费到顶，这一轮就停。",steps:[{title:"钉住上下文",body:"用 @ 点名一份文件或一个目录。打一个 @，不会把整个文件夹倒进提示词。"},{title:"先问再写",body:"写文件或跑命令时，问题出现在这一轮里。只放行这一次，记住这个对话，记在这台电脑上，或拒绝。"},{title:"按段留下",body:"右侧是待处理的队列。按片段勾选再应用。没勾的不会写进去。"},{title:"到顶就停",body:"预算不是一条提醒。到了就停。Stop 取消这一轮，已经写好的会留下。"}]},film:{title:`视频是另一场对话。
片子不进代码仓库。`,dek:"创作、短剧、画布共用一套界面。短剧的家在对话里：先把一章拆成几集，确认之后才拍。要看一集，再把导演台从旁边拉开。",modes:[{title:"创作",body:"描述一段片子。图像、语音和视频适配器在设置里，画布用的是同一批。密钥留在这台电脑的保管库，不进对话。"},{title:"短剧",body:"贴上一章，先确认分成几集，再开拍。对话一直留着。打开一集，旁边才分出导演台。短剧里面没有第二块节点画布，也不是一条八步流水线。画幅锁在这一部上。"},{title:"画布",body:"镜头排在同一张空间桌面上。节点是画面。工具停在窗口边缘，画面才看得清。Agent 钉在检查器里，不是另一套聊天。"}],phasesTitle:"导演台有四段。每一段只有一个往下走的动作。",phases:[{title:"剧本",body:"把这一章拆成集。计划没确认，不开拍。"},{title:"造型",body:"角色和场景的样子在这里定下来。"},{title:"分镜",body:"镜头排成一条胶片。板子上能看，也能改。"},{title:"成片",body:"看片和检查在同一张桌子上。胶片条还在。"}],apart:"视频会话和写代码的会话不是同一条。Profile 管不到这里。片子有自己的目录，和代码仓库分开。"},limits:{title:"开始之前，这几件事先说清楚。",dek:"它是这台电脑上的工作站，不是一个免注册的云聊天。",items:["密钥要你自己准备。对话不托管到别人的云上。","不训练专属模型。权重留在你接上的服务里。","技能包默认关闭。你为当前文件夹打开，它才进来。评测不加载它们。","密钥、评测器和更新程序，它改不了。新的干活方式要你亲自换成正在用的那一份。","Windows 和 Linux 安装包未签名。macOS 是 ad-hoc 签名，未经公证。"]},gate:{title:`写下去之前，
它先在对话里问你。`,dek:"写文件或跑命令，问题就出现在这一轮里。你可以只放行这一次，记住整个对话，记在这台电脑上，或让这一轮停下。桌面应用默认是问你，不是自己做主。",once:"一次",session:"本对话",always:"总是",deny:"拒绝",onceHint:"只放行这一次",sessionHint:"这个对话里记住",alwaysHint:"记在这台电脑上",denyHint:"这一轮停下"},model:{title:`模型你自己带。
变的是它怎么干活。`,dek:"Yoyo 不训练一个专属模型，也不把提示词越写越长叫做变聪明。你接上 OpenAI 兼容的接口，权重留在原来的服务里。它能打磨的是提示词、工作手册和技能。打磨结果先停在一旁。你确认之后，才换成正在用的版本。",solver:"你带来的模型",solverBody:"出厂默认是 gpt-4.1-mini，也可以换成任何兼容接口。同一个模型可以准备两套干活方式，互不影响。",shape:"上下文有边界",shapeBody:"用 @ 钉住一份文件或一个目录。打一个 @ 不会把整个仓库塞进提示词。旧的工具输出会被裁短，把位置留给这一轮。",evolve:"先证明，再换上",evolveBody:"你点的赞只进草稿。自动打磨写出的是待晋升版本。老题目和新题目都不能变差，安全检查失败就停。正在使用的版本，只有你能切换。",weightsStill:"权重不动",modelName:"gpt-4.1-mini",lineLive:"错误要返回，不要吞掉。",lineNext:"同时写上调用方。",l1:"L1",l1Body:"提示词、playbook、技能。Evolve 可以改，必须过 Harbor。",l2:"L2",l2Body:"签名的 WASM。未签名进不了 active。没有文件系统。",l3:"L3",l3Body:"循环与策略拓扑。只有你能改。",l4:"L4",l4Body:"内核、评测器、密钥、更新器。智能体不行，evolve 也不行。",fp:"跟着模型走",fpBody:"换了模型，就换一套对应的干活方式。为这个模型调好的手册，不会硬套到另一个模型上。"},see:{title:`改动留在
你看得见的地方。`,dek:"代码按一小段一小段勾选，再决定留下哪些。短剧如果要拍，走自己的目录，不和写代码的工作区混在一起。",pin:"钉一份文件",pinBody:"工作从一份你点名的文件开始，而不是把整个文件夹倒进对话。",diff:"勾选再留下",diffBody:"每一处改动单独决定。原来的文件如果已经不在，会直接告诉你失败，不会悄悄把仓库倒回去。",drama:"短剧单独拍",dramaBody:"贴上一章，先确认分成几集，再开拍。创作、短剧和画布是同一套界面，目录和代码仓库分开。",folderAlt:"深色橡木桌上，一本合上的牛皮纸文件夹和一支铜笔。"},trust:{title:`它可以打磨自己。
不能自己上岗。`,dek:"草稿、待晋升、正在用，是三份不同的干活方式。评测通过，只说明这一份可以换。换上去，是你的动作。密钥和更新程序不在它能碰到的范围里。",staging:"草稿",stagingHint:"点赞只写到这里",canary:"待晋升",canaryHint:"评测过了，还没轮到你",active:"正在用",activeHint:"只有你能拨这根指针",checkout:"晋升",done:"指针是你拨的。不是它。"},doors:{title:`同一套能力。
三扇门都能进。`,dek:"桌面有窗口和托盘。浏览器开在你自己的电脑上。命令行走的是同一份本地数据。页面里没有第二套智能体。",desktop:"桌面",desktopBody:"原生窗口、菜单、托盘和通知。安装包就是这个应用。",browser:"浏览器",browserBody:"对话和评测都在这台电脑上。没有托盘，也没有云端托管。",cli:"命令行",cliBody:"跑一轮、做评测、打磨干活方式，或安装一个技能包。",skills:"技能默认关闭",skillsBody:"技能包先放在这台电脑上，不会自动参加每一轮。你为当前文件夹打开一个，它才进来。评测时不读它们。",budget:"花费到顶就停",budgetBody:"上限到了，这一轮就停，不是弹一条提醒。密钥留在本地。它改不了这些设置。"},get:{title:`装到这台电脑上，
就可以开始。`,dek:"不用注册，也不把对话托管到别人的云上。选一个文件夹，填入你自己的接口密钥，然后让它先问再写。",premise:"密钥要你自己准备。如果想要一个不用密钥、完全托管的聊天产品，这不是那一页。",version:"版本",steps:["按系统安装桌面应用。Windows 不用管理员。","设置里选工作文件夹。它只在这个文件夹里动手。","设置里粘贴 OpenAI 兼容的 Key。","输入任务。用 @ 钉上下文。写入先问你。"],thisMachine:"这台电脑",note:"Windows / Linux 安装包未签名。SmartScreen 可能拦一下：更多信息 → 仍要运行。macOS 是 ad-hoc 签名，未经公证。源码 Apache-2.0，不舒服就自己编译。"},pitch:{title:"它实际会做的事",dek:"不是云上的另一个你。是这台电脑上的一轮工作：你给文件夹，你给模型，你决定什么能写进去。",cards:[{title:"改你文件夹里的代码",body:"选一个目录，输入任务。它边读边改。只要方案、先不落笔，就打开 Plan。停掉这一轮时，已经写好的会留下。"},{title:"先问，再写，再执行",body:"写文件或跑命令时，问题出现在对话里。只放行这一次，记住这次对话，记在这台电脑上，或者拒绝。"},{title:"每一处改动都看得见",body:"右侧是待处理的队列。按片段勾选再应用。原来的文件如果不在了，会直接失败，不会悄悄把仓库倒回去。"},{title:"短剧不和代码混放",body:"贴上一章，先确认分成几集，再开拍。创作、短剧、画布是同一套界面。片子有自己的目录。"},{title:"技能先关着",body:"别人写的技能包不会自动进入每一轮。你在这个工作区打开，它才进来。评测的时候一律不加载。"},{title:"钱花到上限就停",body:"预算不是一条提醒。到了就停。密钥留在这台电脑上。它改不了自己的设置，也换不了正在运行的程序。"}]},foot:{product:"产品",invariants:"不变量",issues:"Issues",license:"Apache-2.0",stack:"Go · Wails"},demo:{agent:"Agent",review:"审查",video:"视频",harbor:"Harbor",module:"模块",chats:"会话",threadLive:"Parser fix",threadLiveSub:"apply_patch · 等你",threadIdle:"Harbor smoke",newChat:"新会话",hello:"下午好。",bubble:"把 parser 里那处错误返回补上",read:"已读取",patch:"apply_patch · +24 −6",needsOk:"等你点头",needsApproval:"需要审批",reading:"读取",patching:"应用补丁",worked:"工作了 4s",plan:"Plan",action:"write_file",path:"internal/runtime/loop.go",allowOnce:"允许一次",allowSession:"本对话",allowAlways:"总是允许",reject:"拒绝",apply:"应用",hunks:"处改动",composer:"修 loop.go 里的 parser",createTab:"创作",canvasTab:"画布",browserBar:"localhost · preview",browserBody:"解析器现在会把错误返回去。",trailRead:"Read",trailPatch:"apply_patch",queue:"队列",queueItem:"loop.go · +24 −6",statusFolder:"~/work/parser",local:"本地",kept:"已留下",keptNote:"勾上的留在工作区里。没勾的不会写进去。",createBody:"描述一段片子。图像、语音和视频适配器是你自己的。",canvasHint:"镜头排在同一张桌面上。目录和代码仓库分开。",dramaPrompt:"把这一章拍成三集。先确认分集，再拍。",ep1:"抵达",ep2:"灯",ep3:"离开",shotCap:"02 · 灯",createHint:"橡木桌上的一盏灯，黄昏。"}},en:{meta:{title:"Yoyo — a local agent workstation",description:"A local workstation. You bring the model; the weights stay put. Writes ask first. A new way of working is tested first. You switch it on."},nav:{workstation:"Workstation",video:"Video",model:"Model",trust:"Trust",doors:"Doors",download:"Download",menu:"Menu",close:"Close menu",themeToDark:"Switch to dark",themeToLight:"Switch to light"},hero:{title:`Run an agent
in a folder you choose.`,dek:"It edits code, lets you keep each change, and can turn a chapter into a short film. It asks before it writes or runs a command. A new way of working is tested first. You are the one who switches it on.",download:"Download",github:"GitHub",builds:"All builds",chips:["Apache-2.0","Windows · macOS · Linux","Desktop · browser · CLI","Bring your own model"]},stage:{ask:"A write or a command asks in the turn. Allow it once, for this chat, on this machine, or stop the turn.",diff:"Keep each change on its own. If the file is already gone, restore fails out loud. Nothing is quietly rolled back.",drama:"Paste a chapter, confirm the episodes, then shoot. Create, drama, and canvas share one shell. The footage has its own folder.",trust:"A passing eval only means this one can be switched on. Switching it on is your action. Keys and the updater stay out of reach."},turn:{title:"A turn happens in this order.",dek:"Name a file first. Then decide what may be written. Changes stay where you can see them. When the budget is spent, the turn stops.",steps:[{title:"Pin context",body:"Type @ to name one file or one folder. One @ does not pour the whole tree into the prompt."},{title:"Ask, then write",body:"A write or a command asks in the turn. Allow it once, for this chat, on this machine, or turn it down."},{title:"Keep by hunk",body:"The queue sits beside the thread. Tick the pieces you want. What you leave unticked is not written."},{title:"Stop at the cap",body:"The budget is not a warning. When it hits, the turn stops. Stop cancels the turn. What already landed stays."}]},film:{title:`Video is another conversation.
Footage stays out of the repo.`,dek:"Create, drama, and canvas share one shell. Drama lives in the chat: split a chapter into episodes, confirm the plan, then shoot. Open an episode, and the director desk splits out beside it.",modes:[{title:"Create",body:"Describe a piece. Image, voice, and video adapters live in Settings. Canvas uses the same ones. Keys stay in the vault on this machine."},{title:"Drama",body:"Paste a chapter, confirm the episodes, then shoot. The chat stays mounted. Opening an episode splits a director desk beside it. There is no second node canvas inside drama, and it is not an eight-stage factory. Aspect is locked on the series."},{title:"Canvas",body:"Shots sit on one spatial desk. A node is a picture. Tools stay at the window edge so the frame stays clear. The agent pins to the inspector. It is not a second chat."}],phasesTitle:"The director desk has four phases. Each one has a single next step.",phases:[{title:"Script",body:"Split the chapter into episodes. Nothing shoots before the plan is confirmed."},{title:"Looks",body:"Characters and places take their shape here."},{title:"Board",body:"Shots line up as a film strip. You can watch the board and change it."},{title:"Cut",body:"The viewer and the inspector share the desk. The film strip stays."}],apart:"Video chats are not Agent chats. A profile does not apply on this channel. Footage has its own folder, apart from the code workspace."},limits:{title:"Before you start, five things are already true.",dek:"This is a workstation on your machine. It is not a hosted chat you can use without a key.",items:["You bring the key. The conversation is not hosted in someone else's cloud.","Yoyo does not train a model of its own. The weights stay with the service you connect.","Skill packs start off. You enable one for this folder. Evaluation never loads them.","It cannot change keys, the evaluator, or the updater. You switch on a new way of working yourself.","Windows and Linux packages are unsigned. macOS is ad-hoc signed, not notarized."]},gate:{title:"The question lands in the turn.",dek:"A write or a command asks in the thread: allow once, this chat, always, or reject. The desktop default is to ask.",once:"Once",session:"This chat",always:"Always",deny:"Deny",onceHint:"This write only",sessionHint:"Remembered in this chat",alwaysHint:"Kept on this machine",denyHint:"The turn stops"},model:{title:`You bring the model.
What changes is how it works.`,dek:"Yoyo does not train a model of its own, and a longer prompt is not getting smarter. You connect an OpenAI-compatible API. The weights stay with that service. It may refine the prompt, the working notes, and skills. The result waits aside. You switch it on.",solver:"Solver",solverBody:"In a turn the model reads context, calls tools, and writes prose. Stop cancels the turn. What already landed stays. Plan only proposes. Writes and shell stay off.",shape:"Shape",shapeBody:"Old tool results get shortened. Context has a budget. @file pins one file. It does not dump the tree into the prompt. That shape is the harness, not a personality.",evolve:"Evolve",evolveBody:"Thumbs write staging. Evolve writes canary. Held-in and held-out must not drop. A safety failure blocks. You check out, and only then the pointer is active.",weightsStill:"Weights stay put",modelName:"gpt-4.1-mini",lineLive:"Return the error. Do not swallow it.",lineNext:"Name the caller as well.",l1:"L1",l1Body:"Prompt, playbook, skills. Evolve may change them. Harbor must pass.",l2:"L2",l2Body:"Signed WASM. Unsigned never reaches active. No filesystem.",l3:"L3",l3Body:"Loop and policy. Only you change these.",l4:"L4",l4Body:"Kernel, evaluator, keys, updater. Not the agent. Not evolve.",fp:"Model fingerprint",fpBody:"refs/models/<fingerprint>/active. Same weights. The harness is per model."},see:{title:"After you ask, the change stays where you can see it.",dek:"Pin a file. Tick the hunks you want. Video, if you shoot it, lives in its own folder.",pin:"Pin a file",pinBody:"Typing @ does not dump the tree into the prompt.",diff:"Tick, then keep",diffBody:"Accept by hunk. Restore fails out loud if the files are gone.",drama:"Video is a conversation",dramaBody:"Paste a chapter. Confirm the episodes, then shoot. Not mixed with the code workspace.",folderAlt:"A closed kraft folder and a brass pen on a dark oak desk."},trust:{title:"It can get smarter. It cannot promote itself.",dek:"Thumbs write a draft. After eval passes, you check it out. The pointer moves a harness, not the model weights. Keys and the updater stay off limits.",staging:"Draft",stagingHint:"Thumbs write here only",canary:"Waiting",canaryHint:"Eval passed. You have not checked it out.",active:"Live",activeHint:"Only you move this pointer",checkout:"Promote",done:"You moved it. The agent did not."},doors:{title:"One loop. Three doors.",dek:"Desktop has a window and a tray. The browser runs on localhost. The CLI shares the same home.",desktop:"Desktop",desktopBody:"Native window, menus, tray, notifications. The installer is this app.",browser:"Browser",browserBody:"Chat and eval on localhost. No tray.",cli:"CLI",cliBody:"Run a turn, eval, evolve, or install a pack.",skills:"Skills stay off",skillsBody:"Packs live in ~/.yoyo/packs/. You enable one for the workspace. Eval never loads them.",budget:"Budget is a hard stop",budgetBody:"When the cap hits, the loop stops. Keys stay in the local vault."},get:{title:"No account. No cloud. Install and run.",dek:"Pick a folder. Paste an OpenAI-compatible key. Type a task, pin context with @, approve writes.",premise:"You need your own key. If you want a hosted chat with no keys, this page is not for that.",version:"Version",steps:["Install the desktop app for your OS.","Settings: one folder the agent may touch.","Settings: paste an OpenAI-compatible key.","Type a task. Pin context with @. Writes ask first."],thisMachine:"This machine",note:"Windows and Linux packages are unsigned. SmartScreen may ask for More info, then Run anyway. macOS is ad-hoc signed, not notarized. Source is Apache-2.0."},pitch:{title:"What it actually does",dek:"Not another you in the cloud. A turn on this machine: you give the folder, you give the model, you decide what may be written.",cards:[{title:"Edit the folder you chose",body:"Pick a directory and type a task. It reads and edits as it goes. Plan proposes only. Stop keeps what already landed."},{title:"Ask, then write, then run",body:"A write or a command asks in the thread. Allow it once, for this chat, on this machine, or turn it down."},{title:"Every change stays visible",body:"The queue sits beside the thread. Keep changes piece by piece. If the file is gone, restore fails out loud."},{title:"Film stays out of the repo",body:"Paste a chapter, confirm the episodes, then shoot. Create, drama, and canvas share one shell. The footage has its own folder."},{title:"Skills start off",body:"Packs do not enter every turn. You enable one for this folder. Evaluation never loads them."},{title:"The budget is a stop",body:"When the cap hits, the turn stops. Keys stay on this machine. It cannot edit those settings or replace the running app."}]},foot:{product:"Product",invariants:"Invariants",issues:"Issues",license:"Apache-2.0",stack:"Go · Wails"},demo:{agent:"Agent",review:"Review",video:"Video",harbor:"Harbor",module:"Module",chats:"Chats",threadLive:"Parser fix",threadLiveSub:"apply_patch · waiting",threadIdle:"Harbor smoke",newChat:"New chat",hello:"Good afternoon.",bubble:"Return the parse error instead of swallowing it",read:"Read",patch:"apply_patch · +24 −6",needsOk:"Needs your OK",needsApproval:"Needs approval",reading:"Reading",patching:"Patching",worked:"Worked for 4s",plan:"Plan",action:"write_file",path:"internal/runtime/loop.go",allowOnce:"Allow once",allowSession:"This chat",allowAlways:"Always allow",reject:"Reject",apply:"Apply",hunks:"hunks",composer:"Fix the parser in @file:loop.go",createTab:"Create",canvasTab:"Canvas",browserBar:"localhost · preview",browserBody:"The parser now returns the error.",trailRead:"Read",trailPatch:"apply_patch",queue:"Queue",queueItem:"loop.go · +24 −6",statusFolder:"~/work/parser",local:"Local",kept:"Kept",keptNote:"Ticked hunks stay in the folder. The rest are not written.",createBody:"Describe a piece. Image, voice, and video adapters are yours.",canvasHint:"Shots sit on one desk. The folder is not the code workspace.",dramaPrompt:"Shoot this chapter as three episodes. Confirm the plan, then shoot.",ep1:"Arrival",ep2:"The lamp",ep3:"Leaving",shotCap:"02 · The lamp",createHint:"A lamp on an oak desk, dusk."}},ja:{meta:{title:"Yoyo — ローカルの作業台",description:"ローカルの作業台。モデルは自分で持つ。重みは動かない。書き込みの前に尋ねる。証明したあと、切り替えるのはあなた。"},nav:{workstation:"作業台",video:"ビデオ",model:"モデル",trust:"信頼",doors:"入口",download:"入手",menu:"メニュー",close:"メニューを閉じる",themeToDark:"ダークに切り替え",themeToLight:"ライトに切り替え"},hero:{title:`選んだフォルダで、
エージェントを走らせる。`,dek:"コードを直し、変更は断片ごとに残し、章を短編にもできる。書く前、コマンドの前に尋ねる。新しい働き方はまず評価され、切り替えるのはあなた。",download:"入手",github:"GitHub",builds:"すべてのビルド",chips:["Apache-2.0","Windows · macOS · Linux","デスクトップ · ブラウザ · CLI","モデルは自分で"]},stage:{ask:"書き込みもコマンドも、このターンの中で尋ねる。一度、この会話、このマシン、またはここで止める。",diff:"変更は断片ごとに残す。ファイルがもう無いときは、静かに戻さず、失敗してはっきり言う。",drama:"章を貼り、話数を確認してから撮る。作成、ドラマ、キャンバスは一つの殻。映像はコードの作業場と分ける。",trust:"評価が通ったのは、この一份が切り替えられるという意味だけ。切り替えるのはあなた。鍵とアップデータには触れない。"},turn:{title:"一つのターンは、この順で進む。",dek:"まずファイルを名指しする。それから書いてよいかを決める。変更は見える場所に残る。予算の上限で、ターンは止まる。",steps:[{title:"文脈をピンする",body:"@ で一つのファイルかフォルダを名指しする。@ を一つ打っても、木全体はプロンプトに入らない。"},{title:"聞いてから書く",body:"書き込みもコマンドも、このターンで尋ねる。一度、この会話、このマシン、または拒否。"},{title:"断片ごとに残す",body:"キューは会話のそば。残す断片だけを選ぶ。選ばなかったものは書き込まれない。"},{title:"上限で止まる",body:"予算は注意書きではない。達したら止まる。Stop はそのターンを止める。すでに書いたものは残る。"}]},film:{title:`ビデオは別の会話。
映像はリポジトリの外。`,dek:"作成、ドラマ、キャンバスは一つの殻。ドラマの家は会話の中。章を話に分け、確認してから撮る。一話を開くと、隣に演出机が分かれる。",modes:[{title:"作成",body:"一本を説明する。画像、音声、動画のアダプタは設定にあり、キャンバスと同じものを使う。鍵はこのマシンの保管庫に残る。"},{title:"ドラマ",body:"章を貼り、話数を確認してから撮る。会話はそのまま残る。一話を開くと、隣に演出机が分かれる。ドラマの中に二つ目のノードキャンバスはなく、八段階の工場でもない。画角はそのシリーズにロックされる。"},{title:"キャンバス",body:"ショットは一つの空間の机に並ぶ。ノードは絵だ。道具は窓の縁に置き、画面を空けておく。Agent はインスペクタにピンされ、別のチャットではない。"}],phasesTitle:"演出机は四つの段階。各段階の次の動作は一つ。",phases:[{title:"脚本",body:"章を話に分ける。計画を確認するまで撮らない。"},{title:"造型",body:"人物と場所の見た目をここで決める。"},{title:"絵コンテ",body:"ショットはフィルムの帯になる。板の上で見て、直せる。"},{title:"完成",body:"試写と検査は同じ机。フィルムの帯は残る。"}],apart:"ビデオの会話は Agent の会話ではない。プロフィールはこの通道に届かない。映像はコードの作業場とは別のフォルダ。"},limits:{title:"始める前に、決まっていること。",dek:"これはこのマシンの作業台です。鍵なしで使えるホストされたチャットではありません。",items:["鍵は自分で持つ。会話は他人のクラウドに預けない。","自前のモデルは訓練しない。重みは接続したサービスに残る。","スキルパックは最初オフ。このフォルダで有効化してから入る。評価は読み込まない。","鍵、評価器、アップデータは変更できない。新しい働き方を切り替えるのはあなた。","Windows と Linux は未署名。macOS はアドホック署名で、公証はない。"]},gate:{title:"問いはターンの中に出る。",dek:"書き込みもコマンドも、スレッドで尋ねる。一度許可、この会話、常に許可、または拒否。デスクトップの既定は尋ねること。",once:"一度",session:"この会話",always:"常に",deny:"拒否",onceHint:"この書き込みだけ",sessionHint:"この会話で覚える",alwaysHint:"このマシンに記憶",denyHint:"このターンは止まる"},model:{title:`モデルは自分で持つ。
変わるのは働き方。`,dek:"Yoyo は自前モデルを訓練しない。プロンプトを長くしても、賢くなったことにはならない。OpenAI 互換の API につなぐ。重みは元のサービスに残る。磨けるのはプロンプト、手順、スキル。結果は脇に置かれ、切り替えるのはあなた。",solver:"ソルバー",solverBody:"ターンの中で、モデルは文脈を読み、ツールを呼び、文章を書く。Stop はそのターンを止める。すでに書いたものは残る。Plan は提案だけ。書き込みと shell は閉じたまま。",shape:"形",shapeBody:"古いツール結果は短く切られる。文脈には予算がある。@file は一つのファイルをピンする。木全体はプロンプトに入らない。これはハーネスの形で、個性ではない。",evolve:"進化",evolveBody:"評価は staging へ。Evolve は canary へ。保持セットもホールドアウトも下がってはならない。安全失敗は常に止める。checkout したあとだけ、ポインタは active。",weightsStill:"重みは動かない",modelName:"gpt-4.1-mini",lineLive:"エラーは返す。握りつぶさない。",lineNext:"呼び出し元の名前も書く。",l1:"L1",l1Body:"プロンプト、playbook、スキル。Evolve は変えられる。Harbor を通す。",l2:"L2",l2Body:"署名済み WASM。未署名は active に入らない。ファイルシステムはない。",l3:"L3",l3Body:"ループと方針。変えられるのは人だけ。",l4:"L4",l4Body:"カーネル、評価器、鍵、アップデータ。エージェントも evolve も不可。",fp:"モデル指紋",fpBody:"refs/models/<指紋>/active。同じ重みでも、ハーネスはモデルごと。"},see:{title:"頼んだあと、変更は見える場所に残る。",dek:"ファイルをピンする。残すハンクを選ぶ。動画は専用フォルダ。",pin:"ファイルをピン",pinBody:"@ を打っただけでリポジトリ全体は入らない。",diff:"選んで残す",diffBody:"ハンク単位で適用。見つからなければ失敗する。",drama:"動画は会話",dramaBody:"章を貼る。話数を確認してから撮る。コードの作業場とは混ぜない。",folderAlt:"暗いオークの机の上の、閉じたクラフト紙のフォルダと真鍮のペン。"},trust:{title:"賢くなってよい。自分では昇格できない。",dek:"評価は下書きへ。評価が通ったら、あなたが checkout する。動くのはハーネスで、モデルの重みではない。鍵とアップデータには触れない。",staging:"下書き",stagingHint:"評価はここだけ",canary:"待機",canaryHint:"評価は通った。まだ checkout していない",active:"使用中",activeHint:"動かすのは人だけ",checkout:"昇格",done:"動かしたのはあなた。エージェントではない。"},doors:{title:"ひとつのループ。三つの入口。",dek:"デスクトップは窓とトレイ。ブラウザは本機。CLI は同じ home。",desktop:"デスクトップ",desktopBody:"ネイティブ窓、メニュー、トレイ、通知。インストーラはこれ。",browser:"ブラウザ",browserBody:"会話と評価は本機。トレイはない。",cli:"CLI",cliBody:"ターン、評価、進化、パック。",skills:"スキルは既定オフ",skillsBody:"パックは ~/.yoyo/packs/。ワークスペースで有効化。評価は読まない。",budget:"予算はハードストップ",budgetBody:"上限に達したら止まる。鍵はローカル保管庫。"},get:{title:"アカウントもクラウドもない。入れて走らせる。",dek:"フォルダを選ぶ。OpenAI 互換キーを貼る。タスクを打ち、@ でピンし、書き込みは承認。",premise:"キーは自分で持つ。ホストされたチャットが欲しいなら、このページは対象外。",version:"バージョン",steps:["OS のデスクトップアプリを入れる。","設定で作業フォルダを選ぶ。そこだけ触る。","設定に OpenAI 互換キーを貼る。","タスクを打つ。@ でピン。書き込みは承認。"],thisMachine:"このマシン",note:"Windows と Linux は未署名。SmartScreen は「詳細情報 → 実行」。macOS はアドホック署名で公証なし。ソースは Apache-2.0。"},pitch:{title:"実際にすること",dek:"クラウド上のもう一人ではない。このマシンでの一つのターン。フォルダもモデルも、書いてよいかも、決めるのはあなた。",cards:[{title:"選んだフォルダを直す",body:"ディレクトリを選び、タスクを打つ。読みながら直す。Plan は提案だけ。止めても、すでに書いたものは残る。"},{title:"聞いてから書き、実行する",body:"書き込みもコマンドも、スレッドで尋ねる。一度、この会話、このマシン、または拒否。"},{title:"変更は見えるところに残る",body:"キューは会話のそば。断片ごとに残す。ファイルがなければ、復元は失敗してはっきり言う。"},{title:"映像はリポジトリの外",body:"章を貼り、話数を確認してから撮る。作成、ドラマ、キャンバスは一つの殻。映像は別フォルダ。"},{title:"スキルは最初オフ",body:"パックは毎ターン入らない。このフォルダで有効化してから。評価は読まない。"},{title:"予算は停止",body:"上限に達したらターンは止まる。鍵はこのマシン。設定も実行中のアプリも、自分では替えられない。"}]},foot:{product:"製品",invariants:"不変条件",issues:"Issues",license:"Apache-2.0",stack:"Go · Wails"},demo:{agent:"Agent",review:"審査",video:"ビデオ",harbor:"Harbor",module:"モジュール",chats:"チャット",threadLive:"Parser fix",threadLiveSub:"apply_patch · 待ち",threadIdle:"Harbor smoke",newChat:"新しい会話",hello:"こんにちは。",bubble:"parser のエラーを握りつぶさず返す",read:"読んだ",patch:"apply_patch · +24 −6",needsOk:"確認が必要",needsApproval:"承認が必要",reading:"読み取り",patching:"パッチ",worked:"4s 作業した",plan:"Plan",action:"write_file",path:"internal/runtime/loop.go",allowOnce:"一度許可",allowSession:"この会話",allowAlways:"常に許可",reject:"拒否",apply:"適用",hunks:"ハンク",composer:"@file:loop.go の parser を直す",createTab:"作成",canvasTab:"キャンバス",browserBar:"localhost · preview",browserBody:"パーサーはエラーを返すようになった。",trailRead:"Read",trailPatch:"apply_patch",queue:"キュー",queueItem:"loop.go · +24 −6",statusFolder:"~/work/parser",local:"ローカル",kept:"残した",keptNote:"選んだハンクはフォルダに残る。選ばなかったものは書き込まれない。",createBody:"一本を説明する。画像、音声、動画のアダプタは自分のもの。",canvasHint:"ショットは一つの机に並ぶ。フォルダはコードの作業場とは別。",dramaPrompt:"この章を三話で撮る。計画を確認してから撮る。",ep1:"到着",ep2:"ランプ",ep3:"出発",shotCap:"02 · ランプ",createHint:"樫の机のランプ、黄昏。"}}};function jM(){try{const t=new URLSearchParams(window.location.search).get("lang");if(t&&gv.includes(t))return t;const i=localStorage.getItem("yoyo-site-lang");if(i&&gv.includes(i))return i}catch{}const o=(typeof navigator<"u"?navigator.language:"en").toLowerCase();return o.startsWith("zh")?"zh":o.startsWith("ja")?"ja":"en"}function ZM(o){return o==="zh"?"zh-Hans":o}const Fp="Shenchangxin/yoyo",hh=`https://github.com/${Fp}/releases/latest`,_l="0.4.2";function rl(o,t){return`https://github.com/${Fp}/releases/download/v${o}/${t}`}function KM(o){return o?o<1024*1024?`${Math.round(o/1024)} KB`:`${(o/(1024*1024)).toFixed(0)} MB`:""}function jh(o){return[{os:"windows",label:"Windows",name:`Yoyo-${o}-windows-x64-setup.exe`,href:rl(o,`Yoyo-${o}-windows-x64-setup.exe`),size:""},{os:"mac",label:"macOS",name:`Yoyo-${o}-darwin-universal.dmg`,href:rl(o,`Yoyo-${o}-darwin-universal.dmg`),size:""},{os:"linux",label:"Linux",name:`Yoyo-${o}-linux-amd64.AppImage`,href:rl(o,`Yoyo-${o}-linux-amd64.AppImage`),size:""},{os:"deb",label:"Debian",name:`Yoyo-${o}-linux-amd64.deb`,href:rl(o,`Yoyo-${o}-linux-amd64.deb`),size:""},{os:"rpm",label:"RPM",name:`Yoyo-${o}-linux-amd64.rpm`,href:rl(o,`Yoyo-${o}-linux-amd64.rpm`),size:""}]}function QM(){const o=navigator.userAgent;return/Mac OS X|Macintosh/.test(o)?"mac":/Linux/.test(o)&&!/Android/.test(o)?"linux":"windows"}function ol(o,t){for(const i of t){const r=o.find(c=>i.test(c.name));if(r)return r}return null}async function JM(){const o=jh(_l);try{const t=await fetch(`https://api.github.com/repos/${Fp}/releases/latest`);if(!t.ok)throw new Error("release");const i=await t.json(),r=String(i.tag_name||_l).replace(/^v/,""),c=i.assets||[],f=jh(r),d=ol(c,[/windows-x64-setup\.exe$/i]),h=ol(c,[/darwin-universal\.dmg$/i,/\.dmg$/i]),m=ol(c,[/linux-amd64\.AppImage$/i,/\.AppImage$/i]),g=ol(c,[/\.deb$/i]),x=ol(c,[/\.rpm$/i]),_=[d,h,m,g,x];return{version:r,assets:f.map((v,E)=>{const R=_[E];return R?{...v,name:R.name,href:R.browser_download_url,size:KM(R.size)}:v})}}catch{return{version:_l,assets:o}}}/**
 * @license
 * Copyright 2010-2026 Three.js Authors
 * SPDX-License-Identifier: MIT
 */const Hp="186",$M=0,_v=1,eE=2,vu=1,tE=2,ml=3,Ks=0,Kn=1,na=2,La=0,vl=1,vv=2,xv=3,Sv=4,nE=5,eo=100,iE=101,aE=102,sE=103,rE=104,oE=200,lE=201,cE=202,uE=203,xx=204,Sx=205,fE=206,dE=207,hE=208,pE=209,mE=210,gE=211,_E=212,vE=213,xE=214,Zh=0,Kh=1,Qh=2,xl=3,Jh=4,$h=5,ep=6,tp=7,yx=0,SE=1,yE=2,sa=0,Mx=1,Ex=2,bx=3,Tx=4,Ax=5,Rx=6,wx=7,Cx=300,Qs=301,ao=302,ph=303,mh=304,Du=306,np=1e3,Ua=1001,ip=1002,Un=1003,ME=1004,Zc=1005,In=1006,gh=1007,js=1008,fi=1009,Nx=1010,Dx=1011,Sl=1012,Gp=1013,ra=1014,ia=1015,oa=1016,Vp=1017,kp=1018,yl=1020,Ux=35902,Lx=35899,Ox=1021,Px=1022,Ii=1023,Ia=1026,Zs=1027,Ix=1028,Xp=1029,Js=1030,Wp=1031,qp=1033,xu=33776,Su=33777,yu=33778,Mu=33779,ap=35840,sp=35841,rp=35842,op=35843,lp=36196,cp=37492,up=37496,fp=37488,dp=37489,bu=37490,hp=37491,pp=37808,mp=37809,gp=37810,_p=37811,vp=37812,xp=37813,Sp=37814,yp=37815,Mp=37816,Ep=37817,bp=37818,Tp=37819,Ap=37820,Rp=37821,wp=36492,Cp=36494,Np=36495,Dp=36283,Up=36284,Tu=36285,Lp=36286,EE=3200,Op=0,bE=1,vs="",bi="srgb",Au="srgb-linear",Ru="linear",Yt="srgb",_h=7680,TE=519,AE=512,RE=513,wE=514,Yp=515,CE=516,NE=517,jp=518,DE=519,UE=35044,yv="300 es",aa=2e3,Ml=2001;function LE(o){for(let t=o.length-1;t>=0;--t)if(o[t]>=65535)return!0;return!1}function wu(o){return document.createElementNS("http://www.w3.org/1999/xhtml",o)}function OE(){const o=wu("canvas");return o.style.display="block",o}const Mv={};function Ev(...o){const t="THREE."+o.shift();console.log(t,...o)}function Bx(o){const t=o[0];if(typeof t=="string"&&t.startsWith("TSL:")){const i=o[1];i&&i.isStackTrace?o[0]+=" "+i.getLocation():o[1]='Stack trace not available. Enable "THREE.Node.captureStackTrace" to capture stack traces.'}return o}function ot(...o){o=Bx(o);const t="THREE."+o.shift();{const i=o[0];i&&i.isStackTrace?console.warn(i.getError(t)):console.warn(t,...o)}}function It(...o){o=Bx(o);const t="THREE."+o.shift();{const i=o[0];i&&i.isStackTrace?console.error(i.getError(t)):console.error(t,...o)}}function no(...o){const t=o.join(" ");t in Mv||(Mv[t]=!0,ot(...o))}function PE(o,t,i){return new Promise(function(r,c){function f(){switch(o.clientWaitSync(t,o.SYNC_FLUSH_COMMANDS_BIT,0)){case o.WAIT_FAILED:c();break;case o.TIMEOUT_EXPIRED:setTimeout(f,i);break;default:r()}}setTimeout(f,i)})}const IE={[Zh]:Kh,[Qh]:ep,[Jh]:tp,[xl]:$h,[Kh]:Zh,[ep]:Qh,[tp]:Jh,[$h]:xl};class $s{addEventListener(t,i){this._listeners===void 0&&(this._listeners={});const r=this._listeners;r[t]===void 0&&(r[t]=[]),r[t].indexOf(i)===-1&&r[t].push(i)}hasEventListener(t,i){const r=this._listeners;return r===void 0?!1:r[t]!==void 0&&r[t].indexOf(i)!==-1}removeEventListener(t,i){const r=this._listeners;if(r===void 0)return;const c=r[t];if(c!==void 0){const f=c.indexOf(i);f!==-1&&c.splice(f,1)}}dispatchEvent(t){const i=this._listeners;if(i===void 0)return;const r=i[t.type];if(r!==void 0){t.target=this;const c=r.slice(0);for(let f=0,d=c.length;f<d;f++)c[f].call(this,t);t.target=null}}}const On=["00","01","02","03","04","05","06","07","08","09","0a","0b","0c","0d","0e","0f","10","11","12","13","14","15","16","17","18","19","1a","1b","1c","1d","1e","1f","20","21","22","23","24","25","26","27","28","29","2a","2b","2c","2d","2e","2f","30","31","32","33","34","35","36","37","38","39","3a","3b","3c","3d","3e","3f","40","41","42","43","44","45","46","47","48","49","4a","4b","4c","4d","4e","4f","50","51","52","53","54","55","56","57","58","59","5a","5b","5c","5d","5e","5f","60","61","62","63","64","65","66","67","68","69","6a","6b","6c","6d","6e","6f","70","71","72","73","74","75","76","77","78","79","7a","7b","7c","7d","7e","7f","80","81","82","83","84","85","86","87","88","89","8a","8b","8c","8d","8e","8f","90","91","92","93","94","95","96","97","98","99","9a","9b","9c","9d","9e","9f","a0","a1","a2","a3","a4","a5","a6","a7","a8","a9","aa","ab","ac","ad","ae","af","b0","b1","b2","b3","b4","b5","b6","b7","b8","b9","ba","bb","bc","bd","be","bf","c0","c1","c2","c3","c4","c5","c6","c7","c8","c9","ca","cb","cc","cd","ce","cf","d0","d1","d2","d3","d4","d5","d6","d7","d8","d9","da","db","dc","dd","de","df","e0","e1","e2","e3","e4","e5","e6","e7","e8","e9","ea","eb","ec","ed","ee","ef","f0","f1","f2","f3","f4","f5","f6","f7","f8","f9","fa","fb","fc","fd","fe","ff"],vh=Math.PI/180,Pp=180/Math.PI;function bl(){const o=Math.random()*4294967295|0,t=Math.random()*4294967295|0,i=Math.random()*4294967295|0,r=Math.random()*4294967295|0;return(On[o&255]+On[o>>8&255]+On[o>>16&255]+On[o>>24&255]+"-"+On[t&255]+On[t>>8&255]+"-"+On[t>>16&15|64]+On[t>>24&255]+"-"+On[i&63|128]+On[i>>8&255]+"-"+On[i>>16&255]+On[i>>24&255]+On[r&255]+On[r>>8&255]+On[r>>16&255]+On[r>>24&255]).toLowerCase()}function Nt(o,t,i){return Math.max(t,Math.min(i,o))}function BE(o,t){return(o%t+t)%t}function xh(o,t,i){return(1-i)*o+i*t}function ll(o,t){switch(t.constructor){case Float32Array:return o;case Uint32Array:return o/4294967295;case Uint16Array:return o/65535;case Uint8Array:case Uint8ClampedArray:return o/255;case Int32Array:return Math.max(o/2147483647,-1);case Int16Array:return Math.max(o/32767,-1);case Int8Array:return Math.max(o/127,-1);default:throw new Error("THREE.MathUtils: Invalid component type.")}}function Zn(o,t){switch(t.constructor){case Float32Array:return o;case Uint32Array:return Math.round(o*4294967295);case Uint16Array:return Math.round(o*65535);case Uint8Array:case Uint8ClampedArray:return Math.round(o*255);case Int32Array:return Math.round(o*2147483647);case Int16Array:return Math.round(o*32767);case Int8Array:return Math.round(o*127);default:throw new Error("THREE.MathUtils: Invalid component type.")}}const Jp=class Jp{constructor(t=0,i=0){this.x=t,this.y=i}get width(){return this.x}set width(t){this.x=t}get height(){return this.y}set height(t){this.y=t}set(t,i){return this.x=t,this.y=i,this}setScalar(t){return this.x=t,this.y=t,this}setX(t){return this.x=t,this}setY(t){return this.y=t,this}setComponent(t,i){switch(t){case 0:this.x=i;break;case 1:this.y=i;break;default:throw new Error("THREE.Vector2: index is out of range: "+t)}return this}getComponent(t){switch(t){case 0:return this.x;case 1:return this.y;default:throw new Error("THREE.Vector2: index is out of range: "+t)}}clone(){return new this.constructor(this.x,this.y)}copy(t){return this.x=t.x,this.y=t.y,this}add(t){return this.x+=t.x,this.y+=t.y,this}addScalar(t){return this.x+=t,this.y+=t,this}addVectors(t,i){return this.x=t.x+i.x,this.y=t.y+i.y,this}addScaledVector(t,i){return this.x+=t.x*i,this.y+=t.y*i,this}sub(t){return this.x-=t.x,this.y-=t.y,this}subScalar(t){return this.x-=t,this.y-=t,this}subVectors(t,i){return this.x=t.x-i.x,this.y=t.y-i.y,this}multiply(t){return this.x*=t.x,this.y*=t.y,this}multiplyScalar(t){return this.x*=t,this.y*=t,this}divide(t){return this.x/=t.x,this.y/=t.y,this}divideScalar(t){return this.multiplyScalar(1/t)}applyMatrix3(t){const i=this.x,r=this.y,c=t.elements;return this.x=c[0]*i+c[3]*r+c[6],this.y=c[1]*i+c[4]*r+c[7],this}min(t){return this.x=Math.min(this.x,t.x),this.y=Math.min(this.y,t.y),this}max(t){return this.x=Math.max(this.x,t.x),this.y=Math.max(this.y,t.y),this}clamp(t,i){return this.x=Nt(this.x,t.x,i.x),this.y=Nt(this.y,t.y,i.y),this}clampScalar(t,i){return this.x=Nt(this.x,t,i),this.y=Nt(this.y,t,i),this}clampLength(t,i){const r=this.length();return this.divideScalar(r||1).multiplyScalar(Nt(r,t,i))}floor(){return this.x=Math.floor(this.x),this.y=Math.floor(this.y),this}ceil(){return this.x=Math.ceil(this.x),this.y=Math.ceil(this.y),this}round(){return this.x=Math.round(this.x),this.y=Math.round(this.y),this}roundToZero(){return this.x=Math.trunc(this.x),this.y=Math.trunc(this.y),this}negate(){return this.x=-this.x,this.y=-this.y,this}dot(t){return this.x*t.x+this.y*t.y}cross(t){return this.x*t.y-this.y*t.x}lengthSq(){return this.x*this.x+this.y*this.y}length(){return Math.sqrt(this.x*this.x+this.y*this.y)}manhattanLength(){return Math.abs(this.x)+Math.abs(this.y)}normalize(){return this.divideScalar(this.length()||1)}angle(){return Math.atan2(-this.y,-this.x)+Math.PI}angleTo(t){const i=Math.sqrt(this.lengthSq()*t.lengthSq());if(i===0)return Math.PI/2;const r=this.dot(t)/i;return Math.acos(Nt(r,-1,1))}distanceTo(t){return Math.sqrt(this.distanceToSquared(t))}distanceToSquared(t){const i=this.x-t.x,r=this.y-t.y;return i*i+r*r}manhattanDistanceTo(t){return Math.abs(this.x-t.x)+Math.abs(this.y-t.y)}setLength(t){return this.normalize().multiplyScalar(t)}lerp(t,i){return this.x+=(t.x-this.x)*i,this.y+=(t.y-this.y)*i,this}lerpVectors(t,i,r){return this.x=t.x+(i.x-t.x)*r,this.y=t.y+(i.y-t.y)*r,this}equals(t){return t.x===this.x&&t.y===this.y}fromArray(t,i=0){return this.x=t[i],this.y=t[i+1],this}toArray(t=[],i=0){return t[i]=this.x,t[i+1]=this.y,t}fromBufferAttribute(t,i){return this.x=t.getX(i),this.y=t.getY(i),this}rotateAround(t,i){const r=Math.cos(i),c=Math.sin(i),f=this.x-t.x,d=this.y-t.y;return this.x=f*r-d*c+t.x,this.y=f*c+d*r+t.y,this}random(){return this.x=Math.random(),this.y=Math.random(),this}*[Symbol.iterator](){yield this.x,yield this.y}};Jp.prototype.isVector2=!0;let Ut=Jp;class ro{constructor(t=0,i=0,r=0,c=1){this.isQuaternion=!0,this._x=t,this._y=i,this._z=r,this._w=c}static slerpFlat(t,i,r,c,f,d,h){let m=r[c+0],g=r[c+1],x=r[c+2],_=r[c+3],v=f[d+0],E=f[d+1],R=f[d+2],N=f[d+3];if(_!==N||m!==v||g!==E||x!==R){let y=m*v+g*E+x*R+_*N;y<0&&(v=-v,E=-E,R=-R,N=-N,y=-y);let S=1-h;if(y<.9995){const z=Math.acos(y),W=Math.sin(z);S=Math.sin(S*z)/W,h=Math.sin(h*z)/W,m=m*S+v*h,g=g*S+E*h,x=x*S+R*h,_=_*S+N*h}else{m=m*S+v*h,g=g*S+E*h,x=x*S+R*h,_=_*S+N*h;const z=1/Math.sqrt(m*m+g*g+x*x+_*_);m*=z,g*=z,x*=z,_*=z}}t[i]=m,t[i+1]=g,t[i+2]=x,t[i+3]=_}static multiplyQuaternionsFlat(t,i,r,c,f,d){const h=r[c],m=r[c+1],g=r[c+2],x=r[c+3],_=f[d],v=f[d+1],E=f[d+2],R=f[d+3];return t[i]=h*R+x*_+m*E-g*v,t[i+1]=m*R+x*v+g*_-h*E,t[i+2]=g*R+x*E+h*v-m*_,t[i+3]=x*R-h*_-m*v-g*E,t}get x(){return this._x}set x(t){this._x=t,this._onChangeCallback()}get y(){return this._y}set y(t){this._y=t,this._onChangeCallback()}get z(){return this._z}set z(t){this._z=t,this._onChangeCallback()}get w(){return this._w}set w(t){this._w=t,this._onChangeCallback()}set(t,i,r,c){return this._x=t,this._y=i,this._z=r,this._w=c,this._onChangeCallback(),this}clone(){return new this.constructor(this._x,this._y,this._z,this._w)}copy(t){return this._x=t.x,this._y=t.y,this._z=t.z,this._w=t.w,this._onChangeCallback(),this}setFromEuler(t,i=!0){const r=t._x,c=t._y,f=t._z,d=t._order,h=Math.cos,m=Math.sin,g=h(r/2),x=h(c/2),_=h(f/2),v=m(r/2),E=m(c/2),R=m(f/2);switch(d){case"XYZ":this._x=v*x*_+g*E*R,this._y=g*E*_-v*x*R,this._z=g*x*R+v*E*_,this._w=g*x*_-v*E*R;break;case"YXZ":this._x=v*x*_+g*E*R,this._y=g*E*_-v*x*R,this._z=g*x*R-v*E*_,this._w=g*x*_+v*E*R;break;case"ZXY":this._x=v*x*_-g*E*R,this._y=g*E*_+v*x*R,this._z=g*x*R+v*E*_,this._w=g*x*_-v*E*R;break;case"ZYX":this._x=v*x*_-g*E*R,this._y=g*E*_+v*x*R,this._z=g*x*R-v*E*_,this._w=g*x*_+v*E*R;break;case"YZX":this._x=v*x*_+g*E*R,this._y=g*E*_+v*x*R,this._z=g*x*R-v*E*_,this._w=g*x*_-v*E*R;break;case"XZY":this._x=v*x*_-g*E*R,this._y=g*E*_-v*x*R,this._z=g*x*R+v*E*_,this._w=g*x*_+v*E*R;break;default:ot("Quaternion: .setFromEuler() encountered an unknown order: "+d)}return i===!0&&this._onChangeCallback(),this}setFromAxisAngle(t,i){const r=i/2,c=Math.sin(r);return this._x=t.x*c,this._y=t.y*c,this._z=t.z*c,this._w=Math.cos(r),this._onChangeCallback(),this}setFromRotationMatrix(t){const i=t.elements,r=i[0],c=i[4],f=i[8],d=i[1],h=i[5],m=i[9],g=i[2],x=i[6],_=i[10],v=r+h+_;if(v>0){const E=.5/Math.sqrt(v+1);this._w=.25/E,this._x=(x-m)*E,this._y=(f-g)*E,this._z=(d-c)*E}else if(r>h&&r>_){const E=2*Math.sqrt(1+r-h-_);this._w=(x-m)/E,this._x=.25*E,this._y=(c+d)/E,this._z=(f+g)/E}else if(h>_){const E=2*Math.sqrt(1+h-r-_);this._w=(f-g)/E,this._x=(c+d)/E,this._y=.25*E,this._z=(m+x)/E}else{const E=2*Math.sqrt(1+_-r-h);this._w=(d-c)/E,this._x=(f+g)/E,this._y=(m+x)/E,this._z=.25*E}return this._onChangeCallback(),this}setFromUnitVectors(t,i){let r=t.dot(i)+1;return r<1e-8?(r=0,Math.abs(t.x)>Math.abs(t.z)?(this._x=-t.y,this._y=t.x,this._z=0,this._w=r):(this._x=0,this._y=-t.z,this._z=t.y,this._w=r)):(this._x=t.y*i.z-t.z*i.y,this._y=t.z*i.x-t.x*i.z,this._z=t.x*i.y-t.y*i.x,this._w=r),this.normalize()}angleTo(t){return 2*Math.acos(Math.abs(Nt(this.dot(t),-1,1)))}rotateTowards(t,i){const r=this.angleTo(t);if(r===0)return this;const c=Math.min(1,i/r);return this.slerp(t,c),this}identity(){return this.set(0,0,0,1)}invert(){return this.conjugate()}conjugate(){return this._x*=-1,this._y*=-1,this._z*=-1,this._onChangeCallback(),this}dot(t){return this._x*t._x+this._y*t._y+this._z*t._z+this._w*t._w}lengthSq(){return this._x*this._x+this._y*this._y+this._z*this._z+this._w*this._w}length(){return Math.sqrt(this._x*this._x+this._y*this._y+this._z*this._z+this._w*this._w)}normalize(){let t=this.length();return t===0?(this._x=0,this._y=0,this._z=0,this._w=1):(t=1/t,this._x=this._x*t,this._y=this._y*t,this._z=this._z*t,this._w=this._w*t),this._onChangeCallback(),this}multiply(t){return this.multiplyQuaternions(this,t)}premultiply(t){return this.multiplyQuaternions(t,this)}multiplyQuaternions(t,i){const r=t._x,c=t._y,f=t._z,d=t._w,h=i._x,m=i._y,g=i._z,x=i._w;return this._x=r*x+d*h+c*g-f*m,this._y=c*x+d*m+f*h-r*g,this._z=f*x+d*g+r*m-c*h,this._w=d*x-r*h-c*m-f*g,this._onChangeCallback(),this}slerp(t,i){let r=t._x,c=t._y,f=t._z,d=t._w,h=this.dot(t);h<0&&(r=-r,c=-c,f=-f,d=-d,h=-h);let m=1-i;if(h<.9995){const g=Math.acos(h),x=Math.sin(g);m=Math.sin(m*g)/x,i=Math.sin(i*g)/x,this._x=this._x*m+r*i,this._y=this._y*m+c*i,this._z=this._z*m+f*i,this._w=this._w*m+d*i,this._onChangeCallback()}else this._x=this._x*m+r*i,this._y=this._y*m+c*i,this._z=this._z*m+f*i,this._w=this._w*m+d*i,this.normalize();return this}slerpQuaternions(t,i,r){return this.copy(t).slerp(i,r)}random(){const t=2*Math.PI*Math.random(),i=2*Math.PI*Math.random(),r=Math.random(),c=Math.sqrt(1-r),f=Math.sqrt(r);return this.set(c*Math.sin(t),c*Math.cos(t),f*Math.sin(i),f*Math.cos(i))}equals(t){return t._x===this._x&&t._y===this._y&&t._z===this._z&&t._w===this._w}fromArray(t,i=0){return this._x=t[i],this._y=t[i+1],this._z=t[i+2],this._w=t[i+3],this._onChangeCallback(),this}toArray(t=[],i=0){return t[i]=this._x,t[i+1]=this._y,t[i+2]=this._z,t[i+3]=this._w,t}fromBufferAttribute(t,i){return this._x=t.getX(i),this._y=t.getY(i),this._z=t.getZ(i),this._w=t.getW(i),this._onChangeCallback(),this}toJSON(){return this.toArray()}_onChange(t){return this._onChangeCallback=t,this}_onChangeCallback(){}*[Symbol.iterator](){yield this._x,yield this._y,yield this._z,yield this._w}}const $p=class $p{constructor(t=0,i=0,r=0){this.x=t,this.y=i,this.z=r}set(t,i,r){return r===void 0&&(r=this.z),this.x=t,this.y=i,this.z=r,this}setScalar(t){return this.x=t,this.y=t,this.z=t,this}setX(t){return this.x=t,this}setY(t){return this.y=t,this}setZ(t){return this.z=t,this}setComponent(t,i){switch(t){case 0:this.x=i;break;case 1:this.y=i;break;case 2:this.z=i;break;default:throw new Error("THREE.Vector3: index is out of range: "+t)}return this}getComponent(t){switch(t){case 0:return this.x;case 1:return this.y;case 2:return this.z;default:throw new Error("THREE.Vector3: index is out of range: "+t)}}clone(){return new this.constructor(this.x,this.y,this.z)}copy(t){return this.x=t.x,this.y=t.y,this.z=t.z,this}add(t){return this.x+=t.x,this.y+=t.y,this.z+=t.z,this}addScalar(t){return this.x+=t,this.y+=t,this.z+=t,this}addVectors(t,i){return this.x=t.x+i.x,this.y=t.y+i.y,this.z=t.z+i.z,this}addScaledVector(t,i){return this.x+=t.x*i,this.y+=t.y*i,this.z+=t.z*i,this}sub(t){return this.x-=t.x,this.y-=t.y,this.z-=t.z,this}subScalar(t){return this.x-=t,this.y-=t,this.z-=t,this}subVectors(t,i){return this.x=t.x-i.x,this.y=t.y-i.y,this.z=t.z-i.z,this}multiply(t){return this.x*=t.x,this.y*=t.y,this.z*=t.z,this}multiplyScalar(t){return this.x*=t,this.y*=t,this.z*=t,this}multiplyVectors(t,i){return this.x=t.x*i.x,this.y=t.y*i.y,this.z=t.z*i.z,this}applyEuler(t){return this.applyQuaternion(bv.setFromEuler(t))}applyAxisAngle(t,i){return this.applyQuaternion(bv.setFromAxisAngle(t,i))}applyMatrix3(t){const i=this.x,r=this.y,c=this.z,f=t.elements;return this.x=f[0]*i+f[3]*r+f[6]*c,this.y=f[1]*i+f[4]*r+f[7]*c,this.z=f[2]*i+f[5]*r+f[8]*c,this}applyNormalMatrix(t){return this.applyMatrix3(t).normalize()}applyMatrix4(t){const i=this.x,r=this.y,c=this.z,f=t.elements,d=1/(f[3]*i+f[7]*r+f[11]*c+f[15]);return this.x=(f[0]*i+f[4]*r+f[8]*c+f[12])*d,this.y=(f[1]*i+f[5]*r+f[9]*c+f[13])*d,this.z=(f[2]*i+f[6]*r+f[10]*c+f[14])*d,this}applyQuaternion(t){const i=this.x,r=this.y,c=this.z,f=t.x,d=t.y,h=t.z,m=t.w,g=2*(d*c-h*r),x=2*(h*i-f*c),_=2*(f*r-d*i);return this.x=i+m*g+d*_-h*x,this.y=r+m*x+h*g-f*_,this.z=c+m*_+f*x-d*g,this}project(t){return this.applyMatrix4(t.matrixWorldInverse).applyMatrix4(t.projectionMatrix)}unproject(t){return this.applyMatrix4(t.projectionMatrixInverse).applyMatrix4(t.matrixWorld)}transformDirection(t){const i=this.x,r=this.y,c=this.z,f=t.elements;return this.x=f[0]*i+f[4]*r+f[8]*c,this.y=f[1]*i+f[5]*r+f[9]*c,this.z=f[2]*i+f[6]*r+f[10]*c,this.normalize()}divide(t){return this.x/=t.x,this.y/=t.y,this.z/=t.z,this}divideScalar(t){return this.multiplyScalar(1/t)}min(t){return this.x=Math.min(this.x,t.x),this.y=Math.min(this.y,t.y),this.z=Math.min(this.z,t.z),this}max(t){return this.x=Math.max(this.x,t.x),this.y=Math.max(this.y,t.y),this.z=Math.max(this.z,t.z),this}clamp(t,i){return this.x=Nt(this.x,t.x,i.x),this.y=Nt(this.y,t.y,i.y),this.z=Nt(this.z,t.z,i.z),this}clampScalar(t,i){return this.x=Nt(this.x,t,i),this.y=Nt(this.y,t,i),this.z=Nt(this.z,t,i),this}clampLength(t,i){const r=this.length();return this.divideScalar(r||1).multiplyScalar(Nt(r,t,i))}floor(){return this.x=Math.floor(this.x),this.y=Math.floor(this.y),this.z=Math.floor(this.z),this}ceil(){return this.x=Math.ceil(this.x),this.y=Math.ceil(this.y),this.z=Math.ceil(this.z),this}round(){return this.x=Math.round(this.x),this.y=Math.round(this.y),this.z=Math.round(this.z),this}roundToZero(){return this.x=Math.trunc(this.x),this.y=Math.trunc(this.y),this.z=Math.trunc(this.z),this}negate(){return this.x=-this.x,this.y=-this.y,this.z=-this.z,this}dot(t){return this.x*t.x+this.y*t.y+this.z*t.z}lengthSq(){return this.x*this.x+this.y*this.y+this.z*this.z}length(){return Math.sqrt(this.x*this.x+this.y*this.y+this.z*this.z)}manhattanLength(){return Math.abs(this.x)+Math.abs(this.y)+Math.abs(this.z)}normalize(){return this.divideScalar(this.length()||1)}setLength(t){return this.normalize().multiplyScalar(t)}lerp(t,i){return this.x+=(t.x-this.x)*i,this.y+=(t.y-this.y)*i,this.z+=(t.z-this.z)*i,this}lerpVectors(t,i,r){return this.x=t.x+(i.x-t.x)*r,this.y=t.y+(i.y-t.y)*r,this.z=t.z+(i.z-t.z)*r,this}cross(t){return this.crossVectors(this,t)}crossVectors(t,i){const r=t.x,c=t.y,f=t.z,d=i.x,h=i.y,m=i.z;return this.x=c*m-f*h,this.y=f*d-r*m,this.z=r*h-c*d,this}projectOnVector(t){const i=t.lengthSq();if(i===0)return this.set(0,0,0);const r=t.dot(this)/i;return this.copy(t).multiplyScalar(r)}projectOnPlane(t){return Sh.copy(this).projectOnVector(t),this.sub(Sh)}reflect(t){return this.sub(Sh.copy(t).multiplyScalar(2*this.dot(t)))}angleTo(t){const i=Math.sqrt(this.lengthSq()*t.lengthSq());if(i===0)return Math.PI/2;const r=this.dot(t)/i;return Math.acos(Nt(r,-1,1))}distanceTo(t){return Math.sqrt(this.distanceToSquared(t))}distanceToSquared(t){const i=this.x-t.x,r=this.y-t.y,c=this.z-t.z;return i*i+r*r+c*c}manhattanDistanceTo(t){return Math.abs(this.x-t.x)+Math.abs(this.y-t.y)+Math.abs(this.z-t.z)}setFromSpherical(t){return this.setFromSphericalCoords(t.radius,t.phi,t.theta)}setFromSphericalCoords(t,i,r){const c=Math.sin(i)*t;return this.x=c*Math.sin(r),this.y=Math.cos(i)*t,this.z=c*Math.cos(r),this}setFromCylindrical(t){return this.setFromCylindricalCoords(t.radius,t.theta,t.y)}setFromCylindricalCoords(t,i,r){return this.x=t*Math.sin(i),this.y=r,this.z=t*Math.cos(i),this}setFromMatrixPosition(t){const i=t.elements;return this.x=i[12],this.y=i[13],this.z=i[14],this}setFromMatrixScale(t){const i=this.setFromMatrixColumn(t,0).length(),r=this.setFromMatrixColumn(t,1).length(),c=this.setFromMatrixColumn(t,2).length();return this.x=i,this.y=r,this.z=c,this}setFromMatrixColumn(t,i){return this.fromArray(t.elements,i*4)}setFromMatrix3Column(t,i){return this.fromArray(t.elements,i*3)}setFromEuler(t){return this.x=t._x,this.y=t._y,this.z=t._z,this}setFromColor(t){return this.x=t.r,this.y=t.g,this.z=t.b,this}equals(t){return t.x===this.x&&t.y===this.y&&t.z===this.z}fromArray(t,i=0){return this.x=t[i],this.y=t[i+1],this.z=t[i+2],this}toArray(t=[],i=0){return t[i]=this.x,t[i+1]=this.y,t[i+2]=this.z,t}fromBufferAttribute(t,i){return this.x=t.getX(i),this.y=t.getY(i),this.z=t.getZ(i),this}random(){return this.x=Math.random(),this.y=Math.random(),this.z=Math.random(),this}randomDirection(){const t=Math.random()*Math.PI*2,i=Math.random()*2-1,r=Math.sqrt(1-i*i);return this.x=r*Math.cos(t),this.y=i,this.z=r*Math.sin(t),this}*[Symbol.iterator](){yield this.x,yield this.y,yield this.z}};$p.prototype.isVector3=!0;let he=$p;const Sh=new he,bv=new ro,em=class em{constructor(t,i,r,c,f,d,h,m,g){this.elements=[1,0,0,0,1,0,0,0,1],t!==void 0&&this.set(t,i,r,c,f,d,h,m,g)}set(t,i,r,c,f,d,h,m,g){const x=this.elements;return x[0]=t,x[1]=c,x[2]=h,x[3]=i,x[4]=f,x[5]=m,x[6]=r,x[7]=d,x[8]=g,this}identity(){return this.set(1,0,0,0,1,0,0,0,1),this}copy(t){const i=this.elements,r=t.elements;return i[0]=r[0],i[1]=r[1],i[2]=r[2],i[3]=r[3],i[4]=r[4],i[5]=r[5],i[6]=r[6],i[7]=r[7],i[8]=r[8],this}extractBasis(t,i,r){return t.setFromMatrix3Column(this,0),i.setFromMatrix3Column(this,1),r.setFromMatrix3Column(this,2),this}setFromMatrix4(t){const i=t.elements;return this.set(i[0],i[4],i[8],i[1],i[5],i[9],i[2],i[6],i[10]),this}multiply(t){return this.multiplyMatrices(this,t)}premultiply(t){return this.multiplyMatrices(t,this)}multiplyMatrices(t,i){const r=t.elements,c=i.elements,f=this.elements,d=r[0],h=r[3],m=r[6],g=r[1],x=r[4],_=r[7],v=r[2],E=r[5],R=r[8],N=c[0],y=c[3],S=c[6],z=c[1],W=c[4],C=c[7],D=c[2],U=c[5],I=c[8];return f[0]=d*N+h*z+m*D,f[3]=d*y+h*W+m*U,f[6]=d*S+h*C+m*I,f[1]=g*N+x*z+_*D,f[4]=g*y+x*W+_*U,f[7]=g*S+x*C+_*I,f[2]=v*N+E*z+R*D,f[5]=v*y+E*W+R*U,f[8]=v*S+E*C+R*I,this}multiplyScalar(t){const i=this.elements;return i[0]*=t,i[3]*=t,i[6]*=t,i[1]*=t,i[4]*=t,i[7]*=t,i[2]*=t,i[5]*=t,i[8]*=t,this}determinant(){const t=this.elements,i=t[0],r=t[1],c=t[2],f=t[3],d=t[4],h=t[5],m=t[6],g=t[7],x=t[8];return i*d*x-i*h*g-r*f*x+r*h*m+c*f*g-c*d*m}invert(){const t=this.elements,i=t[0],r=t[1],c=t[2],f=t[3],d=t[4],h=t[5],m=t[6],g=t[7],x=t[8],_=x*d-h*g,v=h*m-x*f,E=g*f-d*m,R=i*_+r*v+c*E;if(R===0)return this.set(0,0,0,0,0,0,0,0,0);const N=1/R;return t[0]=_*N,t[1]=(c*g-x*r)*N,t[2]=(h*r-c*d)*N,t[3]=v*N,t[4]=(x*i-c*m)*N,t[5]=(c*f-h*i)*N,t[6]=E*N,t[7]=(r*m-g*i)*N,t[8]=(d*i-r*f)*N,this}transpose(){let t;const i=this.elements;return t=i[1],i[1]=i[3],i[3]=t,t=i[2],i[2]=i[6],i[6]=t,t=i[5],i[5]=i[7],i[7]=t,this}getNormalMatrix(t){return this.setFromMatrix4(t).invert().transpose()}transposeIntoArray(t){const i=this.elements;return t[0]=i[0],t[1]=i[3],t[2]=i[6],t[3]=i[1],t[4]=i[4],t[5]=i[7],t[6]=i[2],t[7]=i[5],t[8]=i[8],this}setUvTransform(t,i,r,c,f,d,h){const m=Math.cos(f),g=Math.sin(f);return this.set(r*m,r*g,-r*(m*d+g*h)+d+t,-c*g,c*m,-c*(-g*d+m*h)+h+i,0,0,1),this}scale(t,i){return no("Matrix3: .scale() is deprecated. Use .makeScale() instead."),this.premultiply(yh.makeScale(t,i)),this}rotate(t){return no("Matrix3: .rotate() is deprecated. Use .makeRotation() instead."),this.premultiply(yh.makeRotation(-t)),this}translate(t,i){return no("Matrix3: .translate() is deprecated. Use .makeTranslation() instead."),this.premultiply(yh.makeTranslation(t,i)),this}makeTranslation(t,i){return t.isVector2?this.set(1,0,t.x,0,1,t.y,0,0,1):this.set(1,0,t,0,1,i,0,0,1),this}makeRotation(t){const i=Math.cos(t),r=Math.sin(t);return this.set(i,-r,0,r,i,0,0,0,1),this}makeScale(t,i){return this.set(t,0,0,0,i,0,0,0,1),this}equals(t){const i=this.elements,r=t.elements;for(let c=0;c<9;c++)if(i[c]!==r[c])return!1;return!0}fromArray(t,i=0){for(let r=0;r<9;r++)this.elements[r]=t[r+i];return this}toArray(t=[],i=0){const r=this.elements;return t[i]=r[0],t[i+1]=r[1],t[i+2]=r[2],t[i+3]=r[3],t[i+4]=r[4],t[i+5]=r[5],t[i+6]=r[6],t[i+7]=r[7],t[i+8]=r[8],t}clone(){return new this.constructor().fromArray(this.elements)}};em.prototype.isMatrix3=!0;let ft=em;const yh=new ft,Tv=new ft().set(.4123908,.3575843,.1804808,.212639,.7151687,.0721923,.0193308,.1191948,.9505322),Av=new ft().set(3.2409699,-1.5373832,-.4986108,-.9692436,1.8759675,.0415551,.0556301,-.203977,1.0569715);function zE(){const o={enabled:!0,workingColorSpace:Au,spaces:{},convert:function(c,f,d){return this.enabled===!1||f===d||!f||!d||(this.spaces[f].transfer===Yt&&(c.r=Oa(c.r),c.g=Oa(c.g),c.b=Oa(c.b)),this.spaces[f].primaries!==this.spaces[d].primaries&&(c.applyMatrix3(this.spaces[f].toXYZ),c.applyMatrix3(this.spaces[d].fromXYZ)),this.spaces[d].transfer===Yt&&(c.r=io(c.r),c.g=io(c.g),c.b=io(c.b))),c},workingToColorSpace:function(c,f){return this.convert(c,this.workingColorSpace,f)},colorSpaceToWorking:function(c,f){return this.convert(c,f,this.workingColorSpace)},getPrimaries:function(c){return this.spaces[c].primaries},getTransfer:function(c){return c===vs?Ru:this.spaces[c].transfer},getToneMappingMode:function(c){return this.spaces[c].outputColorSpaceConfig.toneMappingMode||"standard"},getLuminanceCoefficients:function(c,f=this.workingColorSpace){return c.fromArray(this.spaces[f].luminanceCoefficients)},define:function(c){Object.assign(this.spaces,c)},_getMatrix:function(c,f,d){return c.copy(this.spaces[f].toXYZ).multiply(this.spaces[d].fromXYZ)},_getDrawingBufferColorSpace:function(c){return this.spaces[c].outputColorSpaceConfig.drawingBufferColorSpace},_getUnpackColorSpace:function(c=this.workingColorSpace){return this.spaces[c].workingColorSpaceConfig.unpackColorSpace},fromWorkingColorSpace:function(c,f){return no("ColorManagement: .fromWorkingColorSpace() has been renamed to .workingToColorSpace()."),o.workingToColorSpace(c,f)},toWorkingColorSpace:function(c,f){return no("ColorManagement: .toWorkingColorSpace() has been renamed to .colorSpaceToWorking()."),o.colorSpaceToWorking(c,f)}},t=[.64,.33,.3,.6,.15,.06],i=[.2126,.7152,.0722],r=[.3127,.329];return o.define({[Au]:{primaries:t,whitePoint:r,transfer:Ru,toXYZ:Tv,fromXYZ:Av,luminanceCoefficients:i,workingColorSpaceConfig:{unpackColorSpace:bi},outputColorSpaceConfig:{drawingBufferColorSpace:bi}},[bi]:{primaries:t,whitePoint:r,transfer:Yt,toXYZ:Tv,fromXYZ:Av,luminanceCoefficients:i,outputColorSpaceConfig:{drawingBufferColorSpace:bi}}}),o}const Ct=zE();function Oa(o){return o<.04045?o*.0773993808:Math.pow(o*.9478672986+.0521327014,2.4)}function io(o){return o<.0031308?o*12.92:1.055*Math.pow(o,.41666)-.055}let Gr;class FE{static getDataURL(t,i="image/png"){if(/^data:/i.test(t.src)||typeof HTMLCanvasElement>"u")return t.src;let r;if(t instanceof HTMLCanvasElement)r=t;else{Gr===void 0&&(Gr=wu("canvas")),Gr.width=t.width,Gr.height=t.height;const c=Gr.getContext("2d");t instanceof ImageData?c.putImageData(t,0,0):c.drawImage(t,0,0,t.width,t.height),r=Gr}return r.toDataURL(i)}static sRGBToLinear(t){if(typeof HTMLImageElement<"u"&&t instanceof HTMLImageElement||typeof HTMLCanvasElement<"u"&&t instanceof HTMLCanvasElement||typeof ImageBitmap<"u"&&t instanceof ImageBitmap){const i=wu("canvas");i.width=t.width,i.height=t.height;const r=i.getContext("2d");r.drawImage(t,0,0,t.width,t.height);const c=r.getImageData(0,0,t.width,t.height),f=c.data;for(let d=0;d<f.length;d++)f[d]=Oa(f[d]/255)*255;return r.putImageData(c,0,0),i}else if(t.data){const i=t.data.slice(0);for(let r=0;r<i.length;r++)i instanceof Uint8Array||i instanceof Uint8ClampedArray?i[r]=Math.floor(Oa(i[r]/255)*255):i[r]=Oa(i[r]);return{data:i,width:t.width,height:t.height}}else return ot("ImageUtils.sRGBToLinear(): Unsupported image type. No color space conversion applied."),t}}let HE=0;class Zp{constructor(t=null){this.isTextureSource=!0,Object.defineProperty(this,"id",{value:HE++}),this.uuid=bl(),this.data=t,this.dataReady=!0,this.version=0}getSize(t){const i=this.data;return typeof HTMLVideoElement<"u"&&i instanceof HTMLVideoElement?t.set(i.videoWidth,i.videoHeight,0):typeof VideoFrame<"u"&&i instanceof VideoFrame?t.set(i.displayWidth,i.displayHeight,0):i!==null?t.set(i.width,i.height,i.depth||0):t.set(0,0,0),t}set needsUpdate(t){t===!0&&this.version++}toJSON(t){const i=t===void 0||typeof t=="string";if(!i&&t.images[this.uuid]!==void 0)return t.images[this.uuid];const r={uuid:this.uuid,url:""},c=this.data;if(c!==null){let f;if(Array.isArray(c)){f=[];for(let d=0,h=c.length;d<h;d++)c[d].isDataTexture?f.push(Mh(c[d].image)):f.push(Mh(c[d]))}else f=Mh(c);r.url=f}return i||(t.images[this.uuid]=r),r}}function Mh(o){return typeof HTMLImageElement<"u"&&o instanceof HTMLImageElement||typeof HTMLCanvasElement<"u"&&o instanceof HTMLCanvasElement||typeof ImageBitmap<"u"&&o instanceof ImageBitmap?FE.getDataURL(o):o.data?{data:Array.from(o.data),width:o.width,height:o.height,type:o.data.constructor.name}:(ot("Texture: Unable to serialize Texture."),{})}let GE=0;const Eh=new he;class kn extends $s{constructor(t=kn.DEFAULT_IMAGE,i=kn.DEFAULT_MAPPING,r=Ua,c=Ua,f=In,d=js,h=Ii,m=fi,g=kn.DEFAULT_ANISOTROPY,x=vs){super(),this.isTexture=!0,Object.defineProperty(this,"id",{value:GE++}),this.uuid=bl(),this.name="",this.source=new Zp(t),this.mipmaps=[],this.mapping=i,this.channel=0,this.wrapS=r,this.wrapT=c,this.magFilter=f,this.minFilter=d,this.anisotropy=g,this.format=h,this.internalFormat=null,this.type=m,this.offset=new Ut(0,0),this.repeat=new Ut(1,1),this.center=new Ut(0,0),this.rotation=0,this.matrixAutoUpdate=!0,this.matrix=new ft,this.generateMipmaps=!0,this.premultiplyAlpha=!1,this.flipY=!0,this.unpackAlignment=4,this.colorSpace=x,this.userData={},this.updateRanges=[],this.version=0,this.onUpdate=null,this.renderTarget=null,this.isRenderTargetTexture=!1,this.isArrayTexture=!!(t&&t.depth&&t.depth>1),this.pmremVersion=0,this.normalized=!1}get width(){return this.source.getSize(Eh).x}get height(){return this.source.getSize(Eh).y}get depth(){return this.source.getSize(Eh).z}get image(){return this.source.data}set image(t){this.source.data=t}updateMatrix(){this.matrix.setUvTransform(this.offset.x,this.offset.y,this.repeat.x,this.repeat.y,this.rotation,this.center.x,this.center.y)}addUpdateRange(t,i){this.updateRanges.push({start:t,count:i})}clearUpdateRanges(){this.updateRanges.length=0}clone(){return new this.constructor().copy(this)}copy(t){return this.name=t.name,this.source=t.source,this.mipmaps=t.mipmaps.slice(0),this.mapping=t.mapping,this.channel=t.channel,this.wrapS=t.wrapS,this.wrapT=t.wrapT,this.magFilter=t.magFilter,this.minFilter=t.minFilter,this.anisotropy=t.anisotropy,this.format=t.format,this.internalFormat=t.internalFormat,this.type=t.type,this.normalized=t.normalized,this.offset.copy(t.offset),this.repeat.copy(t.repeat),this.center.copy(t.center),this.rotation=t.rotation,this.matrixAutoUpdate=t.matrixAutoUpdate,this.matrix.copy(t.matrix),this.generateMipmaps=t.generateMipmaps,this.premultiplyAlpha=t.premultiplyAlpha,this.flipY=t.flipY,this.unpackAlignment=t.unpackAlignment,this.colorSpace=t.colorSpace,this.renderTarget=t.renderTarget,this.isRenderTargetTexture=t.isRenderTargetTexture,this.isArrayTexture=t.isArrayTexture,this.userData=JSON.parse(JSON.stringify(t.userData)),this.needsUpdate=!0,this}setValues(t){for(const i in t){const r=t[i];if(r===void 0){ot(`Texture.setValues(): parameter '${i}' has value of undefined.`);continue}const c=this[i];if(c===void 0){ot(`Texture.setValues(): property '${i}' does not exist.`);continue}c&&r&&c.isVector2&&r.isVector2||c&&r&&c.isVector3&&r.isVector3||c&&r&&c.isMatrix3&&r.isMatrix3?c.copy(r):this[i]=r}}toJSON(t){const i=t===void 0||typeof t=="string";if(!i&&t.textures[this.uuid]!==void 0)return t.textures[this.uuid];const r={metadata:{version:4.7,type:"Texture",generator:"Texture.toJSON"},uuid:this.uuid,name:this.name,image:this.source.toJSON(t).uuid,mapping:this.mapping,channel:this.channel,repeat:[this.repeat.x,this.repeat.y],offset:[this.offset.x,this.offset.y],center:[this.center.x,this.center.y],rotation:this.rotation,wrap:[this.wrapS,this.wrapT],format:this.format,internalFormat:this.internalFormat,type:this.type,normalized:this.normalized,colorSpace:this.colorSpace,minFilter:this.minFilter,magFilter:this.magFilter,anisotropy:this.anisotropy,flipY:this.flipY,generateMipmaps:this.generateMipmaps,premultiplyAlpha:this.premultiplyAlpha,unpackAlignment:this.unpackAlignment};return Object.keys(this.userData).length>0&&(r.userData=this.userData),i||(t.textures[this.uuid]=r),r}dispose(){this.dispatchEvent({type:"dispose"})}transformUv(t){if(this.mapping!==Cx)return t;if(t.applyMatrix3(this.matrix),t.x<0||t.x>1)switch(this.wrapS){case np:t.x=t.x-Math.floor(t.x);break;case Ua:t.x=t.x<0?0:1;break;case ip:Math.abs(Math.floor(t.x)%2)===1?t.x=Math.ceil(t.x)-t.x:t.x=t.x-Math.floor(t.x);break}if(t.y<0||t.y>1)switch(this.wrapT){case np:t.y=t.y-Math.floor(t.y);break;case Ua:t.y=t.y<0?0:1;break;case ip:Math.abs(Math.floor(t.y)%2)===1?t.y=Math.ceil(t.y)-t.y:t.y=t.y-Math.floor(t.y);break}return this.flipY&&(t.y=1-t.y),t}set needsUpdate(t){t===!0&&(this.version++,this.source.needsUpdate=!0)}set needsPMREMUpdate(t){t===!0&&this.pmremVersion++}}kn.DEFAULT_IMAGE=null;kn.DEFAULT_MAPPING=Cx;kn.DEFAULT_ANISOTROPY=1;const tm=class tm{constructor(t=0,i=0,r=0,c=1){this.x=t,this.y=i,this.z=r,this.w=c}get width(){return this.z}set width(t){this.z=t}get height(){return this.w}set height(t){this.w=t}set(t,i,r,c){return this.x=t,this.y=i,this.z=r,this.w=c,this}setScalar(t){return this.x=t,this.y=t,this.z=t,this.w=t,this}setX(t){return this.x=t,this}setY(t){return this.y=t,this}setZ(t){return this.z=t,this}setW(t){return this.w=t,this}setComponent(t,i){switch(t){case 0:this.x=i;break;case 1:this.y=i;break;case 2:this.z=i;break;case 3:this.w=i;break;default:throw new Error("THREE.Vector4: index is out of range: "+t)}return this}getComponent(t){switch(t){case 0:return this.x;case 1:return this.y;case 2:return this.z;case 3:return this.w;default:throw new Error("THREE.Vector4: index is out of range: "+t)}}clone(){return new this.constructor(this.x,this.y,this.z,this.w)}copy(t){return this.x=t.x,this.y=t.y,this.z=t.z,this.w=t.w!==void 0?t.w:1,this}add(t){return this.x+=t.x,this.y+=t.y,this.z+=t.z,this.w+=t.w,this}addScalar(t){return this.x+=t,this.y+=t,this.z+=t,this.w+=t,this}addVectors(t,i){return this.x=t.x+i.x,this.y=t.y+i.y,this.z=t.z+i.z,this.w=t.w+i.w,this}addScaledVector(t,i){return this.x+=t.x*i,this.y+=t.y*i,this.z+=t.z*i,this.w+=t.w*i,this}sub(t){return this.x-=t.x,this.y-=t.y,this.z-=t.z,this.w-=t.w,this}subScalar(t){return this.x-=t,this.y-=t,this.z-=t,this.w-=t,this}subVectors(t,i){return this.x=t.x-i.x,this.y=t.y-i.y,this.z=t.z-i.z,this.w=t.w-i.w,this}multiply(t){return this.x*=t.x,this.y*=t.y,this.z*=t.z,this.w*=t.w,this}multiplyScalar(t){return this.x*=t,this.y*=t,this.z*=t,this.w*=t,this}applyMatrix4(t){const i=this.x,r=this.y,c=this.z,f=this.w,d=t.elements;return this.x=d[0]*i+d[4]*r+d[8]*c+d[12]*f,this.y=d[1]*i+d[5]*r+d[9]*c+d[13]*f,this.z=d[2]*i+d[6]*r+d[10]*c+d[14]*f,this.w=d[3]*i+d[7]*r+d[11]*c+d[15]*f,this}divide(t){return this.x/=t.x,this.y/=t.y,this.z/=t.z,this.w/=t.w,this}divideScalar(t){return this.multiplyScalar(1/t)}setAxisAngleFromQuaternion(t){this.w=2*Math.acos(t.w);const i=Math.sqrt(1-t.w*t.w);return i<1e-4?(this.x=1,this.y=0,this.z=0):(this.x=t.x/i,this.y=t.y/i,this.z=t.z/i),this}setAxisAngleFromRotationMatrix(t){let i,r,c,f;const m=t.elements,g=m[0],x=m[4],_=m[8],v=m[1],E=m[5],R=m[9],N=m[2],y=m[6],S=m[10];if(Math.abs(x-v)<.01&&Math.abs(_-N)<.01&&Math.abs(R-y)<.01){if(Math.abs(x+v)<.1&&Math.abs(_+N)<.1&&Math.abs(R+y)<.1&&Math.abs(g+E+S-3)<.1)return this.set(1,0,0,0),this;i=Math.PI;const W=(g+1)/2,C=(E+1)/2,D=(S+1)/2,U=(x+v)/4,I=(_+N)/4,b=(R+y)/4;return W>C&&W>D?W<.01?(r=0,c=.707106781,f=.707106781):(r=Math.sqrt(W),c=U/r,f=I/r):C>D?C<.01?(r=.707106781,c=0,f=.707106781):(c=Math.sqrt(C),r=U/c,f=b/c):D<.01?(r=.707106781,c=.707106781,f=0):(f=Math.sqrt(D),r=I/f,c=b/f),this.set(r,c,f,i),this}let z=Math.sqrt((y-R)*(y-R)+(_-N)*(_-N)+(v-x)*(v-x));return Math.abs(z)<.001&&(z=1),this.x=(y-R)/z,this.y=(_-N)/z,this.z=(v-x)/z,this.w=Math.acos((g+E+S-1)/2),this}setFromMatrixPosition(t){const i=t.elements;return this.x=i[12],this.y=i[13],this.z=i[14],this.w=i[15],this}min(t){return this.x=Math.min(this.x,t.x),this.y=Math.min(this.y,t.y),this.z=Math.min(this.z,t.z),this.w=Math.min(this.w,t.w),this}max(t){return this.x=Math.max(this.x,t.x),this.y=Math.max(this.y,t.y),this.z=Math.max(this.z,t.z),this.w=Math.max(this.w,t.w),this}clamp(t,i){return this.x=Nt(this.x,t.x,i.x),this.y=Nt(this.y,t.y,i.y),this.z=Nt(this.z,t.z,i.z),this.w=Nt(this.w,t.w,i.w),this}clampScalar(t,i){return this.x=Nt(this.x,t,i),this.y=Nt(this.y,t,i),this.z=Nt(this.z,t,i),this.w=Nt(this.w,t,i),this}clampLength(t,i){const r=this.length();return this.divideScalar(r||1).multiplyScalar(Nt(r,t,i))}floor(){return this.x=Math.floor(this.x),this.y=Math.floor(this.y),this.z=Math.floor(this.z),this.w=Math.floor(this.w),this}ceil(){return this.x=Math.ceil(this.x),this.y=Math.ceil(this.y),this.z=Math.ceil(this.z),this.w=Math.ceil(this.w),this}round(){return this.x=Math.round(this.x),this.y=Math.round(this.y),this.z=Math.round(this.z),this.w=Math.round(this.w),this}roundToZero(){return this.x=Math.trunc(this.x),this.y=Math.trunc(this.y),this.z=Math.trunc(this.z),this.w=Math.trunc(this.w),this}negate(){return this.x=-this.x,this.y=-this.y,this.z=-this.z,this.w=-this.w,this}dot(t){return this.x*t.x+this.y*t.y+this.z*t.z+this.w*t.w}lengthSq(){return this.x*this.x+this.y*this.y+this.z*this.z+this.w*this.w}length(){return Math.sqrt(this.x*this.x+this.y*this.y+this.z*this.z+this.w*this.w)}manhattanLength(){return Math.abs(this.x)+Math.abs(this.y)+Math.abs(this.z)+Math.abs(this.w)}normalize(){return this.divideScalar(this.length()||1)}setLength(t){return this.normalize().multiplyScalar(t)}lerp(t,i){return this.x+=(t.x-this.x)*i,this.y+=(t.y-this.y)*i,this.z+=(t.z-this.z)*i,this.w+=(t.w-this.w)*i,this}lerpVectors(t,i,r){return this.x=t.x+(i.x-t.x)*r,this.y=t.y+(i.y-t.y)*r,this.z=t.z+(i.z-t.z)*r,this.w=t.w+(i.w-t.w)*r,this}equals(t){return t.x===this.x&&t.y===this.y&&t.z===this.z&&t.w===this.w}fromArray(t,i=0){return this.x=t[i],this.y=t[i+1],this.z=t[i+2],this.w=t[i+3],this}toArray(t=[],i=0){return t[i]=this.x,t[i+1]=this.y,t[i+2]=this.z,t[i+3]=this.w,t}fromBufferAttribute(t,i){return this.x=t.getX(i),this.y=t.getY(i),this.z=t.getZ(i),this.w=t.getW(i),this}random(){return this.x=Math.random(),this.y=Math.random(),this.z=Math.random(),this.w=Math.random(),this}*[Symbol.iterator](){yield this.x,yield this.y,yield this.z,yield this.w}};tm.prototype.isVector4=!0;let sn=tm;class VE extends $s{constructor(t=1,i=1,r={}){super(),r=Object.assign({generateMipmaps:!1,internalFormat:null,minFilter:In,depthBuffer:!0,stencilBuffer:!1,resolveColorBuffer:!0,resolveDepthBuffer:!0,resolveStencilBuffer:!0,storeMultisampledColorBuffer:!0,storeMultisampledDepthBuffer:!0,storeMultisampledStencilBuffer:!0,depthTexture:null,samples:0,count:1,depth:1,multiview:!1,useArrayDepthTexture:!1},r),this.isRenderTarget=!0,this.width=t,this.height=i,this.depth=r.depth,this.scissor=new sn(0,0,t,i),this.scissorTest=!1,this.viewport=new sn(0,0,t,i),this.textures=[];const c={width:t,height:i,depth:r.depth},f=new kn(c),d=r.count;for(let h=0;h<d;h++)this.textures[h]=f.clone(),this.textures[h].isRenderTargetTexture=!0,this.textures[h].renderTarget=this;this._setTextureOptions(r),this.depthBuffer=r.depthBuffer,this.stencilBuffer=r.stencilBuffer,this.resolveColorBuffer=r.resolveColorBuffer,this.resolveDepthBuffer=r.resolveDepthBuffer,this.resolveStencilBuffer=r.resolveStencilBuffer,this.storeMultisampledColorBuffer=r.storeMultisampledColorBuffer,this.storeMultisampledDepthBuffer=r.storeMultisampledDepthBuffer,this.storeMultisampledStencilBuffer=r.storeMultisampledStencilBuffer,this._depthTexture=null,this.depthTexture=r.depthTexture,this.samples=r.samples,this.multiview=r.multiview,this.useArrayDepthTexture=r.useArrayDepthTexture}_setTextureOptions(t={}){const i={minFilter:In,generateMipmaps:!1,flipY:!1,internalFormat:null};t.mapping!==void 0&&(i.mapping=t.mapping),t.wrapS!==void 0&&(i.wrapS=t.wrapS),t.wrapT!==void 0&&(i.wrapT=t.wrapT),t.wrapR!==void 0&&(i.wrapR=t.wrapR),t.magFilter!==void 0&&(i.magFilter=t.magFilter),t.minFilter!==void 0&&(i.minFilter=t.minFilter),t.format!==void 0&&(i.format=t.format),t.type!==void 0&&(i.type=t.type),t.anisotropy!==void 0&&(i.anisotropy=t.anisotropy),t.colorSpace!==void 0&&(i.colorSpace=t.colorSpace),t.flipY!==void 0&&(i.flipY=t.flipY),t.generateMipmaps!==void 0&&(i.generateMipmaps=t.generateMipmaps),t.internalFormat!==void 0&&(i.internalFormat=t.internalFormat);for(let r=0;r<this.textures.length;r++)this.textures[r].setValues(i)}get texture(){return this.textures[0]}set texture(t){this.textures[0]=t}set depthTexture(t){this._depthTexture!==null&&this._depthTexture.renderTarget===this&&(this._depthTexture.renderTarget=null),t!==null&&t.renderTarget===null&&(t.renderTarget=this),this._depthTexture=t}get depthTexture(){return this._depthTexture}setSize(t,i,r=1){if(this.width!==t||this.height!==i||this.depth!==r){this.width=t,this.height=i,this.depth=r;for(let c=0,f=this.textures.length;c<f;c++)this.textures[c].image.width=t,this.textures[c].image.height=i,this.textures[c].image.depth=r,this.textures[c].isData3DTexture!==!0&&(this.textures[c].isArrayTexture=this.textures[c].image.depth>1);this.dispose()}this.viewport.set(0,0,t,i),this.scissor.set(0,0,t,i)}clone(){return new this.constructor().copy(this)}copy(t){this.width=t.width,this.height=t.height,this.depth=t.depth,this.scissor.copy(t.scissor),this.scissorTest=t.scissorTest,this.viewport.copy(t.viewport),this.textures.length=0;for(let i=0,r=t.textures.length;i<r;i++){this.textures[i]=t.textures[i].clone(),this.textures[i].isRenderTargetTexture=!0,this.textures[i].renderTarget=this;const c=Object.assign({},t.textures[i].image);this.textures[i].source=new Zp(c)}if(this.depthBuffer=t.depthBuffer,this.stencilBuffer=t.stencilBuffer,this.resolveColorBuffer=t.resolveColorBuffer,this.resolveDepthBuffer=t.resolveDepthBuffer,this.resolveStencilBuffer=t.resolveStencilBuffer,this.storeMultisampledColorBuffer=t.storeMultisampledColorBuffer,this.storeMultisampledDepthBuffer=t.storeMultisampledDepthBuffer,this.storeMultisampledStencilBuffer=t.storeMultisampledStencilBuffer,t.depthTexture!==null)if(t.depthTexture.renderTarget===t){const i=t.depthTexture.clone();i.renderTarget=null,this.depthTexture=i}else this.depthTexture=t.depthTexture;return this.samples=t.samples,this.multiview=t.multiview,this.useArrayDepthTexture=t.useArrayDepthTexture,this}dispose(){this.dispatchEvent({type:"dispose"})}}class Bi extends VE{constructor(t=1,i=1,r={}){super(t,i,r),this.isWebGLRenderTarget=!0}}class zx extends kn{constructor(t=null,i=1,r=1,c=1){super(null),this.isDataArrayTexture=!0,this.image={data:t,width:i,height:r,depth:c},this.magFilter=Un,this.minFilter=Un,this.wrapR=Ua,this.generateMipmaps=!1,this.flipY=!1,this.unpackAlignment=1,this.layerUpdates=new Set}copy(t){return super.copy(t),this.wrapR=t.wrapR,this}addLayerUpdate(t){this.layerUpdates.add(t)}clearLayerUpdates(){this.layerUpdates.clear()}}class kE extends kn{constructor(t=null,i=1,r=1,c=1){super(null),this.isData3DTexture=!0,this.image={data:t,width:i,height:r,depth:c},this.magFilter=Un,this.minFilter=Un,this.wrapR=Ua,this.generateMipmaps=!1,this.flipY=!1,this.unpackAlignment=1}copy(t){return super.copy(t),this.wrapR=t.wrapR,this}}const Nu=class Nu{constructor(t,i,r,c,f,d,h,m,g,x,_,v,E,R,N,y){this.elements=[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1],t!==void 0&&this.set(t,i,r,c,f,d,h,m,g,x,_,v,E,R,N,y)}set(t,i,r,c,f,d,h,m,g,x,_,v,E,R,N,y){const S=this.elements;return S[0]=t,S[4]=i,S[8]=r,S[12]=c,S[1]=f,S[5]=d,S[9]=h,S[13]=m,S[2]=g,S[6]=x,S[10]=_,S[14]=v,S[3]=E,S[7]=R,S[11]=N,S[15]=y,this}identity(){return this.set(1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1),this}clone(){return new Nu().fromArray(this.elements)}copy(t){const i=this.elements,r=t.elements;return i[0]=r[0],i[1]=r[1],i[2]=r[2],i[3]=r[3],i[4]=r[4],i[5]=r[5],i[6]=r[6],i[7]=r[7],i[8]=r[8],i[9]=r[9],i[10]=r[10],i[11]=r[11],i[12]=r[12],i[13]=r[13],i[14]=r[14],i[15]=r[15],this}copyPosition(t){const i=this.elements,r=t.elements;return i[12]=r[12],i[13]=r[13],i[14]=r[14],this}setFromMatrix3(t){const i=t.elements;return this.set(i[0],i[3],i[6],0,i[1],i[4],i[7],0,i[2],i[5],i[8],0,0,0,0,1),this}extractBasis(t,i,r){return this.determinantAffine()===0?(t.set(1,0,0),i.set(0,1,0),r.set(0,0,1),this):(t.setFromMatrixColumn(this,0),i.setFromMatrixColumn(this,1),r.setFromMatrixColumn(this,2),this)}makeBasis(t,i,r){return this.set(t.x,i.x,r.x,0,t.y,i.y,r.y,0,t.z,i.z,r.z,0,0,0,0,1),this}extractRotation(t){if(t.determinantAffine()===0)return this.identity();const i=this.elements,r=t.elements,c=1/Vr.setFromMatrixColumn(t,0).length(),f=1/Vr.setFromMatrixColumn(t,1).length(),d=1/Vr.setFromMatrixColumn(t,2).length();return i[0]=r[0]*c,i[1]=r[1]*c,i[2]=r[2]*c,i[3]=0,i[4]=r[4]*f,i[5]=r[5]*f,i[6]=r[6]*f,i[7]=0,i[8]=r[8]*d,i[9]=r[9]*d,i[10]=r[10]*d,i[11]=0,i[12]=0,i[13]=0,i[14]=0,i[15]=1,this}makeRotationFromEuler(t){const i=this.elements,r=t.x,c=t.y,f=t.z,d=Math.cos(r),h=Math.sin(r),m=Math.cos(c),g=Math.sin(c),x=Math.cos(f),_=Math.sin(f);if(t.order==="XYZ"){const v=d*x,E=d*_,R=h*x,N=h*_;i[0]=m*x,i[4]=-m*_,i[8]=g,i[1]=E+R*g,i[5]=v-N*g,i[9]=-h*m,i[2]=N-v*g,i[6]=R+E*g,i[10]=d*m}else if(t.order==="YXZ"){const v=m*x,E=m*_,R=g*x,N=g*_;i[0]=v+N*h,i[4]=R*h-E,i[8]=d*g,i[1]=d*_,i[5]=d*x,i[9]=-h,i[2]=E*h-R,i[6]=N+v*h,i[10]=d*m}else if(t.order==="ZXY"){const v=m*x,E=m*_,R=g*x,N=g*_;i[0]=v-N*h,i[4]=-d*_,i[8]=R+E*h,i[1]=E+R*h,i[5]=d*x,i[9]=N-v*h,i[2]=-d*g,i[6]=h,i[10]=d*m}else if(t.order==="ZYX"){const v=d*x,E=d*_,R=h*x,N=h*_;i[0]=m*x,i[4]=R*g-E,i[8]=v*g+N,i[1]=m*_,i[5]=N*g+v,i[9]=E*g-R,i[2]=-g,i[6]=h*m,i[10]=d*m}else if(t.order==="YZX"){const v=d*m,E=d*g,R=h*m,N=h*g;i[0]=m*x,i[4]=N-v*_,i[8]=R*_+E,i[1]=_,i[5]=d*x,i[9]=-h*x,i[2]=-g*x,i[6]=E*_+R,i[10]=v-N*_}else if(t.order==="XZY"){const v=d*m,E=d*g,R=h*m,N=h*g;i[0]=m*x,i[4]=-_,i[8]=g*x,i[1]=v*_+N,i[5]=d*x,i[9]=E*_-R,i[2]=R*_-E,i[6]=h*x,i[10]=N*_+v}return i[3]=0,i[7]=0,i[11]=0,i[12]=0,i[13]=0,i[14]=0,i[15]=1,this}makeRotationFromQuaternion(t){return this.compose(XE,t,WE)}lookAt(t,i,r){const c=this.elements;return li.subVectors(t,i),li.lengthSq()===0&&(li.z=1),li.normalize(),fs.crossVectors(r,li),fs.lengthSq()===0&&(Math.abs(r.z)===1?li.x+=1e-4:li.z+=1e-4,li.normalize(),fs.crossVectors(r,li)),fs.normalize(),Kc.crossVectors(li,fs),c[0]=fs.x,c[4]=Kc.x,c[8]=li.x,c[1]=fs.y,c[5]=Kc.y,c[9]=li.y,c[2]=fs.z,c[6]=Kc.z,c[10]=li.z,this}multiply(t){return this.multiplyMatrices(this,t)}premultiply(t){return this.multiplyMatrices(t,this)}multiplyMatrices(t,i){const r=t.elements,c=i.elements,f=this.elements,d=r[0],h=r[4],m=r[8],g=r[12],x=r[1],_=r[5],v=r[9],E=r[13],R=r[2],N=r[6],y=r[10],S=r[14],z=r[3],W=r[7],C=r[11],D=r[15],U=c[0],I=c[4],b=c[8],L=c[12],k=c[1],Y=c[5],$=c[9],O=c[13],q=c[2],ne=c[6],j=c[10],Q=c[14],pe=c[3],ce=c[7],me=c[11],be=c[15];return f[0]=d*U+h*k+m*q+g*pe,f[4]=d*I+h*Y+m*ne+g*ce,f[8]=d*b+h*$+m*j+g*me,f[12]=d*L+h*O+m*Q+g*be,f[1]=x*U+_*k+v*q+E*pe,f[5]=x*I+_*Y+v*ne+E*ce,f[9]=x*b+_*$+v*j+E*me,f[13]=x*L+_*O+v*Q+E*be,f[2]=R*U+N*k+y*q+S*pe,f[6]=R*I+N*Y+y*ne+S*ce,f[10]=R*b+N*$+y*j+S*me,f[14]=R*L+N*O+y*Q+S*be,f[3]=z*U+W*k+C*q+D*pe,f[7]=z*I+W*Y+C*ne+D*ce,f[11]=z*b+W*$+C*j+D*me,f[15]=z*L+W*O+C*Q+D*be,this}multiplyScalar(t){const i=this.elements;return i[0]*=t,i[4]*=t,i[8]*=t,i[12]*=t,i[1]*=t,i[5]*=t,i[9]*=t,i[13]*=t,i[2]*=t,i[6]*=t,i[10]*=t,i[14]*=t,i[3]*=t,i[7]*=t,i[11]*=t,i[15]*=t,this}determinant(){const t=this.elements,i=t[0],r=t[4],c=t[8],f=t[12],d=t[1],h=t[5],m=t[9],g=t[13],x=t[2],_=t[6],v=t[10],E=t[14],R=t[3],N=t[7],y=t[11],S=t[15],z=m*E-g*v,W=h*E-g*_,C=h*v-m*_,D=d*E-g*x,U=d*v-m*x,I=d*_-h*x;return i*(N*z-y*W+S*C)-r*(R*z-y*D+S*U)+c*(R*W-N*D+S*I)-f*(R*C-N*U+y*I)}determinantAffine(){const t=this.elements,i=t[0],r=t[4],c=t[8],f=t[1],d=t[5],h=t[9],m=t[2],g=t[6],x=t[10];return i*(d*x-h*g)-r*(f*x-h*m)+c*(f*g-d*m)}transpose(){const t=this.elements;let i;return i=t[1],t[1]=t[4],t[4]=i,i=t[2],t[2]=t[8],t[8]=i,i=t[6],t[6]=t[9],t[9]=i,i=t[3],t[3]=t[12],t[12]=i,i=t[7],t[7]=t[13],t[13]=i,i=t[11],t[11]=t[14],t[14]=i,this}setPosition(t,i,r){const c=this.elements;return t.isVector3?(c[12]=t.x,c[13]=t.y,c[14]=t.z):(c[12]=t,c[13]=i,c[14]=r),this}invert(){const t=this.elements,i=t[0],r=t[1],c=t[2],f=t[3],d=t[4],h=t[5],m=t[6],g=t[7],x=t[8],_=t[9],v=t[10],E=t[11],R=t[12],N=t[13],y=t[14],S=t[15],z=i*h-r*d,W=i*m-c*d,C=i*g-f*d,D=r*m-c*h,U=r*g-f*h,I=c*g-f*m,b=x*N-_*R,L=x*y-v*R,k=x*S-E*R,Y=_*y-v*N,$=_*S-E*N,O=v*S-E*y,q=z*O-W*$+C*Y+D*k-U*L+I*b;if(q===0)return this.set(0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0);const ne=1/q;return t[0]=(h*O-m*$+g*Y)*ne,t[1]=(c*$-r*O-f*Y)*ne,t[2]=(N*I-y*U+S*D)*ne,t[3]=(v*U-_*I-E*D)*ne,t[4]=(m*k-d*O-g*L)*ne,t[5]=(i*O-c*k+f*L)*ne,t[6]=(y*C-R*I-S*W)*ne,t[7]=(x*I-v*C+E*W)*ne,t[8]=(d*$-h*k+g*b)*ne,t[9]=(r*k-i*$-f*b)*ne,t[10]=(R*U-N*C+S*z)*ne,t[11]=(_*C-x*U-E*z)*ne,t[12]=(h*L-d*Y-m*b)*ne,t[13]=(i*Y-r*L+c*b)*ne,t[14]=(N*W-R*D-y*z)*ne,t[15]=(x*D-_*W+v*z)*ne,this}scale(t){const i=this.elements,r=t.x,c=t.y,f=t.z;return i[0]*=r,i[4]*=c,i[8]*=f,i[1]*=r,i[5]*=c,i[9]*=f,i[2]*=r,i[6]*=c,i[10]*=f,i[3]*=r,i[7]*=c,i[11]*=f,this}getMaxScaleOnAxis(){const t=this.elements,i=t[0]*t[0]+t[1]*t[1]+t[2]*t[2],r=t[4]*t[4]+t[5]*t[5]+t[6]*t[6],c=t[8]*t[8]+t[9]*t[9]+t[10]*t[10];return Math.sqrt(Math.max(i,r,c))}makeTranslation(t,i,r){return t.isVector3?this.set(1,0,0,t.x,0,1,0,t.y,0,0,1,t.z,0,0,0,1):this.set(1,0,0,t,0,1,0,i,0,0,1,r,0,0,0,1),this}makeRotationX(t){const i=Math.cos(t),r=Math.sin(t);return this.set(1,0,0,0,0,i,-r,0,0,r,i,0,0,0,0,1),this}makeRotationY(t){const i=Math.cos(t),r=Math.sin(t);return this.set(i,0,r,0,0,1,0,0,-r,0,i,0,0,0,0,1),this}makeRotationZ(t){const i=Math.cos(t),r=Math.sin(t);return this.set(i,-r,0,0,r,i,0,0,0,0,1,0,0,0,0,1),this}makeRotationAxis(t,i){const r=Math.cos(i),c=Math.sin(i),f=1-r,d=t.x,h=t.y,m=t.z,g=f*d,x=f*h;return this.set(g*d+r,g*h-c*m,g*m+c*h,0,g*h+c*m,x*h+r,x*m-c*d,0,g*m-c*h,x*m+c*d,f*m*m+r,0,0,0,0,1),this}makeScale(t,i,r){return this.set(t,0,0,0,0,i,0,0,0,0,r,0,0,0,0,1),this}makeShear(t,i,r,c,f,d){return this.set(1,r,f,0,t,1,d,0,i,c,1,0,0,0,0,1),this}compose(t,i,r){const c=this.elements,f=i._x,d=i._y,h=i._z,m=i._w,g=f+f,x=d+d,_=h+h,v=f*g,E=f*x,R=f*_,N=d*x,y=d*_,S=h*_,z=m*g,W=m*x,C=m*_,D=r.x,U=r.y,I=r.z;return c[0]=(1-(N+S))*D,c[1]=(E+C)*D,c[2]=(R-W)*D,c[3]=0,c[4]=(E-C)*U,c[5]=(1-(v+S))*U,c[6]=(y+z)*U,c[7]=0,c[8]=(R+W)*I,c[9]=(y-z)*I,c[10]=(1-(v+N))*I,c[11]=0,c[12]=t.x,c[13]=t.y,c[14]=t.z,c[15]=1,this}decompose(t,i,r){const c=this.elements;t.x=c[12],t.y=c[13],t.z=c[14];const f=this.determinantAffine();if(f===0)return r.set(1,1,1),i.identity(),this;let d=Vr.set(c[0],c[1],c[2]).length();const h=Vr.set(c[4],c[5],c[6]).length(),m=Vr.set(c[8],c[9],c[10]).length();f<0&&(d=-d),Ui.copy(this);const g=1/d,x=1/h,_=1/m;return Ui.elements[0]*=g,Ui.elements[1]*=g,Ui.elements[2]*=g,Ui.elements[4]*=x,Ui.elements[5]*=x,Ui.elements[6]*=x,Ui.elements[8]*=_,Ui.elements[9]*=_,Ui.elements[10]*=_,i.setFromRotationMatrix(Ui),r.x=d,r.y=h,r.z=m,this}makePerspective(t,i,r,c,f,d,h=aa,m=!1){const g=this.elements,x=2*f/(i-t),_=2*f/(r-c),v=(i+t)/(i-t),E=(r+c)/(r-c);let R,N;if(m)R=f/(d-f),N=d*f/(d-f);else if(h===aa)R=-(d+f)/(d-f),N=-2*d*f/(d-f);else if(h===Ml)R=-d/(d-f),N=-d*f/(d-f);else throw new Error("THREE.Matrix4.makePerspective(): Invalid coordinate system: "+h);return g[0]=x,g[4]=0,g[8]=v,g[12]=0,g[1]=0,g[5]=_,g[9]=E,g[13]=0,g[2]=0,g[6]=0,g[10]=R,g[14]=N,g[3]=0,g[7]=0,g[11]=-1,g[15]=0,this}makeOrthographic(t,i,r,c,f,d,h=aa,m=!1){const g=this.elements,x=2/(i-t),_=2/(r-c),v=-(i+t)/(i-t),E=-(r+c)/(r-c);let R,N;if(m)R=1/(d-f),N=d/(d-f);else if(h===aa)R=-2/(d-f),N=-(d+f)/(d-f);else if(h===Ml)R=-1/(d-f),N=-f/(d-f);else throw new Error("THREE.Matrix4.makeOrthographic(): Invalid coordinate system: "+h);return g[0]=x,g[4]=0,g[8]=0,g[12]=v,g[1]=0,g[5]=_,g[9]=0,g[13]=E,g[2]=0,g[6]=0,g[10]=R,g[14]=N,g[3]=0,g[7]=0,g[11]=0,g[15]=1,this}equals(t){const i=this.elements,r=t.elements;for(let c=0;c<16;c++)if(i[c]!==r[c])return!1;return!0}fromArray(t,i=0){for(let r=0;r<16;r++)this.elements[r]=t[r+i];return this}toArray(t=[],i=0){const r=this.elements;return t[i]=r[0],t[i+1]=r[1],t[i+2]=r[2],t[i+3]=r[3],t[i+4]=r[4],t[i+5]=r[5],t[i+6]=r[6],t[i+7]=r[7],t[i+8]=r[8],t[i+9]=r[9],t[i+10]=r[10],t[i+11]=r[11],t[i+12]=r[12],t[i+13]=r[13],t[i+14]=r[14],t[i+15]=r[15],t}};Nu.prototype.isMatrix4=!0;let un=Nu;const Vr=new he,Ui=new un,XE=new he(0,0,0),WE=new he(1,1,1),fs=new he,Kc=new he,li=new he,Rv=new un,wv=new ro;class xs{constructor(t=0,i=0,r=0,c=xs.DEFAULT_ORDER){this.isEuler=!0,this._x=t,this._y=i,this._z=r,this._order=c}get x(){return this._x}set x(t){this._x=t,this._onChangeCallback()}get y(){return this._y}set y(t){this._y=t,this._onChangeCallback()}get z(){return this._z}set z(t){this._z=t,this._onChangeCallback()}get order(){return this._order}set order(t){this._order=t,this._onChangeCallback()}set(t,i,r,c=this._order){return this._x=t,this._y=i,this._z=r,this._order=c,this._onChangeCallback(),this}clone(){return new this.constructor(this._x,this._y,this._z,this._order)}copy(t){return this._x=t._x,this._y=t._y,this._z=t._z,this._order=t._order,this._onChangeCallback(),this}setFromRotationMatrix(t,i=this._order,r=!0){const c=t.elements,f=c[0],d=c[4],h=c[8],m=c[1],g=c[5],x=c[9],_=c[2],v=c[6],E=c[10];switch(i){case"XYZ":this._y=Math.asin(Nt(h,-1,1)),Math.abs(h)<.9999999?(this._x=Math.atan2(-x,E),this._z=Math.atan2(-d,f)):(this._x=Math.atan2(v,g),this._z=0);break;case"YXZ":this._x=Math.asin(-Nt(x,-1,1)),Math.abs(x)<.9999999?(this._y=Math.atan2(h,E),this._z=Math.atan2(m,g)):(this._y=Math.atan2(-_,f),this._z=0);break;case"ZXY":this._x=Math.asin(Nt(v,-1,1)),Math.abs(v)<.9999999?(this._y=Math.atan2(-_,E),this._z=Math.atan2(-d,g)):(this._y=0,this._z=Math.atan2(m,f));break;case"ZYX":this._y=Math.asin(-Nt(_,-1,1)),Math.abs(_)<.9999999?(this._x=Math.atan2(v,E),this._z=Math.atan2(m,f)):(this._x=0,this._z=Math.atan2(-d,g));break;case"YZX":this._z=Math.asin(Nt(m,-1,1)),Math.abs(m)<.9999999?(this._x=Math.atan2(-x,g),this._y=Math.atan2(-_,f)):(this._x=0,this._y=Math.atan2(h,E));break;case"XZY":this._z=Math.asin(-Nt(d,-1,1)),Math.abs(d)<.9999999?(this._x=Math.atan2(v,g),this._y=Math.atan2(h,f)):(this._x=Math.atan2(-x,E),this._y=0);break;default:ot("Euler: .setFromRotationMatrix() encountered an unknown order: "+i)}return this._order=i,r===!0&&this._onChangeCallback(),this}setFromQuaternion(t,i,r){return Rv.makeRotationFromQuaternion(t),this.setFromRotationMatrix(Rv,i,r)}setFromVector3(t,i=this._order){return this.set(t.x,t.y,t.z,i)}reorder(t){return wv.setFromEuler(this),this.setFromQuaternion(wv,t)}equals(t){return t._x===this._x&&t._y===this._y&&t._z===this._z&&t._order===this._order}fromArray(t){return this._x=t[0],this._y=t[1],this._z=t[2],t[3]!==void 0&&(this._order=t[3]),this._onChangeCallback(),this}toArray(t=[],i=0){return t[i]=this._x,t[i+1]=this._y,t[i+2]=this._z,t[i+3]=this._order,t}_onChange(t){return this._onChangeCallback=t,this}_onChangeCallback(){}*[Symbol.iterator](){yield this._x,yield this._y,yield this._z,yield this._order}}xs.DEFAULT_ORDER="XYZ";class Fx{constructor(){this.mask=1}set(t){this.mask=(1<<t|0)>>>0}enable(t){this.mask|=1<<t|0}enableAll(){this.mask=-1}toggle(t){this.mask^=1<<t|0}disable(t){this.mask&=~(1<<t|0)}disableAll(){this.mask=0}test(t){return(this.mask&t.mask)!==0}isEnabled(t){return(this.mask&(1<<t|0))!==0}}let qE=0;const Cv=new he,kr=new ro,Aa=new un,Qc=new he,cl=new he,YE=new he,jE=new ro,Nv=new he(1,0,0),Dv=new he(0,1,0),Uv=new he(0,0,1),Lv={type:"added"},ZE={type:"removed"},Xr={type:"childadded",child:null},bh={type:"childremoved",child:null};class Qn extends $s{constructor(){super(),this.isObject3D=!0,Object.defineProperty(this,"id",{value:qE++}),this.uuid=bl(),this.name="",this.type="Object3D",this.parent=null,this.children=[],this.up=Qn.DEFAULT_UP.clone();const t=new he,i=new xs,r=new ro,c=new he(1,1,1);function f(){r.setFromEuler(i,!1)}function d(){i.setFromQuaternion(r,void 0,!1)}i._onChange(f),r._onChange(d),Object.defineProperties(this,{position:{configurable:!0,enumerable:!0,value:t},rotation:{configurable:!0,enumerable:!0,value:i},quaternion:{configurable:!0,enumerable:!0,value:r},scale:{configurable:!0,enumerable:!0,value:c},modelViewMatrix:{value:new un},normalMatrix:{value:new ft}}),this.matrix=new un,this.matrixWorld=new un,this.matrixAutoUpdate=Qn.DEFAULT_MATRIX_AUTO_UPDATE,this.matrixWorldAutoUpdate=Qn.DEFAULT_MATRIX_WORLD_AUTO_UPDATE,this.matrixWorldNeedsUpdate=!1,this.layers=new Fx,this.visible=!0,this.castShadow=!1,this.receiveShadow=!1,this.frustumCulled=!0,this.renderOrder=0,this.animations=[],this.customDepthMaterial=void 0,this.customDistanceMaterial=void 0,this.static=!1,this.userData={},this.pivot=null}onBeforeShadow(){}onAfterShadow(){}onBeforeRender(){}onAfterRender(){}applyMatrix4(t){this.matrixAutoUpdate&&this.updateMatrix(),this.matrix.premultiply(t),this.matrix.decompose(this.position,this.quaternion,this.scale)}applyQuaternion(t){return this.quaternion.premultiply(t),this}setRotationFromAxisAngle(t,i){this.quaternion.setFromAxisAngle(t,i)}setRotationFromEuler(t){this.quaternion.setFromEuler(t,!0)}setRotationFromMatrix(t){this.quaternion.setFromRotationMatrix(t)}setRotationFromQuaternion(t){this.quaternion.copy(t)}rotateOnAxis(t,i){return kr.setFromAxisAngle(t,i),this.quaternion.multiply(kr),this}rotateOnWorldAxis(t,i){return kr.setFromAxisAngle(t,i),this.quaternion.premultiply(kr),this}rotateX(t){return this.rotateOnAxis(Nv,t)}rotateY(t){return this.rotateOnAxis(Dv,t)}rotateZ(t){return this.rotateOnAxis(Uv,t)}translateOnAxis(t,i){return Cv.copy(t).applyQuaternion(this.quaternion),this.position.add(Cv.multiplyScalar(i)),this}translateX(t){return this.translateOnAxis(Nv,t)}translateY(t){return this.translateOnAxis(Dv,t)}translateZ(t){return this.translateOnAxis(Uv,t)}localToWorld(t){return this.updateWorldMatrix(!0,!1),t.applyMatrix4(this.matrixWorld)}worldToLocal(t){return this.updateWorldMatrix(!0,!1),t.applyMatrix4(Aa.copy(this.matrixWorld).invert())}lookAt(t,i,r){t.isVector3?Qc.copy(t):Qc.set(t,i,r);const c=this.parent;this.updateWorldMatrix(!0,!1),cl.setFromMatrixPosition(this.matrixWorld),this.isCamera||this.isLight?Aa.lookAt(cl,Qc,this.up):Aa.lookAt(Qc,cl,this.up),this.quaternion.setFromRotationMatrix(Aa),c&&(Aa.extractRotation(c.matrixWorld),kr.setFromRotationMatrix(Aa),this.quaternion.premultiply(kr.invert()))}add(t){if(arguments.length>1){for(let i=0;i<arguments.length;i++)this.add(arguments[i]);return this}return t===this?(It("Object3D.add: object can't be added as a child of itself.",t),this):(t&&t.isObject3D?(t.removeFromParent(),t.parent=this,this.children.push(t),t.dispatchEvent(Lv),Xr.child=t,this.dispatchEvent(Xr),Xr.child=null):It("Object3D.add: object not an instance of THREE.Object3D.",t),this)}remove(t){if(arguments.length>1){for(let r=0;r<arguments.length;r++)this.remove(arguments[r]);return this}const i=this.children.indexOf(t);return i!==-1&&(t.parent=null,this.children.splice(i,1),t.dispatchEvent(ZE),bh.child=t,this.dispatchEvent(bh),bh.child=null),this}removeFromParent(){const t=this.parent;return t!==null&&t.remove(this),this}clear(){return this.remove(...this.children)}attach(t){return this.updateWorldMatrix(!0,!1),Aa.copy(this.matrixWorld).invert(),t.parent!==null&&(t.parent.updateWorldMatrix(!0,!1),Aa.multiply(t.parent.matrixWorld)),t.applyMatrix4(Aa),t.removeFromParent(),t.parent=this,this.children.push(t),t.updateWorldMatrix(!1,!0),t.dispatchEvent(Lv),Xr.child=t,this.dispatchEvent(Xr),Xr.child=null,this}getObjectById(t){return this.getObjectByProperty("id",t)}getObjectByName(t){return this.getObjectByProperty("name",t)}getObjectByProperty(t,i){if(this[t]===i)return this;for(let r=0,c=this.children.length;r<c;r++){const d=this.children[r].getObjectByProperty(t,i);if(d!==void 0)return d}}getObjectsByProperty(t,i,r=[]){this[t]===i&&r.push(this);const c=this.children;for(let f=0,d=c.length;f<d;f++)c[f].getObjectsByProperty(t,i,r);return r}getWorldPosition(t){return this.updateWorldMatrix(!0,!1),t.setFromMatrixPosition(this.matrixWorld)}getWorldQuaternion(t){return this.updateWorldMatrix(!0,!1),this.matrixWorld.decompose(cl,t,YE),t}getWorldScale(t){return this.updateWorldMatrix(!0,!1),this.matrixWorld.decompose(cl,jE,t),t}getWorldDirection(t){this.updateWorldMatrix(!0,!1);const i=this.matrixWorld.elements;return t.set(i[8],i[9],i[10]).normalize()}raycast(){}intersectsFrustum(){}traverse(t){t(this);const i=this.children;for(let r=0,c=i.length;r<c;r++)i[r].traverse(t)}traverseVisible(t){if(this.visible===!1)return;t(this);const i=this.children;for(let r=0,c=i.length;r<c;r++)i[r].traverseVisible(t)}traverseAncestors(t){const i=this.parent;i!==null&&(t(i),i.traverseAncestors(t))}updateMatrix(){this.matrix.compose(this.position,this.quaternion,this.scale);const t=this.pivot;if(t!==null){const i=t.x,r=t.y,c=t.z,f=this.matrix.elements;f[12]+=i-f[0]*i-f[4]*r-f[8]*c,f[13]+=r-f[1]*i-f[5]*r-f[9]*c,f[14]+=c-f[2]*i-f[6]*r-f[10]*c}this.matrixWorldNeedsUpdate=!0}updateMatrixWorld(t){this.matrixAutoUpdate&&this.updateMatrix(),(this.matrixWorldNeedsUpdate||t)&&(this.matrixWorldAutoUpdate===!0&&(this.parent===null?this.matrixWorld.copy(this.matrix):this.matrixWorld.multiplyMatrices(this.parent.matrixWorld,this.matrix)),this.matrixWorldNeedsUpdate=!1,t=!0);const i=this.children;for(let r=0,c=i.length;r<c;r++)i[r].updateMatrixWorld(t)}updateWorldMatrix(t,i,r=!1){const c=this.parent;if(t===!0&&c!==null&&c.updateWorldMatrix(!0,!1),this.matrixAutoUpdate&&this.updateMatrix(),(this.matrixWorldNeedsUpdate||r)&&(this.matrixWorldAutoUpdate===!0&&(this.parent===null?this.matrixWorld.copy(this.matrix):this.matrixWorld.multiplyMatrices(this.parent.matrixWorld,this.matrix)),this.matrixWorldNeedsUpdate=!1,r=!0),i===!0){const f=this.children;for(let d=0,h=f.length;d<h;d++)f[d].updateWorldMatrix(!1,!0,r)}}toJSON(t){const i=t===void 0||typeof t=="string",r={};i&&(t={geometries:{},materials:{},textures:{},images:{},shapes:{},skeletons:{},animations:{},nodes:{}},r.metadata={version:4.7,type:"Object",generator:"Object3D.toJSON"});const c={};c.uuid=this.uuid,c.type=this.type,c.name=this.name,c.castShadow=this.castShadow,c.receiveShadow=this.receiveShadow,c.visible=this.visible,c.frustumCulled=this.frustumCulled,c.renderOrder=this.renderOrder,c.static=this.static,c.matrixAutoUpdate=this.matrixAutoUpdate,Object.keys(this.userData).length>0&&(c.userData=this.userData),c.layers=this.layers.mask,c.matrix=this.matrix.toArray(),c.up=this.up.toArray(),this.pivot!==null&&(c.pivot=this.pivot.toArray()),this.morphTargetDictionary!==void 0&&(c.morphTargetDictionary=Object.assign({},this.morphTargetDictionary)),this.morphTargetInfluences!==void 0&&(c.morphTargetInfluences=this.morphTargetInfluences.slice()),this.isInstancedMesh&&(c.type="InstancedMesh",c.count=this.count,c.instanceMatrix=this.instanceMatrix.toJSON(),this.instanceColor!==null&&(c.instanceColor=this.instanceColor.toJSON())),this.isBatchedMesh&&(c.type="BatchedMesh",c.perObjectFrustumCulled=this.perObjectFrustumCulled,c.sortObjects=this.sortObjects,c.drawRanges=this._drawRanges,c.reservedRanges=this._reservedRanges,c.geometryInfo=this._geometryInfo.map(h=>({...h,boundingBox:h.boundingBox?h.boundingBox.toJSON():void 0,boundingSphere:h.boundingSphere?h.boundingSphere.toJSON():void 0})),c.instanceInfo=this._instanceInfo.map(h=>({...h})),c.availableInstanceIds=this._availableInstanceIds.slice(),c.availableGeometryIds=this._availableGeometryIds.slice(),c.nextIndexStart=this._nextIndexStart,c.nextVertexStart=this._nextVertexStart,c.geometryCount=this._geometryCount,c.maxInstanceCount=this._maxInstanceCount,c.maxVertexCount=this._maxVertexCount,c.maxIndexCount=this._maxIndexCount,c.geometryInitialized=this._geometryInitialized,c.matricesTexture=this._matricesTexture.toJSON(t),c.indirectTexture=this._indirectTexture.toJSON(t),this._colorsTexture!==null&&(c.colorsTexture=this._colorsTexture.toJSON(t)),this.boundingSphere!==null&&(c.boundingSphere=this.boundingSphere.toJSON()),this.boundingBox!==null&&(c.boundingBox=this.boundingBox.toJSON()));function f(h,m){return h[m.uuid]===void 0&&(h[m.uuid]=m.toJSON(t)),m.uuid}if(this.isScene)this.background&&(this.background.isColor?c.background=this.background.toJSON():this.background.isTexture&&(c.background=this.background.toJSON(t).uuid)),this.environment&&this.environment.isTexture&&this.environment.isRenderTargetTexture!==!0&&(c.environment=this.environment.toJSON(t).uuid);else if(this.isMesh||this.isLine||this.isPoints){c.geometry=f(t.geometries,this.geometry);const h=this.geometry.parameters;if(h!==void 0&&h.shapes!==void 0){const m=h.shapes;if(Array.isArray(m))for(let g=0,x=m.length;g<x;g++){const _=m[g];f(t.shapes,_)}else f(t.shapes,m)}}if(this.isSkinnedMesh&&(c.bindMode=this.bindMode,c.bindMatrix=this.bindMatrix.toArray(),this.skeleton!==void 0&&(f(t.skeletons,this.skeleton),c.skeleton=this.skeleton.uuid)),this.material!==void 0)if(Array.isArray(this.material)){const h=[];for(let m=0,g=this.material.length;m<g;m++)h.push(f(t.materials,this.material[m]));c.material=h}else c.material=f(t.materials,this.material);if(this.children.length>0){c.children=[];for(let h=0;h<this.children.length;h++)c.children.push(this.children[h].toJSON(t).object)}if(this.animations.length>0){c.animations=[];for(let h=0;h<this.animations.length;h++){const m=this.animations[h];c.animations.push(f(t.animations,m))}}if(i){const h=d(t.geometries),m=d(t.materials),g=d(t.textures),x=d(t.images),_=d(t.shapes),v=d(t.skeletons),E=d(t.animations),R=d(t.nodes);h.length>0&&(r.geometries=h),m.length>0&&(r.materials=m),g.length>0&&(r.textures=g),x.length>0&&(r.images=x),_.length>0&&(r.shapes=_),v.length>0&&(r.skeletons=v),E.length>0&&(r.animations=E),R.length>0&&(r.nodes=R)}return r.object=c,r;function d(h){const m=[];for(const g in h){const x=h[g];delete x.metadata,m.push(x)}return m}}clone(t){return new this.constructor().copy(this,t)}copy(t,i=!0){if(this.name=t.name,this.up.copy(t.up),this.position.copy(t.position),this.rotation.order=t.rotation.order,this.quaternion.copy(t.quaternion),this.scale.copy(t.scale),this.pivot=t.pivot!==null?t.pivot.clone():null,this.matrix.copy(t.matrix),this.matrixWorld.copy(t.matrixWorld),this.matrixAutoUpdate=t.matrixAutoUpdate,this.matrixWorldAutoUpdate=t.matrixWorldAutoUpdate,this.matrixWorldNeedsUpdate=t.matrixWorldNeedsUpdate,this.layers.mask=t.layers.mask,this.visible=t.visible,this.castShadow=t.castShadow,this.receiveShadow=t.receiveShadow,this.frustumCulled=t.frustumCulled,this.renderOrder=t.renderOrder,this.static=t.static,this.animations=t.animations.slice(),this.userData=JSON.parse(JSON.stringify(t.userData)),i===!0)for(let r=0;r<t.children.length;r++){const c=t.children[r];this.add(c.clone())}return this}dispose(){this.dispatchEvent({type:"dispose"})}}Qn.DEFAULT_UP=new he(0,1,0);Qn.DEFAULT_MATRIX_AUTO_UPDATE=!0;Qn.DEFAULT_MATRIX_WORLD_AUTO_UPDATE=!0;class Jc extends Qn{constructor(){super(),this.isGroup=!0,this.type="Group"}}const KE={type:"move"};class Th{constructor(){this._targetRay=null,this._grip=null,this._hand=null}getHandSpace(){return this._hand===null&&(this._hand=new Jc,this._hand.matrixAutoUpdate=!1,this._hand.visible=!1,this._hand.joints={},this._hand.inputState={pinching:!1}),this._hand}getTargetRaySpace(){return this._targetRay===null&&(this._targetRay=new Jc,this._targetRay.matrixAutoUpdate=!1,this._targetRay.visible=!1,this._targetRay.hasLinearVelocity=!1,this._targetRay.linearVelocity=new he,this._targetRay.hasAngularVelocity=!1,this._targetRay.angularVelocity=new he),this._targetRay}getGripSpace(){return this._grip===null&&(this._grip=new Jc,this._grip.matrixAutoUpdate=!1,this._grip.visible=!1,this._grip.hasLinearVelocity=!1,this._grip.linearVelocity=new he,this._grip.hasAngularVelocity=!1,this._grip.angularVelocity=new he,this._grip.eventsEnabled=!1),this._grip}dispatchEvent(t){return this._targetRay!==null&&this._targetRay.dispatchEvent(t),this._grip!==null&&this._grip.dispatchEvent(t),this._hand!==null&&this._hand.dispatchEvent(t),this}connect(t){if(t&&t.hand){const i=this._hand;if(i)for(const r of t.hand.values())this._getHandJoint(i,r)}return this.dispatchEvent({type:"connected",data:t}),this}disconnect(t){return this.dispatchEvent({type:"disconnected",data:t}),this._targetRay!==null&&(this._targetRay.visible=!1),this._grip!==null&&(this._grip.visible=!1),this._hand!==null&&(this._hand.visible=!1),this}update(t,i,r){let c=null,f=null,d=null;const h=this._targetRay,m=this._grip,g=this._hand;if(t&&i.session.visibilityState!=="visible-blurred"){if(g&&t.hand){d=!0;for(const N of t.hand.values()){const y=i.getJointPose(N,r),S=this._getHandJoint(g,N);y!==null&&(S.matrix.fromArray(y.transform.matrix),S.matrix.decompose(S.position,S.rotation,S.scale),S.matrixWorldNeedsUpdate=!0,S.jointRadius=y.radius),S.visible=y!==null}const x=g.joints["index-finger-tip"],_=g.joints["thumb-tip"],v=x.position.distanceTo(_.position),E=.02,R=.005;g.inputState.pinching&&v>E+R?(g.inputState.pinching=!1,this.dispatchEvent({type:"pinchend",handedness:t.handedness,target:this})):!g.inputState.pinching&&v<=E-R&&(g.inputState.pinching=!0,this.dispatchEvent({type:"pinchstart",handedness:t.handedness,target:this}))}else m!==null&&t.gripSpace&&(f=i.getPose(t.gripSpace,r),f!==null&&(m.matrix.fromArray(f.transform.matrix),m.matrix.decompose(m.position,m.rotation,m.scale),m.matrixWorldNeedsUpdate=!0,f.linearVelocity?(m.hasLinearVelocity=!0,m.linearVelocity.copy(f.linearVelocity)):m.hasLinearVelocity=!1,f.angularVelocity?(m.hasAngularVelocity=!0,m.angularVelocity.copy(f.angularVelocity)):m.hasAngularVelocity=!1,m.eventsEnabled&&m.dispatchEvent({type:"gripUpdated",data:t,target:this})));h!==null&&(c=i.getPose(t.targetRaySpace,r),c===null&&f!==null&&(c=f),c!==null&&(h.matrix.fromArray(c.transform.matrix),h.matrix.decompose(h.position,h.rotation,h.scale),h.matrixWorldNeedsUpdate=!0,c.linearVelocity?(h.hasLinearVelocity=!0,h.linearVelocity.copy(c.linearVelocity)):h.hasLinearVelocity=!1,c.angularVelocity?(h.hasAngularVelocity=!0,h.angularVelocity.copy(c.angularVelocity)):h.hasAngularVelocity=!1,this.dispatchEvent(KE)))}return h!==null&&(h.visible=c!==null),m!==null&&(m.visible=f!==null),g!==null&&(g.visible=d!==null),this}_getHandJoint(t,i){if(t.joints[i.jointName]===void 0){const r=new Jc;r.matrixAutoUpdate=!1,r.visible=!1,t.joints[i.jointName]=r,t.add(r)}return t.joints[i.jointName]}}const Hx={aliceblue:15792383,antiquewhite:16444375,aqua:65535,aquamarine:8388564,azure:15794175,beige:16119260,bisque:16770244,black:0,blanchedalmond:16772045,blue:255,blueviolet:9055202,brown:10824234,burlywood:14596231,cadetblue:6266528,chartreuse:8388352,chocolate:13789470,coral:16744272,cornflowerblue:6591981,cornsilk:16775388,crimson:14423100,cyan:65535,darkblue:139,darkcyan:35723,darkgoldenrod:12092939,darkgray:11119017,darkgreen:25600,darkgrey:11119017,darkkhaki:12433259,darkmagenta:9109643,darkolivegreen:5597999,darkorange:16747520,darkorchid:10040012,darkred:9109504,darksalmon:15308410,darkseagreen:9419919,darkslateblue:4734347,darkslategray:3100495,darkslategrey:3100495,darkturquoise:52945,darkviolet:9699539,deeppink:16716947,deepskyblue:49151,dimgray:6908265,dimgrey:6908265,dodgerblue:2003199,firebrick:11674146,floralwhite:16775920,forestgreen:2263842,fuchsia:16711935,gainsboro:14474460,ghostwhite:16316671,gold:16766720,goldenrod:14329120,gray:8421504,green:32768,greenyellow:11403055,grey:8421504,honeydew:15794160,hotpink:16738740,indianred:13458524,indigo:4915330,ivory:16777200,khaki:15787660,lavender:15132410,lavenderblush:16773365,lawngreen:8190976,lemonchiffon:16775885,lightblue:11393254,lightcoral:15761536,lightcyan:14745599,lightgoldenrodyellow:16448210,lightgray:13882323,lightgreen:9498256,lightgrey:13882323,lightpink:16758465,lightsalmon:16752762,lightseagreen:2142890,lightskyblue:8900346,lightslategray:7833753,lightslategrey:7833753,lightsteelblue:11584734,lightyellow:16777184,lime:65280,limegreen:3329330,linen:16445670,magenta:16711935,maroon:8388608,mediumaquamarine:6737322,mediumblue:205,mediumorchid:12211667,mediumpurple:9662683,mediumseagreen:3978097,mediumslateblue:8087790,mediumspringgreen:64154,mediumturquoise:4772300,mediumvioletred:13047173,midnightblue:1644912,mintcream:16121850,mistyrose:16770273,moccasin:16770229,navajowhite:16768685,navy:128,oldlace:16643558,olive:8421376,olivedrab:7048739,orange:16753920,orangered:16729344,orchid:14315734,palegoldenrod:15657130,palegreen:10025880,paleturquoise:11529966,palevioletred:14381203,papayawhip:16773077,peachpuff:16767673,peru:13468991,pink:16761035,plum:14524637,powderblue:11591910,purple:8388736,rebeccapurple:6697881,red:16711680,rosybrown:12357519,royalblue:4286945,saddlebrown:9127187,salmon:16416882,sandybrown:16032864,seagreen:3050327,seashell:16774638,sienna:10506797,silver:12632256,skyblue:8900331,slateblue:6970061,slategray:7372944,slategrey:7372944,snow:16775930,springgreen:65407,steelblue:4620980,tan:13808780,teal:32896,thistle:14204888,tomato:16737095,turquoise:4251856,violet:15631086,wheat:16113331,white:16777215,whitesmoke:16119285,yellow:16776960,yellowgreen:10145074},ds={h:0,s:0,l:0},$c={h:0,s:0,l:0};function Ah(o,t,i){return i<0&&(i+=1),i>1&&(i-=1),i<1/6?o+(t-o)*6*i:i<1/2?t:i<2/3?o+(t-o)*6*(2/3-i):o}class Dt{constructor(t,i,r){return this.isColor=!0,this.r=1,this.g=1,this.b=1,this.set(t,i,r)}set(t,i,r){if(i===void 0&&r===void 0){const c=t;c&&c.isColor?this.copy(c):typeof c=="number"?this.setHex(c):typeof c=="string"&&this.setStyle(c)}else this.setRGB(t,i,r);return this}setScalar(t){return this.r=t,this.g=t,this.b=t,this}setHex(t,i=bi){return t=Math.floor(t),this.r=(t>>16&255)/255,this.g=(t>>8&255)/255,this.b=(t&255)/255,Ct.colorSpaceToWorking(this,i),this}setRGB(t,i,r,c=Ct.workingColorSpace){return this.r=t,this.g=i,this.b=r,Ct.colorSpaceToWorking(this,c),this}setHSL(t,i,r,c=Ct.workingColorSpace){if(t=BE(t,1),i=Nt(i,0,1),r=Nt(r,0,1),i===0)this.r=this.g=this.b=r;else{const f=r<=.5?r*(1+i):r+i-r*i,d=2*r-f;this.r=Ah(d,f,t+1/3),this.g=Ah(d,f,t),this.b=Ah(d,f,t-1/3)}return Ct.colorSpaceToWorking(this,c),this}setStyle(t,i=bi){function r(f){f!==void 0&&parseFloat(f)<1&&ot("Color: Alpha component of "+t+" will be ignored.")}let c;if(c=/^(\w+)\(([^\)]*)\)/.exec(t)){let f;const d=c[1],h=c[2];switch(d){case"rgb":case"rgba":if(f=/^\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*(?:,\s*(\d*\.?\d+)\s*)?$/.exec(h))return r(f[4]),this.setRGB(Math.min(255,parseInt(f[1],10))/255,Math.min(255,parseInt(f[2],10))/255,Math.min(255,parseInt(f[3],10))/255,i);if(f=/^\s*(\d+)\%\s*,\s*(\d+)\%\s*,\s*(\d+)\%\s*(?:,\s*(\d*\.?\d+)\s*)?$/.exec(h))return r(f[4]),this.setRGB(Math.min(100,parseInt(f[1],10))/100,Math.min(100,parseInt(f[2],10))/100,Math.min(100,parseInt(f[3],10))/100,i);break;case"hsl":case"hsla":if(f=/^\s*(\d*\.?\d+)\s*,\s*(\d*\.?\d+)\%\s*,\s*(\d*\.?\d+)\%\s*(?:,\s*(\d*\.?\d+)\s*)?$/.exec(h))return r(f[4]),this.setHSL(parseFloat(f[1])/360,parseFloat(f[2])/100,parseFloat(f[3])/100,i);break;default:ot("Color: Unknown color model "+t)}}else if(c=/^\#([A-Fa-f\d]+)$/.exec(t)){const f=c[1],d=f.length;if(d===3)return this.setRGB(parseInt(f.charAt(0),16)/15,parseInt(f.charAt(1),16)/15,parseInt(f.charAt(2),16)/15,i);if(d===6)return this.setHex(parseInt(f,16),i);ot("Color: Invalid hex color "+t)}else if(t&&t.length>0)return this.setColorName(t,i);return this}setColorName(t,i=bi){const r=Hx[t.toLowerCase()];return r!==void 0?this.setHex(r,i):ot("Color: Unknown color "+t),this}clone(){return new this.constructor(this.r,this.g,this.b)}copy(t){return this.r=t.r,this.g=t.g,this.b=t.b,this}copySRGBToLinear(t){return this.r=Oa(t.r),this.g=Oa(t.g),this.b=Oa(t.b),this}copyLinearToSRGB(t){return this.r=io(t.r),this.g=io(t.g),this.b=io(t.b),this}convertSRGBToLinear(){return this.copySRGBToLinear(this),this}convertLinearToSRGB(){return this.copyLinearToSRGB(this),this}getHex(t=bi){return Ct.workingToColorSpace(Pn.copy(this),t),Math.round(Nt(Pn.r*255,0,255))*65536+Math.round(Nt(Pn.g*255,0,255))*256+Math.round(Nt(Pn.b*255,0,255))}getHexString(t=bi){return("000000"+this.getHex(t).toString(16)).slice(-6)}getHSL(t,i=Ct.workingColorSpace){Ct.workingToColorSpace(Pn.copy(this),i);const r=Pn.r,c=Pn.g,f=Pn.b,d=Math.max(r,c,f),h=Math.min(r,c,f);let m,g;const x=(h+d)/2;if(h===d)m=0,g=0;else{const _=d-h;switch(g=x<=.5?_/(d+h):_/(2-d-h),d){case r:m=(c-f)/_+(c<f?6:0);break;case c:m=(f-r)/_+2;break;case f:m=(r-c)/_+4;break}m/=6}return t.h=m,t.s=g,t.l=x,t}getRGB(t,i=Ct.workingColorSpace){return Ct.workingToColorSpace(Pn.copy(this),i),t.r=Pn.r,t.g=Pn.g,t.b=Pn.b,t}getStyle(t=bi){Ct.workingToColorSpace(Pn.copy(this),t);const i=Pn.r,r=Pn.g,c=Pn.b;return t!==bi?`color(${t} ${i.toFixed(3)} ${r.toFixed(3)} ${c.toFixed(3)})`:`rgb(${Math.round(i*255)},${Math.round(r*255)},${Math.round(c*255)})`}offsetHSL(t,i,r){return this.getHSL(ds),this.setHSL(ds.h+t,ds.s+i,ds.l+r)}add(t){return this.r+=t.r,this.g+=t.g,this.b+=t.b,this}addColors(t,i){return this.r=t.r+i.r,this.g=t.g+i.g,this.b=t.b+i.b,this}addScalar(t){return this.r+=t,this.g+=t,this.b+=t,this}sub(t){return this.r=Math.max(0,this.r-t.r),this.g=Math.max(0,this.g-t.g),this.b=Math.max(0,this.b-t.b),this}multiply(t){return this.r*=t.r,this.g*=t.g,this.b*=t.b,this}multiplyScalar(t){return this.r*=t,this.g*=t,this.b*=t,this}lerp(t,i){return this.r+=(t.r-this.r)*i,this.g+=(t.g-this.g)*i,this.b+=(t.b-this.b)*i,this}lerpColors(t,i,r){return this.r=t.r+(i.r-t.r)*r,this.g=t.g+(i.g-t.g)*r,this.b=t.b+(i.b-t.b)*r,this}lerpHSL(t,i){this.getHSL(ds),t.getHSL($c);const r=xh(ds.h,$c.h,i),c=xh(ds.s,$c.s,i),f=xh(ds.l,$c.l,i);return this.setHSL(r,c,f),this}setFromVector3(t){return this.r=t.x,this.g=t.y,this.b=t.z,this}applyMatrix3(t){const i=this.r,r=this.g,c=this.b,f=t.elements;return this.r=f[0]*i+f[3]*r+f[6]*c,this.g=f[1]*i+f[4]*r+f[7]*c,this.b=f[2]*i+f[5]*r+f[8]*c,this}equals(t){return t.r===this.r&&t.g===this.g&&t.b===this.b}fromArray(t,i=0){return this.r=t[i],this.g=t[i+1],this.b=t[i+2],this}toArray(t=[],i=0){return t[i]=this.r,t[i+1]=this.g,t[i+2]=this.b,t}fromBufferAttribute(t,i){return this.r=t.getX(i),this.g=t.getY(i),this.b=t.getZ(i),this}toJSON(){return this.getHex()}*[Symbol.iterator](){yield this.r,yield this.g,yield this.b}}const Pn=new Dt;Dt.NAMES=Hx;class QE extends Qn{constructor(){super(),this.isScene=!0,this.type="Scene",this.background=null,this.environment=null,this.fog=null,this.backgroundBlurriness=0,this.backgroundIntensity=1,this.backgroundRotation=new xs,this.environmentIntensity=1,this.environmentRotation=new xs,this.overrideMaterial=null,typeof __THREE_DEVTOOLS__<"u"&&__THREE_DEVTOOLS__.dispatchEvent(new CustomEvent("observe",{detail:this}))}copy(t,i){return super.copy(t,i),t.background!==null&&(this.background=t.background.clone()),t.environment!==null&&(this.environment=t.environment.clone()),t.fog!==null&&(this.fog=t.fog.clone()),this.backgroundBlurriness=t.backgroundBlurriness,this.backgroundIntensity=t.backgroundIntensity,this.backgroundRotation.copy(t.backgroundRotation),this.environmentIntensity=t.environmentIntensity,this.environmentRotation.copy(t.environmentRotation),t.overrideMaterial!==null&&(this.overrideMaterial=t.overrideMaterial.clone()),this.matrixAutoUpdate=t.matrixAutoUpdate,this}toJSON(t){const i=super.toJSON(t);return this.fog!==null&&(i.object.fog=this.fog.toJSON()),i.object.backgroundBlurriness=this.backgroundBlurriness,i.object.backgroundIntensity=this.backgroundIntensity,i.object.backgroundRotation=this.backgroundRotation.toArray(),i.object.environmentIntensity=this.environmentIntensity,i.object.environmentRotation=this.environmentRotation.toArray(),i}}const Li=new he,Ra=new he,Rh=new he,wa=new he,Wr=new he,qr=new he,Ov=new he,wh=new he,Ch=new he,Nh=new he,Dh=new sn,Uh=new sn,Lh=new sn;class Pi{constructor(t=new he,i=new he,r=new he){this.a=t,this.b=i,this.c=r}static getNormal(t,i,r,c){c.subVectors(r,i),Li.subVectors(t,i),c.cross(Li);const f=c.lengthSq();return f>0?c.multiplyScalar(1/Math.sqrt(f)):c.set(0,0,0)}static getBarycoord(t,i,r,c,f){Li.subVectors(c,i),Ra.subVectors(r,i),Rh.subVectors(t,i);const d=Li.dot(Li),h=Li.dot(Ra),m=Li.dot(Rh),g=Ra.dot(Ra),x=Ra.dot(Rh),_=d*g-h*h;if(_===0)return f.set(0,0,0),null;const v=1/_,E=(g*m-h*x)*v,R=(d*x-h*m)*v;return f.set(1-E-R,R,E)}static containsPoint(t,i,r,c){return this.getBarycoord(t,i,r,c,wa)===null?!1:wa.x>=0&&wa.y>=0&&wa.x+wa.y<=1}static getInterpolation(t,i,r,c,f,d,h,m){return this.getBarycoord(t,i,r,c,wa)===null?(m.x=0,m.y=0,"z"in m&&(m.z=0),"w"in m&&(m.w=0),null):(m.setScalar(0),m.addScaledVector(f,wa.x),m.addScaledVector(d,wa.y),m.addScaledVector(h,wa.z),m)}static getInterpolatedAttribute(t,i,r,c,f,d){return Dh.setScalar(0),Uh.setScalar(0),Lh.setScalar(0),Dh.fromBufferAttribute(t,i),Uh.fromBufferAttribute(t,r),Lh.fromBufferAttribute(t,c),d.setScalar(0),d.addScaledVector(Dh,f.x),d.addScaledVector(Uh,f.y),d.addScaledVector(Lh,f.z),d}static isFrontFacing(t,i,r,c){return Li.subVectors(r,i),Ra.subVectors(t,i),Li.cross(Ra).dot(c)<0}set(t,i,r){return this.a.copy(t),this.b.copy(i),this.c.copy(r),this}setFromPointsAndIndices(t,i,r,c){return this.a.copy(t[i]),this.b.copy(t[r]),this.c.copy(t[c]),this}setFromAttributeAndIndices(t,i,r,c){return this.a.fromBufferAttribute(t,i),this.b.fromBufferAttribute(t,r),this.c.fromBufferAttribute(t,c),this}clone(){return new this.constructor().copy(this)}copy(t){return this.a.copy(t.a),this.b.copy(t.b),this.c.copy(t.c),this}getArea(){return Li.subVectors(this.c,this.b),Ra.subVectors(this.a,this.b),Li.cross(Ra).length()*.5}getMidpoint(t){return t.addVectors(this.a,this.b).add(this.c).multiplyScalar(1/3)}getNormal(t){return Pi.getNormal(this.a,this.b,this.c,t)}getPlane(t){return t.setFromCoplanarPoints(this.a,this.b,this.c)}getBarycoord(t,i){return Pi.getBarycoord(t,this.a,this.b,this.c,i)}getInterpolation(t,i,r,c,f){return Pi.getInterpolation(t,this.a,this.b,this.c,i,r,c,f)}containsPoint(t){return Pi.containsPoint(t,this.a,this.b,this.c)}isFrontFacing(t){return Pi.isFrontFacing(this.a,this.b,this.c,t)}intersectsBox(t){return t.intersectsTriangle(this)}closestPointToPoint(t,i){const r=this.a,c=this.b,f=this.c;let d,h;Wr.subVectors(c,r),qr.subVectors(f,r),wh.subVectors(t,r);const m=Wr.dot(wh),g=qr.dot(wh);if(m<=0&&g<=0)return i.copy(r);Ch.subVectors(t,c);const x=Wr.dot(Ch),_=qr.dot(Ch);if(x>=0&&_<=x)return i.copy(c);const v=m*_-x*g;if(v<=0&&m>=0&&x<=0)return d=m/(m-x),i.copy(r).addScaledVector(Wr,d);Nh.subVectors(t,f);const E=Wr.dot(Nh),R=qr.dot(Nh);if(R>=0&&E<=R)return i.copy(f);const N=E*g-m*R;if(N<=0&&g>=0&&R<=0)return h=g/(g-R),i.copy(r).addScaledVector(qr,h);const y=x*R-E*_;if(y<=0&&_-x>=0&&E-R>=0)return Ov.subVectors(f,c),h=(_-x)/(_-x+(E-R)),i.copy(c).addScaledVector(Ov,h);const S=1/(y+N+v);return d=N*S,h=v*S,i.copy(r).addScaledVector(Wr,d).addScaledVector(qr,h)}equals(t){return t.a.equals(this.a)&&t.b.equals(this.b)&&t.c.equals(this.c)}}class Tl{constructor(t=new he(1/0,1/0,1/0),i=new he(-1/0,-1/0,-1/0)){this.isBox3=!0,this.min=t,this.max=i}set(t,i){return this.min.copy(t),this.max.copy(i),this}setFromArray(t){this.makeEmpty();for(let i=0,r=t.length;i<r;i+=3)this.expandByPoint(Oi.fromArray(t,i));return this}setFromBufferAttribute(t){this.makeEmpty();for(let i=0,r=t.count;i<r;i++)this.expandByPoint(Oi.fromBufferAttribute(t,i));return this}setFromPoints(t){this.makeEmpty();for(let i=0,r=t.length;i<r;i++)this.expandByPoint(t[i]);return this}setFromCenterAndSize(t,i){const r=Oi.copy(i).multiplyScalar(.5);return this.min.copy(t).sub(r),this.max.copy(t).add(r),this}setFromObject(t,i=!1){return this.makeEmpty(),this.expandByObject(t,i)}clone(){return new this.constructor().copy(this)}copy(t){return this.min.copy(t.min),this.max.copy(t.max),this}makeEmpty(){return this.min.x=this.min.y=this.min.z=1/0,this.max.x=this.max.y=this.max.z=-1/0,this}isEmpty(){return this.max.x<this.min.x||this.max.y<this.min.y||this.max.z<this.min.z}getCenter(t){return this.isEmpty()?t.set(0,0,0):t.addVectors(this.min,this.max).multiplyScalar(.5)}getSize(t){return this.isEmpty()?t.set(0,0,0):t.subVectors(this.max,this.min)}expandByPoint(t){return this.min.min(t),this.max.max(t),this}expandByVector(t){return this.min.sub(t),this.max.add(t),this}expandByScalar(t){return this.min.addScalar(-t),this.max.addScalar(t),this}expandByObject(t,i=!1){t.updateWorldMatrix(!1,!1);const r=t.geometry;if(r!==void 0){const f=r.getAttribute("position");if(i===!0&&f!==void 0&&t.isInstancedMesh!==!0)for(let d=0,h=f.count;d<h;d++)t.isMesh===!0?t.getVertexPosition(d,Oi):Oi.fromBufferAttribute(f,d),Oi.applyMatrix4(t.matrixWorld),this.expandByPoint(Oi);else t.boundingBox!==void 0?(t.boundingBox===null&&t.computeBoundingBox(),eu.copy(t.boundingBox)):(r.boundingBox===null&&r.computeBoundingBox(),eu.copy(r.boundingBox)),eu.applyMatrix4(t.matrixWorld),this.union(eu)}const c=t.children;for(let f=0,d=c.length;f<d;f++)this.expandByObject(c[f],i);return this}containsPoint(t){return t.x>=this.min.x&&t.x<=this.max.x&&t.y>=this.min.y&&t.y<=this.max.y&&t.z>=this.min.z&&t.z<=this.max.z}containsBox(t){return this.min.x<=t.min.x&&t.max.x<=this.max.x&&this.min.y<=t.min.y&&t.max.y<=this.max.y&&this.min.z<=t.min.z&&t.max.z<=this.max.z}getParameter(t,i){return i.set((t.x-this.min.x)/(this.max.x-this.min.x),(t.y-this.min.y)/(this.max.y-this.min.y),(t.z-this.min.z)/(this.max.z-this.min.z))}intersectsBox(t){return t.max.x>=this.min.x&&t.min.x<=this.max.x&&t.max.y>=this.min.y&&t.min.y<=this.max.y&&t.max.z>=this.min.z&&t.min.z<=this.max.z}intersectsSphere(t){return this.clampPoint(t.center,Oi),Oi.distanceToSquared(t.center)<=t.radius*t.radius}intersectsPlane(t){let i,r;return t.normal.x>0?(i=t.normal.x*this.min.x,r=t.normal.x*this.max.x):(i=t.normal.x*this.max.x,r=t.normal.x*this.min.x),t.normal.y>0?(i+=t.normal.y*this.min.y,r+=t.normal.y*this.max.y):(i+=t.normal.y*this.max.y,r+=t.normal.y*this.min.y),t.normal.z>0?(i+=t.normal.z*this.min.z,r+=t.normal.z*this.max.z):(i+=t.normal.z*this.max.z,r+=t.normal.z*this.min.z),i<=-t.constant&&r>=-t.constant}intersectsTriangle(t){if(this.isEmpty())return!1;this.getCenter(ul),tu.subVectors(this.max,ul),Yr.subVectors(t.a,ul),jr.subVectors(t.b,ul),Zr.subVectors(t.c,ul),hs.subVectors(jr,Yr),ps.subVectors(Zr,jr),ks.subVectors(Yr,Zr);let i=[0,-hs.z,hs.y,0,-ps.z,ps.y,0,-ks.z,ks.y,hs.z,0,-hs.x,ps.z,0,-ps.x,ks.z,0,-ks.x,-hs.y,hs.x,0,-ps.y,ps.x,0,-ks.y,ks.x,0];return!Oh(i,Yr,jr,Zr,tu)||(i=[1,0,0,0,1,0,0,0,1],!Oh(i,Yr,jr,Zr,tu))?!1:(nu.crossVectors(hs,ps),i=[nu.x,nu.y,nu.z],Oh(i,Yr,jr,Zr,tu))}clampPoint(t,i){return i.copy(t).clamp(this.min,this.max)}distanceToPoint(t){return this.clampPoint(t,Oi).distanceTo(t)}getBoundingSphere(t){return this.isEmpty()?t.makeEmpty():(this.getCenter(t.center),t.radius=this.getSize(Oi).length()*.5),t}intersect(t){return this.min.max(t.min),this.max.min(t.max),this.isEmpty()&&this.makeEmpty(),this}union(t){return this.min.min(t.min),this.max.max(t.max),this}applyMatrix4(t){return this.isEmpty()?this:(Ca[0].set(this.min.x,this.min.y,this.min.z).applyMatrix4(t),Ca[1].set(this.min.x,this.min.y,this.max.z).applyMatrix4(t),Ca[2].set(this.min.x,this.max.y,this.min.z).applyMatrix4(t),Ca[3].set(this.min.x,this.max.y,this.max.z).applyMatrix4(t),Ca[4].set(this.max.x,this.min.y,this.min.z).applyMatrix4(t),Ca[5].set(this.max.x,this.min.y,this.max.z).applyMatrix4(t),Ca[6].set(this.max.x,this.max.y,this.min.z).applyMatrix4(t),Ca[7].set(this.max.x,this.max.y,this.max.z).applyMatrix4(t),this.setFromPoints(Ca),this)}translate(t){return this.min.add(t),this.max.add(t),this}equals(t){return t.min.equals(this.min)&&t.max.equals(this.max)}toJSON(){return{min:this.min.toArray(),max:this.max.toArray()}}fromJSON(t){return this.min.fromArray(t.min),this.max.fromArray(t.max),this}}const Ca=[new he,new he,new he,new he,new he,new he,new he,new he],Oi=new he,eu=new Tl,Yr=new he,jr=new he,Zr=new he,hs=new he,ps=new he,ks=new he,ul=new he,tu=new he,nu=new he,Xs=new he;function Oh(o,t,i,r,c){for(let f=0,d=o.length-3;f<=d;f+=3){Xs.fromArray(o,f);const h=c.x*Math.abs(Xs.x)+c.y*Math.abs(Xs.y)+c.z*Math.abs(Xs.z),m=t.dot(Xs),g=i.dot(Xs),x=r.dot(Xs);if(Math.max(-Math.max(m,g,x),Math.min(m,g,x))>h)return!1}return!0}const _n=new he,iu=new Ut;let JE=0;class Pa extends $s{constructor(t,i,r=!1){if(super(),Array.isArray(t))throw new TypeError("THREE.BufferAttribute: array should be a Typed Array.");this.isBufferAttribute=!0,Object.defineProperty(this,"id",{value:JE++}),this.name="",this.array=t,this.itemSize=i,this.count=t!==void 0?t.length/i:0,this.normalized=r,this.usage=UE,this.updateRanges=[],this.gpuType=ia,this.version=0}onUploadCallback(){}set needsUpdate(t){t===!0&&this.version++}setUsage(t){return this.usage=t,this}addUpdateRange(t,i){this.updateRanges.push({start:t,count:i})}clearUpdateRanges(){this.updateRanges.length=0}copy(t){return this.name=t.name,this.array=new t.array.constructor(t.array),this.itemSize=t.itemSize,this.count=t.count,this.normalized=t.normalized,this.usage=t.usage,this.gpuType=t.gpuType,this}copyAt(t,i,r){t*=this.itemSize,r*=i.itemSize;for(let c=0,f=this.itemSize;c<f;c++)this.array[t+c]=i.array[r+c];return this}copyArray(t){return this.array.set(t),this}applyMatrix3(t){if(this.itemSize===2)for(let i=0,r=this.count;i<r;i++)iu.fromBufferAttribute(this,i),iu.applyMatrix3(t),this.setXY(i,iu.x,iu.y);else if(this.itemSize===3)for(let i=0,r=this.count;i<r;i++)_n.fromBufferAttribute(this,i),_n.applyMatrix3(t),this.setXYZ(i,_n.x,_n.y,_n.z);return this}applyMatrix4(t){for(let i=0,r=this.count;i<r;i++)_n.fromBufferAttribute(this,i),_n.applyMatrix4(t),this.setXYZ(i,_n.x,_n.y,_n.z);return this}applyNormalMatrix(t){for(let i=0,r=this.count;i<r;i++)_n.fromBufferAttribute(this,i),_n.applyNormalMatrix(t),this.setXYZ(i,_n.x,_n.y,_n.z);return this}transformDirection(t){for(let i=0,r=this.count;i<r;i++)_n.fromBufferAttribute(this,i),_n.transformDirection(t),this.setXYZ(i,_n.x,_n.y,_n.z);return this}set(t,i=0){return this.array.set(t,i),this}getComponent(t,i){let r=this.array[t*this.itemSize+i];return this.normalized&&(r=ll(r,this.array)),r}setComponent(t,i,r){return this.normalized&&(r=Zn(r,this.array)),this.array[t*this.itemSize+i]=r,this}getX(t){let i=this.array[t*this.itemSize];return this.normalized&&(i=ll(i,this.array)),i}setX(t,i){return this.normalized&&(i=Zn(i,this.array)),this.array[t*this.itemSize]=i,this}getY(t){let i=this.array[t*this.itemSize+1];return this.normalized&&(i=ll(i,this.array)),i}setY(t,i){return this.normalized&&(i=Zn(i,this.array)),this.array[t*this.itemSize+1]=i,this}getZ(t){let i=this.array[t*this.itemSize+2];return this.normalized&&(i=ll(i,this.array)),i}setZ(t,i){return this.normalized&&(i=Zn(i,this.array)),this.array[t*this.itemSize+2]=i,this}getW(t){let i=this.array[t*this.itemSize+3];return this.normalized&&(i=ll(i,this.array)),i}setW(t,i){return this.normalized&&(i=Zn(i,this.array)),this.array[t*this.itemSize+3]=i,this}setXY(t,i,r){return t*=this.itemSize,this.normalized&&(i=Zn(i,this.array),r=Zn(r,this.array)),this.array[t+0]=i,this.array[t+1]=r,this}setXYZ(t,i,r,c){return t*=this.itemSize,this.normalized&&(i=Zn(i,this.array),r=Zn(r,this.array),c=Zn(c,this.array)),this.array[t+0]=i,this.array[t+1]=r,this.array[t+2]=c,this}setXYZW(t,i,r,c,f){return t*=this.itemSize,this.normalized&&(i=Zn(i,this.array),r=Zn(r,this.array),c=Zn(c,this.array),f=Zn(f,this.array)),this.array[t+0]=i,this.array[t+1]=r,this.array[t+2]=c,this.array[t+3]=f,this}onUpload(t){return this.onUploadCallback=t,this}clone(){return new this.constructor(this.array,this.itemSize).copy(this)}toJSON(){const t={itemSize:this.itemSize,type:this.array.constructor.name,array:Array.from(this.array),normalized:this.normalized};return t.name=this.name,t.usage=this.usage,t.gpuType=this.gpuType,t}dispose(){this.dispatchEvent({type:"dispose"})}}class Gx extends Pa{constructor(t,i,r){super(new Uint16Array(t),i,r)}}class Vx extends Pa{constructor(t,i,r){super(new Uint32Array(t),i,r)}}class Ti extends Pa{constructor(t,i,r){super(new Float32Array(t),i,r)}}const $E=new Tl,fl=new he,Ph=new he;class Kp{constructor(t=new he,i=-1){this.isSphere=!0,this.center=t,this.radius=i}set(t,i){return this.center.copy(t),this.radius=i,this}setFromPoints(t,i){const r=this.center;i!==void 0?r.copy(i):$E.setFromPoints(t).getCenter(r);let c=0;for(let f=0,d=t.length;f<d;f++)c=Math.max(c,r.distanceToSquared(t[f]));return this.radius=Math.sqrt(c),this}copy(t){return this.center.copy(t.center),this.radius=t.radius,this}isEmpty(){return this.radius<0}makeEmpty(){return this.center.set(0,0,0),this.radius=-1,this}containsPoint(t){return t.distanceToSquared(this.center)<=this.radius*this.radius}distanceToPoint(t){return t.distanceTo(this.center)-this.radius}intersectsSphere(t){const i=this.radius+t.radius;return t.center.distanceToSquared(this.center)<=i*i}intersectsBox(t){return t.intersectsSphere(this)}intersectsPlane(t){return Math.abs(t.distanceToPoint(this.center))<=this.radius}clampPoint(t,i){const r=this.center.distanceToSquared(t);return i.copy(t),r>this.radius*this.radius&&(i.sub(this.center).normalize(),i.multiplyScalar(this.radius).add(this.center)),i}getBoundingBox(t){return this.isEmpty()?(t.makeEmpty(),t):(t.set(this.center,this.center),t.expandByScalar(this.radius),t)}applyMatrix4(t){return this.center.applyMatrix4(t),this.radius=this.radius*t.getMaxScaleOnAxis(),this}translate(t){return this.center.add(t),this}expandByPoint(t){if(this.isEmpty())return this.center.copy(t),this.radius=0,this;fl.subVectors(t,this.center);const i=fl.lengthSq();if(i>this.radius*this.radius){const r=Math.sqrt(i),c=(r-this.radius)*.5;this.center.addScaledVector(fl,c/r),this.radius+=c}return this}union(t){return t.isEmpty()?this:this.isEmpty()?(this.copy(t),this):(this.center.equals(t.center)===!0?this.radius=Math.max(this.radius,t.radius):(Ph.subVectors(t.center,this.center).setLength(t.radius),this.expandByPoint(fl.copy(t.center).add(Ph)),this.expandByPoint(fl.copy(t.center).sub(Ph))),this)}equals(t){return t.center.equals(this.center)&&t.radius===this.radius}clone(){return new this.constructor().copy(this)}toJSON(){return{radius:this.radius,center:this.center.toArray()}}fromJSON(t){return this.radius=t.radius,this.center.fromArray(t.center),this}}let eb=0;const Ei=new un,Ih=new Qn,Kr=new he,ci=new Tl,dl=new Tl,Tn=new he;class ua extends $s{constructor(){super(),this.isBufferGeometry=!0,Object.defineProperty(this,"id",{value:eb++}),this.uuid=bl(),this.name="",this.type="BufferGeometry",this.index=null,this.indirect=null,this.indirectOffset=0,this.attributes={},this.morphAttributes={},this.morphTargetsRelative=!1,this.groups=[],this.boundingBox=null,this.boundingSphere=null,this.drawRange={start:0,count:1/0},this.userData={},this._transformed=!1}getIndex(){return this.index}setIndex(t){return Array.isArray(t)?this.index=new(LE(t)?Vx:Gx)(t,1):this.index=t,this}setIndirect(t,i=0){return this.indirect=t,this.indirectOffset=i,this}getIndirect(){return this.indirect}getAttribute(t){return this.attributes[t]}setAttribute(t,i){return this.attributes[t]=i,this}deleteAttribute(t){return delete this.attributes[t],this}hasAttribute(t){return this.attributes[t]!==void 0}addGroup(t,i,r=0){this.groups.push({start:t,count:i,materialIndex:r})}clearGroups(){this.groups=[]}setDrawRange(t,i){this.drawRange.start=t,this.drawRange.count=i}applyMatrix4(t){const i=this.attributes.position;i!==void 0&&(i.applyMatrix4(t),i.needsUpdate=!0);const r=this.attributes.normal;if(r!==void 0){const f=new ft().getNormalMatrix(t);r.applyNormalMatrix(f),r.needsUpdate=!0}const c=this.attributes.tangent;return c!==void 0&&(c.transformDirection(t),c.needsUpdate=!0),this.boundingBox!==null&&this.computeBoundingBox(),this.boundingSphere!==null&&this.computeBoundingSphere(),this._transformed=!0,this}applyQuaternion(t){return Ei.makeRotationFromQuaternion(t),this.applyMatrix4(Ei),this}rotateX(t){return Ei.makeRotationX(t),this.applyMatrix4(Ei),this}rotateY(t){return Ei.makeRotationY(t),this.applyMatrix4(Ei),this}rotateZ(t){return Ei.makeRotationZ(t),this.applyMatrix4(Ei),this}translate(t,i,r){return Ei.makeTranslation(t,i,r),this.applyMatrix4(Ei),this}scale(t,i,r){return Ei.makeScale(t,i,r),this.applyMatrix4(Ei),this}lookAt(t){return Ih.lookAt(t),Ih.updateMatrix(),this.applyMatrix4(Ih.matrix),this}center(){return this.computeBoundingBox(),this.boundingBox.getCenter(Kr).negate(),this.translate(Kr.x,Kr.y,Kr.z),this}setFromPoints(t){const i=this.getAttribute("position");if(i===void 0){const r=[];for(let c=0,f=t.length;c<f;c++){const d=t[c];r.push(d.x,d.y,d.z||0)}this.setAttribute("position",new Ti(r,3))}else{const r=Math.min(t.length,i.count);for(let c=0;c<r;c++){const f=t[c];i.setXYZ(c,f.x,f.y,f.z||0)}t.length>i.count&&ot("BufferGeometry: Buffer size too small for points data. Use .dispose() and create a new geometry."),i.needsUpdate=!0}return this}computeBoundingBox(){this.boundingBox===null&&(this.boundingBox=new Tl);const t=this.attributes.position,i=this.morphAttributes.position;if(t&&t.isGLBufferAttribute){It("BufferGeometry.computeBoundingBox(): GLBufferAttribute requires a manual bounding box.",this),this.boundingBox.set(new he(-1/0,-1/0,-1/0),new he(1/0,1/0,1/0));return}if(t!==void 0){if(this.boundingBox.setFromBufferAttribute(t),i)for(let r=0,c=i.length;r<c;r++){const f=i[r];ci.setFromBufferAttribute(f),this.morphTargetsRelative?(Tn.addVectors(this.boundingBox.min,ci.min),this.boundingBox.expandByPoint(Tn),Tn.addVectors(this.boundingBox.max,ci.max),this.boundingBox.expandByPoint(Tn)):(this.boundingBox.expandByPoint(ci.min),this.boundingBox.expandByPoint(ci.max))}}else this.boundingBox.makeEmpty();(isNaN(this.boundingBox.min.x)||isNaN(this.boundingBox.min.y)||isNaN(this.boundingBox.min.z))&&It('BufferGeometry.computeBoundingBox(): Computed min/max have NaN values. The "position" attribute is likely to have NaN values.',this)}computeBoundingSphere(){this.boundingSphere===null&&(this.boundingSphere=new Kp);const t=this.attributes.position,i=this.morphAttributes.position;if(t&&t.isGLBufferAttribute){It("BufferGeometry.computeBoundingSphere(): GLBufferAttribute requires a manual bounding sphere.",this),this.boundingSphere.set(new he,1/0);return}if(t){const r=this.boundingSphere.center;if(ci.setFromBufferAttribute(t),i)for(let f=0,d=i.length;f<d;f++){const h=i[f];dl.setFromBufferAttribute(h),this.morphTargetsRelative?(Tn.addVectors(ci.min,dl.min),ci.expandByPoint(Tn),Tn.addVectors(ci.max,dl.max),ci.expandByPoint(Tn)):(ci.expandByPoint(dl.min),ci.expandByPoint(dl.max))}ci.getCenter(r);let c=0;for(let f=0,d=t.count;f<d;f++)Tn.fromBufferAttribute(t,f),c=Math.max(c,r.distanceToSquared(Tn));if(i)for(let f=0,d=i.length;f<d;f++){const h=i[f],m=this.morphTargetsRelative;for(let g=0,x=h.count;g<x;g++)Tn.fromBufferAttribute(h,g),m&&(Kr.fromBufferAttribute(t,g),Tn.add(Kr)),c=Math.max(c,r.distanceToSquared(Tn))}this.boundingSphere.radius=Math.sqrt(c),isNaN(this.boundingSphere.radius)&&It('BufferGeometry.computeBoundingSphere(): Computed radius is NaN. The "position" attribute is likely to have NaN values.',this)}}computeTangents(){const t=this.index,i=this.attributes;if(t===null||i.position===void 0||i.normal===void 0||i.uv===void 0){It("BufferGeometry: .computeTangents() failed. Missing required attributes (index, position, normal or uv)");return}const r=i.position,c=i.normal,f=i.uv;let d=this.getAttribute("tangent");(d===void 0||d.count!==r.count)&&(d=new Pa(new Float32Array(4*r.count),4),this.setAttribute("tangent",d));const h=[],m=[];for(let b=0;b<r.count;b++)h[b]=new he,m[b]=new he;const g=new he,x=new he,_=new he,v=new Ut,E=new Ut,R=new Ut,N=new he,y=new he;function S(b,L,k){g.fromBufferAttribute(r,b),x.fromBufferAttribute(r,L),_.fromBufferAttribute(r,k),v.fromBufferAttribute(f,b),E.fromBufferAttribute(f,L),R.fromBufferAttribute(f,k),x.sub(g),_.sub(g),E.sub(v),R.sub(v);const Y=1/(E.x*R.y-R.x*E.y);isFinite(Y)&&(N.copy(x).multiplyScalar(R.y).addScaledVector(_,-E.y).multiplyScalar(Y),y.copy(_).multiplyScalar(E.x).addScaledVector(x,-R.x).multiplyScalar(Y),h[b].add(N),h[L].add(N),h[k].add(N),m[b].add(y),m[L].add(y),m[k].add(y))}let z=this.groups;z.length===0&&(z=[{start:0,count:t.count}]);for(let b=0,L=z.length;b<L;++b){const k=z[b],Y=k.start,$=k.count;for(let O=Y,q=Y+$;O<q;O+=3)S(t.getX(O+0),t.getX(O+1),t.getX(O+2))}const W=new he,C=new he,D=new he,U=new he;function I(b){D.fromBufferAttribute(c,b),U.copy(D);const L=h[b];W.copy(L),W.sub(D.multiplyScalar(D.dot(L))).normalize(),C.crossVectors(U,L);const Y=C.dot(m[b])<0?-1:1;d.setXYZW(b,W.x,W.y,W.z,Y)}for(let b=0,L=z.length;b<L;++b){const k=z[b],Y=k.start,$=k.count;for(let O=Y,q=Y+$;O<q;O+=3)I(t.getX(O+0)),I(t.getX(O+1)),I(t.getX(O+2))}this._transformed=!0}computeVertexNormals(){const t=this.index,i=this.getAttribute("position");if(i!==void 0){let r=this.getAttribute("normal");if(r===void 0||r.count!==i.count)r=new Pa(new Float32Array(i.count*3),3),this.setAttribute("normal",r);else for(let v=0,E=r.count;v<E;v++)r.setXYZ(v,0,0,0);const c=new he,f=new he,d=new he,h=new he,m=new he,g=new he,x=new he,_=new he;if(t)for(let v=0,E=t.count;v<E;v+=3){const R=t.getX(v+0),N=t.getX(v+1),y=t.getX(v+2);c.fromBufferAttribute(i,R),f.fromBufferAttribute(i,N),d.fromBufferAttribute(i,y),x.subVectors(d,f),_.subVectors(c,f),x.cross(_),h.fromBufferAttribute(r,R),m.fromBufferAttribute(r,N),g.fromBufferAttribute(r,y),h.add(x),m.add(x),g.add(x),r.setXYZ(R,h.x,h.y,h.z),r.setXYZ(N,m.x,m.y,m.z),r.setXYZ(y,g.x,g.y,g.z)}else for(let v=0,E=i.count;v<E;v+=3)c.fromBufferAttribute(i,v+0),f.fromBufferAttribute(i,v+1),d.fromBufferAttribute(i,v+2),x.subVectors(d,f),_.subVectors(c,f),x.cross(_),r.setXYZ(v+0,x.x,x.y,x.z),r.setXYZ(v+1,x.x,x.y,x.z),r.setXYZ(v+2,x.x,x.y,x.z);this.normalizeNormals(),r.needsUpdate=!0}}normalizeNormals(){const t=this.attributes.normal;for(let i=0,r=t.count;i<r;i++)Tn.fromBufferAttribute(t,i),Tn.normalize(),t.setXYZ(i,Tn.x,Tn.y,Tn.z)}toNonIndexed(){function t(h,m){const g=h.array,x=h.itemSize,_=h.normalized,v=new g.constructor(m.length*x);let E=0,R=0;for(let N=0,y=m.length;N<y;N++){h.isInterleavedBufferAttribute?E=m[N]*h.data.stride+h.offset:E=m[N]*x;for(let S=0;S<x;S++)v[R++]=g[E++]}return new Pa(v,x,_)}if(this.index===null)return ot("BufferGeometry.toNonIndexed(): BufferGeometry is already non-indexed."),this;const i=new ua,r=this.index.array,c=this.attributes;for(const h in c){const m=c[h],g=t(m,r);i.setAttribute(h,g)}const f=this.morphAttributes;for(const h in f){const m=[],g=f[h];for(let x=0,_=g.length;x<_;x++){const v=g[x],E=t(v,r);m.push(E)}i.morphAttributes[h]=m}i.morphTargetsRelative=this.morphTargetsRelative;const d=this.groups;for(let h=0,m=d.length;h<m;h++){const g=d[h];i.addGroup(g.start,g.count,g.materialIndex)}return i}toJSON(){const t={metadata:{version:4.7,type:"BufferGeometry",generator:"BufferGeometry.toJSON"}};if(t.uuid=this.uuid,t.type=this.parameters!==void 0&&this._transformed===!0?"BufferGeometry":this.type,t.name=this.name,Object.keys(this.userData).length>0&&(t.userData=this.userData),this.parameters!==void 0&&this._transformed!==!0){const m=this.parameters;for(const g in m)m[g]!==void 0&&(t[g]=m[g]);return t}t.data={attributes:{}};const i=this.index;i!==null&&(t.data.index={type:i.array.constructor.name,array:Array.prototype.slice.call(i.array)});const r=this.attributes;for(const m in r){const g=r[m];t.data.attributes[m]=g.toJSON(t.data)}const c={};let f=!1;for(const m in this.morphAttributes){const g=this.morphAttributes[m],x=[];for(let _=0,v=g.length;_<v;_++){const E=g[_];x.push(E.toJSON(t.data))}x.length>0&&(c[m]=x,f=!0)}f&&(t.data.morphAttributes=c,t.data.morphTargetsRelative=this.morphTargetsRelative);const d=this.groups;d.length>0&&(t.data.groups=JSON.parse(JSON.stringify(d)));const h=this.boundingSphere;return h!==null&&(t.data.boundingSphere=h.toJSON()),t}clone(){return new this.constructor().copy(this)}copy(t){this.index=null,this.attributes={},this.morphAttributes={},this.groups=[],this.boundingBox=null,this.boundingSphere=null;const i={};this.name=t.name;const r=t.index;r!==null&&this.setIndex(r.clone());const c=t.attributes;for(const g in c){const x=c[g];this.setAttribute(g,x.clone(i))}const f=t.morphAttributes;for(const g in f){const x=[],_=f[g];for(let v=0,E=_.length;v<E;v++)x.push(_[v].clone(i));this.morphAttributes[g]=x}this.morphTargetsRelative=t.morphTargetsRelative;const d=t.groups;for(let g=0,x=d.length;g<x;g++){const _=d[g];this.addGroup(_.start,_.count,_.materialIndex)}const h=t.boundingBox;h!==null&&(this.boundingBox=h.clone());const m=t.boundingSphere;return m!==null&&(this.boundingSphere=m.clone()),this.drawRange.start=t.drawRange.start,this.drawRange.count=t.drawRange.count,this.userData=t.userData,this._transformed=t._transformed,this}dispose(){this.dispatchEvent({type:"dispose"})}}const Bh=new he,tb=new he,nb=new ft;class _s{constructor(t=new he(1,0,0),i=0){this.isPlane=!0,this.normal=t,this.constant=i}set(t,i){return this.normal.copy(t),this.constant=i,this}setComponents(t,i,r,c){return this.normal.set(t,i,r),this.constant=c,this}setFromNormalAndCoplanarPoint(t,i){return this.normal.copy(t),this.constant=-i.dot(this.normal),this}setFromCoplanarPoints(t,i,r){const c=Bh.subVectors(r,i).cross(tb.subVectors(t,i)).normalize();return this.setFromNormalAndCoplanarPoint(c,t),this}copy(t){return this.normal.copy(t.normal),this.constant=t.constant,this}normalize(){const t=1/this.normal.length();return this.normal.multiplyScalar(t),this.constant*=t,this}negate(){return this.constant*=-1,this.normal.negate(),this}distanceToPoint(t){return this.normal.dot(t)+this.constant}distanceToSphere(t){return this.distanceToPoint(t.center)-t.radius}projectPoint(t,i){return i.copy(t).addScaledVector(this.normal,-this.distanceToPoint(t))}intersectLine(t,i,r=!0){const c=t.delta(Bh),f=this.normal.dot(c);if(f===0)return this.distanceToPoint(t.start)===0?i.copy(t.start):null;const d=-(t.start.dot(this.normal)+this.constant)/f;return r===!0&&(d<0||d>1)?null:i.copy(t.start).addScaledVector(c,d)}intersectsLine(t){const i=this.distanceToPoint(t.start),r=this.distanceToPoint(t.end);return i<0&&r>0||r<0&&i>0}intersectsBox(t){return t.intersectsPlane(this)}intersectsSphere(t){return t.intersectsPlane(this)}coplanarPoint(t){return t.copy(this.normal).multiplyScalar(-this.constant)}applyMatrix4(t,i){const r=i||nb.getNormalMatrix(t),c=this.coplanarPoint(Bh).applyMatrix4(t),f=this.normal.applyMatrix3(r).normalize();return this.constant=-c.dot(f),this}translate(t){return this.constant-=t.dot(this.normal),this}equals(t){return t.normal.equals(this.normal)&&t.constant===this.constant}clone(){return new this.constructor().copy(this)}toJSON(){return{normal:this.normal.toArray(),constant:this.constant}}fromJSON(t){return this.normal.fromArray(t.normal),this.constant=t.constant,this}}let ib=0;class Al extends $s{constructor(){super(),this.isMaterial=!0,Object.defineProperty(this,"id",{value:ib++}),this.uuid=bl(),this.name="",this.type="Material",this.blending=vl,this.side=Ks,this.vertexColors=!1,this.opacity=1,this.transparent=!1,this.alphaHash=!1,this.blendSrc=xx,this.blendDst=Sx,this.blendEquation=eo,this.blendSrcAlpha=null,this.blendDstAlpha=null,this.blendEquationAlpha=null,this.blendColor=new Dt(0,0,0),this.blendAlpha=0,this.depthFunc=xl,this.depthTest=!0,this.depthWrite=!0,this.stencilWriteMask=255,this.stencilFunc=TE,this.stencilRef=0,this.stencilFuncMask=255,this.stencilFail=_h,this.stencilZFail=_h,this.stencilZPass=_h,this.stencilWrite=!1,this.clippingPlanes=null,this.clipIntersection=!1,this.clipShadows=!1,this.shadowSide=null,this.colorWrite=!0,this.precision=null,this.polygonOffset=!1,this.polygonOffsetFactor=0,this.polygonOffsetUnits=0,this.dithering=!1,this.alphaToCoverage=!1,this.premultipliedAlpha=!1,this.forceSinglePass=!1,this.allowOverride=!0,this.visible=!0,this.toneMapped=!0,this.userData={},this.version=0,this._alphaTest=0}get alphaTest(){return this._alphaTest}set alphaTest(t){this._alphaTest>0!=t>0&&this.version++,this._alphaTest=t}onBeforeRender(){}onBeforeCompile(){}customProgramCacheKey(){return this.onBeforeCompile.toString()}setValues(t){if(t!==void 0)for(const i in t){const r=t[i];if(r===void 0){ot(`Material: parameter '${i}' has value of undefined.`);continue}const c=this[i];if(c===void 0){ot(`Material: '${i}' is not a property of THREE.${this.type}.`);continue}c&&c.isColor?c.set(r):c&&c.isVector2&&r&&r.isVector2||c&&c.isEuler&&r&&r.isEuler||c&&c.isVector3&&r&&r.isVector3?c.copy(r):this[i]=r}}toJSON(t){const i=t===void 0||typeof t=="string";i&&(t={textures:{},images:{}});const r={metadata:{version:4.7,type:"Material",generator:"Material.toJSON"}};r.uuid=this.uuid,r.type=this.type,r.blending=this.blending,r.side=this.side,r.shadowSide=this.shadowSide,r.vertexColors=this.vertexColors,r.opacity=this.opacity,r.transparent=this.transparent,r.blendSrc=this.blendSrc,r.blendDst=this.blendDst,r.blendEquation=this.blendEquation,r.blendSrcAlpha=this.blendSrcAlpha,r.blendDstAlpha=this.blendDstAlpha,r.blendEquationAlpha=this.blendEquationAlpha,r.blendColor=this.blendColor.getHex(),r.blendAlpha=this.blendAlpha,r.depthFunc=this.depthFunc,r.depthTest=this.depthTest,r.depthWrite=this.depthWrite,r.colorWrite=this.colorWrite,r.clipIntersection=this.clipIntersection,r.clipShadows=this.clipShadows,r.stencilWriteMask=this.stencilWriteMask,r.stencilFunc=this.stencilFunc,r.stencilRef=this.stencilRef,r.stencilFuncMask=this.stencilFuncMask,r.stencilFail=this.stencilFail,r.stencilZFail=this.stencilZFail,r.stencilZPass=this.stencilZPass,r.stencilWrite=this.stencilWrite,r.polygonOffset=this.polygonOffset,r.polygonOffsetFactor=this.polygonOffsetFactor,r.polygonOffsetUnits=this.polygonOffsetUnits,r.dithering=this.dithering,r.alphaTest=this.alphaTest,r.alphaHash=this.alphaHash,r.alphaToCoverage=this.alphaToCoverage,r.premultipliedAlpha=this.premultipliedAlpha,r.forceSinglePass=this.forceSinglePass,r.allowOverride=this.allowOverride,r.visible=this.visible,r.toneMapped=this.toneMapped,r.name=this.name,this.color&&this.color.isColor&&(r.color=this.color.getHex()),this.roughness!==void 0&&(r.roughness=this.roughness),this.metalness!==void 0&&(r.metalness=this.metalness),this.sheen!==void 0&&(r.sheen=this.sheen),this.sheenColor&&this.sheenColor.isColor&&(r.sheenColor=this.sheenColor.getHex()),this.sheenRoughness!==void 0&&(r.sheenRoughness=this.sheenRoughness),this.emissive&&this.emissive.isColor&&(r.emissive=this.emissive.getHex()),this.emissiveIntensity!==void 0&&(r.emissiveIntensity=this.emissiveIntensity),this.specular&&this.specular.isColor&&(r.specular=this.specular.getHex()),this.specularIntensity!==void 0&&(r.specularIntensity=this.specularIntensity),this.specularColor&&this.specularColor.isColor&&(r.specularColor=this.specularColor.getHex()),this.shininess!==void 0&&(r.shininess=this.shininess),this.clearcoat!==void 0&&(r.clearcoat=this.clearcoat),this.clearcoatRoughness!==void 0&&(r.clearcoatRoughness=this.clearcoatRoughness),this.clearcoatMap&&this.clearcoatMap.isTexture&&(r.clearcoatMap=this.clearcoatMap.toJSON(t).uuid),this.clearcoatRoughnessMap&&this.clearcoatRoughnessMap.isTexture&&(r.clearcoatRoughnessMap=this.clearcoatRoughnessMap.toJSON(t).uuid),this.clearcoatNormalMap&&this.clearcoatNormalMap.isTexture&&(r.clearcoatNormalMap=this.clearcoatNormalMap.toJSON(t).uuid,r.clearcoatNormalScale=this.clearcoatNormalScale.toArray()),this.sheenColorMap&&this.sheenColorMap.isTexture&&(r.sheenColorMap=this.sheenColorMap.toJSON(t).uuid),this.sheenRoughnessMap&&this.sheenRoughnessMap.isTexture&&(r.sheenRoughnessMap=this.sheenRoughnessMap.toJSON(t).uuid),this.dispersion!==void 0&&(r.dispersion=this.dispersion),this.retroreflectivity!==void 0&&(r.retroreflectivity=this.retroreflectivity),this.iridescence!==void 0&&(r.iridescence=this.iridescence),this.iridescenceIOR!==void 0&&(r.iridescenceIOR=this.iridescenceIOR),this.iridescenceThicknessRange!==void 0&&(r.iridescenceThicknessRange=this.iridescenceThicknessRange),this.iridescenceMap&&this.iridescenceMap.isTexture&&(r.iridescenceMap=this.iridescenceMap.toJSON(t).uuid),this.iridescenceThicknessMap&&this.iridescenceThicknessMap.isTexture&&(r.iridescenceThicknessMap=this.iridescenceThicknessMap.toJSON(t).uuid),this.anisotropy!==void 0&&(r.anisotropy=this.anisotropy),this.anisotropyRotation!==void 0&&(r.anisotropyRotation=this.anisotropyRotation),this.anisotropyMap&&this.anisotropyMap.isTexture&&(r.anisotropyMap=this.anisotropyMap.toJSON(t).uuid),this.map&&this.map.isTexture&&(r.map=this.map.toJSON(t).uuid),this.matcap&&this.matcap.isTexture&&(r.matcap=this.matcap.toJSON(t).uuid),this.alphaMap&&this.alphaMap.isTexture&&(r.alphaMap=this.alphaMap.toJSON(t).uuid),this.lightMap&&this.lightMap.isTexture&&(r.lightMap=this.lightMap.toJSON(t).uuid,r.lightMapIntensity=this.lightMapIntensity),this.aoMap&&this.aoMap.isTexture&&(r.aoMap=this.aoMap.toJSON(t).uuid,r.aoMapIntensity=this.aoMapIntensity),this.bumpMap&&this.bumpMap.isTexture&&(r.bumpMap=this.bumpMap.toJSON(t).uuid,r.bumpScale=this.bumpScale),this.normalMap&&this.normalMap.isTexture&&(r.normalMap=this.normalMap.toJSON(t).uuid,r.normalMapType=this.normalMapType,r.normalScale=this.normalScale.toArray()),this.displacementMap&&this.displacementMap.isTexture&&(r.displacementMap=this.displacementMap.toJSON(t).uuid,r.displacementScale=this.displacementScale,r.displacementBias=this.displacementBias),this.roughnessMap&&this.roughnessMap.isTexture&&(r.roughnessMap=this.roughnessMap.toJSON(t).uuid),this.metalnessMap&&this.metalnessMap.isTexture&&(r.metalnessMap=this.metalnessMap.toJSON(t).uuid),this.emissiveMap&&this.emissiveMap.isTexture&&(r.emissiveMap=this.emissiveMap.toJSON(t).uuid),this.specularMap&&this.specularMap.isTexture&&(r.specularMap=this.specularMap.toJSON(t).uuid),this.specularIntensityMap&&this.specularIntensityMap.isTexture&&(r.specularIntensityMap=this.specularIntensityMap.toJSON(t).uuid),this.specularColorMap&&this.specularColorMap.isTexture&&(r.specularColorMap=this.specularColorMap.toJSON(t).uuid),this.envMap&&this.envMap.isTexture&&(r.envMap=this.envMap.toJSON(t).uuid,this.combine!==void 0&&(r.combine=this.combine)),this.envMapRotation!==void 0&&(r.envMapRotation=this.envMapRotation.toArray()),this.envMapIntensity!==void 0&&(r.envMapIntensity=this.envMapIntensity),this.reflectivity!==void 0&&(r.reflectivity=this.reflectivity),this.refractionRatio!==void 0&&(r.refractionRatio=this.refractionRatio),this.gradientMap&&this.gradientMap.isTexture&&(r.gradientMap=this.gradientMap.toJSON(t).uuid),this.transmission!==void 0&&(r.transmission=this.transmission),this.transmissionMap&&this.transmissionMap.isTexture&&(r.transmissionMap=this.transmissionMap.toJSON(t).uuid),this.thickness!==void 0&&(r.thickness=this.thickness),this.thicknessMap&&this.thicknessMap.isTexture&&(r.thicknessMap=this.thicknessMap.toJSON(t).uuid),this.attenuationDistance!==void 0&&(r.attenuationDistance=this.attenuationDistance),this.attenuationColor!==void 0&&(r.attenuationColor=this.attenuationColor.getHex()),this.size!==void 0&&(r.size=this.size),this.sizeAttenuation!==void 0&&(r.sizeAttenuation=this.sizeAttenuation),Array.isArray(this.clippingPlanes)&&this.clippingPlanes.length>0&&(r.clippingPlanes=this.clippingPlanes.map(f=>f.toJSON())),this.rotation!==void 0&&(r.rotation=this.rotation),this.depthPacking!==void 0&&(r.depthPacking=this.depthPacking),this.linewidth!==void 0&&(r.linewidth=this.linewidth),this.linecap!==void 0&&(r.linecap=this.linecap),this.linejoin!==void 0&&(r.linejoin=this.linejoin),this.dashSize!==void 0&&(r.dashSize=this.dashSize),this.gapSize!==void 0&&(r.gapSize=this.gapSize),this.scale!==void 0&&(r.scale=this.scale),this.wireframe!==void 0&&(r.wireframe=this.wireframe),this.wireframeLinewidth!==void 0&&(r.wireframeLinewidth=this.wireframeLinewidth),this.wireframeLinecap!==void 0&&(r.wireframeLinecap=this.wireframeLinecap),this.wireframeLinejoin!==void 0&&(r.wireframeLinejoin=this.wireframeLinejoin),this.flatShading!==void 0&&(r.flatShading=this.flatShading),this.fog!==void 0&&(r.fog=this.fog),Object.keys(this.userData).length>0&&(r.userData=this.userData);function c(f){const d=[];for(const h in f){const m=f[h];delete m.metadata,d.push(m)}return d}if(i){const f=c(t.textures),d=c(t.images);f.length>0&&(r.textures=f),d.length>0&&(r.images=d)}return r}fromJSON(t,i){if(t.uuid!==void 0&&(this.uuid=t.uuid),t.name!==void 0&&(this.name=t.name),t.color!==void 0&&this.color!==void 0&&this.color.setHex(t.color),t.roughness!==void 0&&(this.roughness=t.roughness),t.metalness!==void 0&&(this.metalness=t.metalness),t.sheen!==void 0&&(this.sheen=t.sheen),t.sheenColor!==void 0&&(this.sheenColor=new Dt().setHex(t.sheenColor)),t.sheenRoughness!==void 0&&(this.sheenRoughness=t.sheenRoughness),t.emissive!==void 0&&this.emissive!==void 0&&this.emissive.setHex(t.emissive),t.specular!==void 0&&this.specular!==void 0&&this.specular.setHex(t.specular),t.specularIntensity!==void 0&&(this.specularIntensity=t.specularIntensity),t.specularColor!==void 0&&this.specularColor!==void 0&&this.specularColor.setHex(t.specularColor),t.shininess!==void 0&&(this.shininess=t.shininess),t.clearcoat!==void 0&&(this.clearcoat=t.clearcoat),t.clearcoatRoughness!==void 0&&(this.clearcoatRoughness=t.clearcoatRoughness),t.dispersion!==void 0&&(this.dispersion=t.dispersion),t.retroreflectivity!==void 0&&(this.retroreflectivity=t.retroreflectivity),t.iridescence!==void 0&&(this.iridescence=t.iridescence),t.iridescenceIOR!==void 0&&(this.iridescenceIOR=t.iridescenceIOR),t.iridescenceThicknessRange!==void 0&&(this.iridescenceThicknessRange=t.iridescenceThicknessRange),t.transmission!==void 0&&(this.transmission=t.transmission),t.thickness!==void 0&&(this.thickness=t.thickness),t.attenuationDistance!==void 0&&(this.attenuationDistance=t.attenuationDistance),t.attenuationColor!==void 0&&this.attenuationColor!==void 0&&this.attenuationColor.setHex(t.attenuationColor),t.anisotropy!==void 0&&(this.anisotropy=t.anisotropy),t.anisotropyRotation!==void 0&&(this.anisotropyRotation=t.anisotropyRotation),t.fog!==void 0&&(this.fog=t.fog),t.flatShading!==void 0&&(this.flatShading=t.flatShading),t.blending!==void 0&&(this.blending=t.blending),t.combine!==void 0&&(this.combine=t.combine),t.side!==void 0&&(this.side=t.side),t.shadowSide!==void 0&&(this.shadowSide=t.shadowSide),t.opacity!==void 0&&(this.opacity=t.opacity),t.transparent!==void 0&&(this.transparent=t.transparent),t.alphaTest!==void 0&&(this.alphaTest=t.alphaTest),t.alphaHash!==void 0&&(this.alphaHash=t.alphaHash),t.depthFunc!==void 0&&(this.depthFunc=t.depthFunc),t.depthTest!==void 0&&(this.depthTest=t.depthTest),t.depthWrite!==void 0&&(this.depthWrite=t.depthWrite),t.colorWrite!==void 0&&(this.colorWrite=t.colorWrite),t.clippingPlanes!==void 0&&(this.clippingPlanes=t.clippingPlanes.map(r=>new _s().fromJSON(r))),t.clipIntersection!==void 0&&(this.clipIntersection=t.clipIntersection),t.clipShadows!==void 0&&(this.clipShadows=t.clipShadows),t.depthPacking!==void 0&&(this.depthPacking=t.depthPacking),t.blendSrc!==void 0&&(this.blendSrc=t.blendSrc),t.blendDst!==void 0&&(this.blendDst=t.blendDst),t.blendEquation!==void 0&&(this.blendEquation=t.blendEquation),t.blendSrcAlpha!==void 0&&(this.blendSrcAlpha=t.blendSrcAlpha),t.blendDstAlpha!==void 0&&(this.blendDstAlpha=t.blendDstAlpha),t.blendEquationAlpha!==void 0&&(this.blendEquationAlpha=t.blendEquationAlpha),t.blendColor!==void 0&&this.blendColor!==void 0&&this.blendColor.setHex(t.blendColor),t.blendAlpha!==void 0&&(this.blendAlpha=t.blendAlpha),t.stencilWriteMask!==void 0&&(this.stencilWriteMask=t.stencilWriteMask),t.stencilFunc!==void 0&&(this.stencilFunc=t.stencilFunc),t.stencilRef!==void 0&&(this.stencilRef=t.stencilRef),t.stencilFuncMask!==void 0&&(this.stencilFuncMask=t.stencilFuncMask),t.stencilFail!==void 0&&(this.stencilFail=t.stencilFail),t.stencilZFail!==void 0&&(this.stencilZFail=t.stencilZFail),t.stencilZPass!==void 0&&(this.stencilZPass=t.stencilZPass),t.stencilWrite!==void 0&&(this.stencilWrite=t.stencilWrite),t.wireframe!==void 0&&(this.wireframe=t.wireframe),t.wireframeLinewidth!==void 0&&(this.wireframeLinewidth=t.wireframeLinewidth),t.wireframeLinecap!==void 0&&(this.wireframeLinecap=t.wireframeLinecap),t.wireframeLinejoin!==void 0&&(this.wireframeLinejoin=t.wireframeLinejoin),t.rotation!==void 0&&(this.rotation=t.rotation),t.linewidth!==void 0&&(this.linewidth=t.linewidth),t.linecap!==void 0&&(this.linecap=t.linecap),t.linejoin!==void 0&&(this.linejoin=t.linejoin),t.dashSize!==void 0&&(this.dashSize=t.dashSize),t.gapSize!==void 0&&(this.gapSize=t.gapSize),t.scale!==void 0&&(this.scale=t.scale),t.polygonOffset!==void 0&&(this.polygonOffset=t.polygonOffset),t.polygonOffsetFactor!==void 0&&(this.polygonOffsetFactor=t.polygonOffsetFactor),t.polygonOffsetUnits!==void 0&&(this.polygonOffsetUnits=t.polygonOffsetUnits),t.dithering!==void 0&&(this.dithering=t.dithering),t.alphaToCoverage!==void 0&&(this.alphaToCoverage=t.alphaToCoverage),t.premultipliedAlpha!==void 0&&(this.premultipliedAlpha=t.premultipliedAlpha),t.forceSinglePass!==void 0&&(this.forceSinglePass=t.forceSinglePass),t.allowOverride!==void 0&&(this.allowOverride=t.allowOverride),t.visible!==void 0&&(this.visible=t.visible),t.toneMapped!==void 0&&(this.toneMapped=t.toneMapped),t.userData!==void 0&&(this.userData=t.userData),t.vertexColors!==void 0&&(typeof t.vertexColors=="number"?this.vertexColors=t.vertexColors>0:this.vertexColors=t.vertexColors),t.size!==void 0&&(this.size=t.size),t.sizeAttenuation!==void 0&&(this.sizeAttenuation=t.sizeAttenuation),t.map!==void 0&&(this.map=i[t.map]||null),t.matcap!==void 0&&(this.matcap=i[t.matcap]||null),t.alphaMap!==void 0&&(this.alphaMap=i[t.alphaMap]||null),t.bumpMap!==void 0&&(this.bumpMap=i[t.bumpMap]||null),t.bumpScale!==void 0&&(this.bumpScale=t.bumpScale),t.normalMap!==void 0&&(this.normalMap=i[t.normalMap]||null),t.normalMapType!==void 0&&(this.normalMapType=t.normalMapType),t.normalScale!==void 0){let r=t.normalScale;Array.isArray(r)===!1&&(r=[r,r]),this.normalScale=new Ut().fromArray(r)}return t.displacementMap!==void 0&&(this.displacementMap=i[t.displacementMap]||null),t.displacementScale!==void 0&&(this.displacementScale=t.displacementScale),t.displacementBias!==void 0&&(this.displacementBias=t.displacementBias),t.roughnessMap!==void 0&&(this.roughnessMap=i[t.roughnessMap]||null),t.metalnessMap!==void 0&&(this.metalnessMap=i[t.metalnessMap]||null),t.emissiveMap!==void 0&&(this.emissiveMap=i[t.emissiveMap]||null),t.emissiveIntensity!==void 0&&(this.emissiveIntensity=t.emissiveIntensity),t.specularMap!==void 0&&(this.specularMap=i[t.specularMap]||null),t.specularIntensityMap!==void 0&&(this.specularIntensityMap=i[t.specularIntensityMap]||null),t.specularColorMap!==void 0&&(this.specularColorMap=i[t.specularColorMap]||null),t.envMap!==void 0&&(this.envMap=i[t.envMap]||null),t.envMapRotation!==void 0&&this.envMapRotation.fromArray(t.envMapRotation),t.envMapIntensity!==void 0&&(this.envMapIntensity=t.envMapIntensity),t.reflectivity!==void 0&&(this.reflectivity=t.reflectivity),t.refractionRatio!==void 0&&(this.refractionRatio=t.refractionRatio),t.lightMap!==void 0&&(this.lightMap=i[t.lightMap]||null),t.lightMapIntensity!==void 0&&(this.lightMapIntensity=t.lightMapIntensity),t.aoMap!==void 0&&(this.aoMap=i[t.aoMap]||null),t.aoMapIntensity!==void 0&&(this.aoMapIntensity=t.aoMapIntensity),t.gradientMap!==void 0&&(this.gradientMap=i[t.gradientMap]||null),t.clearcoatMap!==void 0&&(this.clearcoatMap=i[t.clearcoatMap]||null),t.clearcoatRoughnessMap!==void 0&&(this.clearcoatRoughnessMap=i[t.clearcoatRoughnessMap]||null),t.clearcoatNormalMap!==void 0&&(this.clearcoatNormalMap=i[t.clearcoatNormalMap]||null),t.clearcoatNormalScale!==void 0&&(this.clearcoatNormalScale=new Ut().fromArray(t.clearcoatNormalScale)),t.iridescenceMap!==void 0&&(this.iridescenceMap=i[t.iridescenceMap]||null),t.iridescenceThicknessMap!==void 0&&(this.iridescenceThicknessMap=i[t.iridescenceThicknessMap]||null),t.transmissionMap!==void 0&&(this.transmissionMap=i[t.transmissionMap]||null),t.thicknessMap!==void 0&&(this.thicknessMap=i[t.thicknessMap]||null),t.anisotropyMap!==void 0&&(this.anisotropyMap=i[t.anisotropyMap]||null),t.sheenColorMap!==void 0&&(this.sheenColorMap=i[t.sheenColorMap]||null),t.sheenRoughnessMap!==void 0&&(this.sheenRoughnessMap=i[t.sheenRoughnessMap]||null),this}clone(){return new this.constructor().copy(this)}copy(t){this.name=t.name,this.blending=t.blending,this.side=t.side,this.vertexColors=t.vertexColors,this.opacity=t.opacity,this.transparent=t.transparent,this.blendSrc=t.blendSrc,this.blendDst=t.blendDst,this.blendEquation=t.blendEquation,this.blendSrcAlpha=t.blendSrcAlpha,this.blendDstAlpha=t.blendDstAlpha,this.blendEquationAlpha=t.blendEquationAlpha,this.blendColor.copy(t.blendColor),this.blendAlpha=t.blendAlpha,this.depthFunc=t.depthFunc,this.depthTest=t.depthTest,this.depthWrite=t.depthWrite,this.stencilWriteMask=t.stencilWriteMask,this.stencilFunc=t.stencilFunc,this.stencilRef=t.stencilRef,this.stencilFuncMask=t.stencilFuncMask,this.stencilFail=t.stencilFail,this.stencilZFail=t.stencilZFail,this.stencilZPass=t.stencilZPass,this.stencilWrite=t.stencilWrite;const i=t.clippingPlanes;let r=null;if(i!==null){const c=i.length;r=new Array(c);for(let f=0;f!==c;++f)r[f]=i[f].clone()}return this.clippingPlanes=r,this.clipIntersection=t.clipIntersection,this.clipShadows=t.clipShadows,this.shadowSide=t.shadowSide,this.colorWrite=t.colorWrite,this.precision=t.precision,this.polygonOffset=t.polygonOffset,this.polygonOffsetFactor=t.polygonOffsetFactor,this.polygonOffsetUnits=t.polygonOffsetUnits,this.dithering=t.dithering,this.alphaTest=t.alphaTest,this.alphaHash=t.alphaHash,this.alphaToCoverage=t.alphaToCoverage,this.premultipliedAlpha=t.premultipliedAlpha,this.forceSinglePass=t.forceSinglePass,this.allowOverride=t.allowOverride,this.visible=t.visible,this.toneMapped=t.toneMapped,this.userData=JSON.parse(JSON.stringify(t.userData)),this}dispose(){this.dispatchEvent({type:"dispose"})}set needsUpdate(t){t===!0&&this.version++}}const Na=new he,zh=new he,au=new he,su=new he;class ab{constructor(t=new he,i=new he(0,0,-1)){this.origin=t,this.direction=i}set(t,i){return this.origin.copy(t),this.direction.copy(i),this}copy(t){return this.origin.copy(t.origin),this.direction.copy(t.direction),this}at(t,i){return i.copy(this.origin).addScaledVector(this.direction,t)}lookAt(t){return this.direction.copy(t).sub(this.origin).normalize(),this}recast(t){return this.origin.copy(this.at(t,Na)),this}closestPointToPoint(t,i){i.subVectors(t,this.origin);const r=i.dot(this.direction);return r<0?i.copy(this.origin):i.copy(this.origin).addScaledVector(this.direction,r)}distanceToPoint(t){return Math.sqrt(this.distanceSqToPoint(t))}distanceSqToPoint(t){const i=Na.subVectors(t,this.origin).dot(this.direction);return i<0?this.origin.distanceToSquared(t):(Na.copy(this.origin).addScaledVector(this.direction,i),Na.distanceToSquared(t))}distanceSqToSegment(t,i,r,c){zh.copy(t).add(i).multiplyScalar(.5),au.copy(i).sub(t).normalize(),su.copy(this.origin).sub(zh);const f=t.distanceTo(i)*.5,d=-this.direction.dot(au),h=su.dot(this.direction),m=-su.dot(au),g=su.lengthSq(),x=Math.abs(1-d*d);let _,v,E,R;if(x>0)if(_=d*m-h,v=d*h-m,R=f*x,_>=0)if(v>=-R)if(v<=R){const N=1/x;_*=N,v*=N,E=_*(_+d*v+2*h)+v*(d*_+v+2*m)+g}else v=f,_=Math.max(0,-(d*v+h)),E=-_*_+v*(v+2*m)+g;else v=-f,_=Math.max(0,-(d*v+h)),E=-_*_+v*(v+2*m)+g;else v<=-R?(_=Math.max(0,-(-d*f+h)),v=_>0?-f:Math.min(Math.max(-f,-m),f),E=-_*_+v*(v+2*m)+g):v<=R?(_=0,v=Math.min(Math.max(-f,-m),f),E=v*(v+2*m)+g):(_=Math.max(0,-(d*f+h)),v=_>0?f:Math.min(Math.max(-f,-m),f),E=-_*_+v*(v+2*m)+g);else v=d>0?-f:f,_=Math.max(0,-(d*v+h)),E=-_*_+v*(v+2*m)+g;return r&&r.copy(this.origin).addScaledVector(this.direction,_),c&&c.copy(zh).addScaledVector(au,v),E}intersectSphere(t,i){if(t.radius<0)return null;Na.subVectors(t.center,this.origin);const r=Na.dot(this.direction),c=Na.dot(Na)-r*r,f=t.radius*t.radius;if(c>f)return null;const d=Math.sqrt(f-c),h=r-d,m=r+d;return m<0?null:h<0?this.at(m,i):this.at(h,i)}intersectsSphere(t){return t.radius<0?!1:this.distanceSqToPoint(t.center)<=t.radius*t.radius}distanceToPlane(t){const i=t.normal.dot(this.direction);if(i===0)return t.distanceToPoint(this.origin)===0?0:null;const r=-(this.origin.dot(t.normal)+t.constant)/i;return r>=0?r:null}intersectPlane(t,i){const r=this.distanceToPlane(t);return r===null?null:this.at(r,i)}intersectsPlane(t){const i=t.distanceToPoint(this.origin);return i===0||t.normal.dot(this.direction)*i<0}intersectBox(t,i){let r,c,f,d,h,m;const g=1/this.direction.x,x=1/this.direction.y,_=1/this.direction.z,v=this.origin;return g>=0?(r=(t.min.x-v.x)*g,c=(t.max.x-v.x)*g):(r=(t.max.x-v.x)*g,c=(t.min.x-v.x)*g),x>=0?(f=(t.min.y-v.y)*x,d=(t.max.y-v.y)*x):(f=(t.max.y-v.y)*x,d=(t.min.y-v.y)*x),r>d||f>c||((f>r||isNaN(r))&&(r=f),(d<c||isNaN(c))&&(c=d),_>=0?(h=(t.min.z-v.z)*_,m=(t.max.z-v.z)*_):(h=(t.max.z-v.z)*_,m=(t.min.z-v.z)*_),r>m||h>c)||((h>r||r!==r)&&(r=h),(m<c||c!==c)&&(c=m),c<0)?null:this.at(r>=0?r:c,i)}intersectsBox(t){return this.intersectBox(t,Na)!==null}intersectTriangle(t,i,r,c,f){const d=this.origin,h=this.direction,m=h.x,g=h.y,x=h.z,_=t.x-d.x,v=t.y-d.y,E=t.z-d.z,R=i.x-d.x,N=i.y-d.y,y=i.z-d.z,S=r.x-d.x,z=r.y-d.y,W=r.z-d.z,C=Math.abs(m),D=Math.abs(g),U=Math.abs(x);let I,b,L,k,Y,$,O,q,ne,j,Q,pe;if(C>=D&&C>=U?(L=m,$=_,ne=R,pe=S,m>=0?(I=g,b=x,k=v,Y=E,O=N,q=y,j=z,Q=W):(I=x,b=g,k=E,Y=v,O=y,q=N,j=W,Q=z)):D>=U?(L=g,$=v,ne=N,pe=z,g>=0?(I=x,b=m,k=E,Y=_,O=y,q=R,j=W,Q=S):(I=m,b=x,k=_,Y=E,O=R,q=y,j=S,Q=W)):(L=x,$=E,ne=y,pe=W,x>=0?(I=m,b=g,k=_,Y=v,O=R,q=N,j=S,Q=z):(I=g,b=m,k=v,Y=_,O=N,q=R,j=z,Q=S)),L===0)return null;const ce=I/L,me=b/L,be=1/L,Ze=k-ce*$,je=Y-me*$,H=O-ce*ne,F=q-me*ne,ge=j-ce*pe,X=Q-me*pe,ue=ge*F-X*H,Ae=Ze*X-je*ge,Ie=H*je-F*Ze;if(c){if(ue<0||Ae<0||Ie<0)return null}else if((ue<0||Ae<0||Ie<0)&&(ue>0||Ae>0||Ie>0))return null;const xe=ue+Ae+Ie;if(xe===0)return null;const we=be*(ue*$+Ae*ne+Ie*pe);return(xe>0?we<0:we>0)?null:this.at(we/xe,f)}applyMatrix4(t){return this.origin.applyMatrix4(t),this.direction.transformDirection(t),this}equals(t){return t.origin.equals(this.origin)&&t.direction.equals(this.direction)}clone(){return new this.constructor().copy(this)}}class kx extends Al{constructor(t){super(),this.isMeshBasicMaterial=!0,this.type="MeshBasicMaterial",this.color=new Dt(16777215),this.map=null,this.lightMap=null,this.lightMapIntensity=1,this.aoMap=null,this.aoMapIntensity=1,this.specularMap=null,this.alphaMap=null,this.envMap=null,this.envMapRotation=new xs,this.combine=yx,this.reflectivity=1,this.refractionRatio=.98,this.wireframe=!1,this.wireframeLinewidth=1,this.wireframeLinecap="round",this.wireframeLinejoin="round",this.fog=!0,this.setValues(t)}copy(t){return super.copy(t),this.color.copy(t.color),this.map=t.map,this.lightMap=t.lightMap,this.lightMapIntensity=t.lightMapIntensity,this.aoMap=t.aoMap,this.aoMapIntensity=t.aoMapIntensity,this.specularMap=t.specularMap,this.alphaMap=t.alphaMap,this.envMap=t.envMap,this.envMapRotation.copy(t.envMapRotation),this.combine=t.combine,this.reflectivity=t.reflectivity,this.refractionRatio=t.refractionRatio,this.wireframe=t.wireframe,this.wireframeLinewidth=t.wireframeLinewidth,this.wireframeLinecap=t.wireframeLinecap,this.wireframeLinejoin=t.wireframeLinejoin,this.fog=t.fog,this}}const Pv=new un,Ws=new ab,ru=new Kp,Iv=new he,ou=new he,lu=new he,cu=new he,Fh=new he,uu=new he,Bv=new he,fu=new he;class la extends Qn{constructor(t=new ua,i=new kx){super(),this.isMesh=!0,this.type="Mesh",this.geometry=t,this.material=i,this.morphTargetDictionary=void 0,this.morphTargetInfluences=void 0,this.count=1,this.updateMorphTargets()}copy(t,i){return super.copy(t,i),t.morphTargetInfluences!==void 0&&(this.morphTargetInfluences=t.morphTargetInfluences.slice()),t.morphTargetDictionary!==void 0&&(this.morphTargetDictionary=Object.assign({},t.morphTargetDictionary)),this.material=Array.isArray(t.material)?t.material.slice():t.material,this.geometry=t.geometry,this}updateMorphTargets(){const i=this.geometry.morphAttributes,r=Object.keys(i);if(r.length>0){const c=i[r[0]];if(c!==void 0){this.morphTargetInfluences=[],this.morphTargetDictionary={};for(let f=0,d=c.length;f<d;f++){const h=c[f].name||String(f);this.morphTargetInfluences.push(0),this.morphTargetDictionary[h]=f}}}}getVertexPosition(t,i){const r=this.geometry,c=r.attributes.position,f=r.morphAttributes.position,d=r.morphTargetsRelative;i.fromBufferAttribute(c,t);const h=this.morphTargetInfluences;if(f&&h){uu.set(0,0,0);for(let m=0,g=f.length;m<g;m++){const x=h[m],_=f[m];x!==0&&(Fh.fromBufferAttribute(_,t),d?uu.addScaledVector(Fh,x):uu.addScaledVector(Fh.sub(i),x))}i.add(uu)}return i}intersectsFrustum(t){return t.intersectsObject(this)}raycast(t,i){const r=this.geometry,c=this.material,f=this.matrixWorld;c!==void 0&&(r.boundingSphere===null&&r.computeBoundingSphere(),ru.copy(r.boundingSphere),ru.applyMatrix4(f),Ws.copy(t.ray).recast(t.near),!(ru.containsPoint(Ws.origin)===!1&&(Ws.intersectSphere(ru,Iv)===null||Ws.origin.distanceToSquared(Iv)>(t.far-t.near)**2))&&(Pv.copy(f).invert(),Ws.copy(t.ray).applyMatrix4(Pv),!(r.boundingBox!==null&&Ws.intersectsBox(r.boundingBox)===!1)&&this._computeIntersections(t,i,Ws)))}_computeIntersections(t,i,r){let c;const f=this.geometry,d=this.material,h=f.index,m=f.attributes.position,g=f.attributes.uv,x=f.attributes.uv1,_=f.attributes.normal,v=f.groups,E=f.drawRange;if(h!==null)if(Array.isArray(d))for(let R=0,N=v.length;R<N;R++){const y=v[R],S=d[y.materialIndex],z=Math.max(y.start,E.start),W=Math.min(h.count,Math.min(y.start+y.count,E.start+E.count));for(let C=z,D=W;C<D;C+=3){const U=h.getX(C),I=h.getX(C+1),b=h.getX(C+2);c=du(this,S,t,r,g,x,_,U,I,b),c&&(c.faceIndex=Math.floor(C/3),c.face.materialIndex=y.materialIndex,i.push(c))}}else{const R=Math.max(0,E.start),N=Math.min(h.count,E.start+E.count);for(let y=R,S=N;y<S;y+=3){const z=h.getX(y),W=h.getX(y+1),C=h.getX(y+2);c=du(this,d,t,r,g,x,_,z,W,C),c&&(c.faceIndex=Math.floor(y/3),i.push(c))}}else if(m!==void 0)if(Array.isArray(d))for(let R=0,N=v.length;R<N;R++){const y=v[R],S=d[y.materialIndex],z=Math.max(y.start,E.start),W=Math.min(m.count,Math.min(y.start+y.count,E.start+E.count));for(let C=z,D=W;C<D;C+=3){const U=C,I=C+1,b=C+2;c=du(this,S,t,r,g,x,_,U,I,b),c&&(c.faceIndex=Math.floor(C/3),c.face.materialIndex=y.materialIndex,i.push(c))}}else{const R=Math.max(0,E.start),N=Math.min(m.count,E.start+E.count);for(let y=R,S=N;y<S;y+=3){const z=y,W=y+1,C=y+2;c=du(this,d,t,r,g,x,_,z,W,C),c&&(c.faceIndex=Math.floor(y/3),i.push(c))}}}}function sb(o,t,i,r,c,f,d,h){let m;if(t.side===Kn?m=r.intersectTriangle(d,f,c,!0,h):m=r.intersectTriangle(c,f,d,t.side===Ks,h),m===null)return null;fu.copy(h),fu.applyMatrix4(o.matrixWorld);const g=i.ray.origin.distanceTo(fu);return g<i.near||g>i.far?null:{distance:g,point:fu.clone(),object:o}}function du(o,t,i,r,c,f,d,h,m,g){o.getVertexPosition(h,ou),o.getVertexPosition(m,lu),o.getVertexPosition(g,cu);const x=sb(o,t,i,r,ou,lu,cu,Bv);if(x){const _=new he;Pi.getBarycoord(Bv,ou,lu,cu,_),c&&(x.uv=Pi.getInterpolatedAttribute(c,h,m,g,_,new Ut)),f&&(x.uv1=Pi.getInterpolatedAttribute(f,h,m,g,_,new Ut)),d&&(x.normal=Pi.getInterpolatedAttribute(d,h,m,g,_,new he),x.normal.dot(r.direction)>0&&x.normal.multiplyScalar(-1));const v={a:h,b:m,c:g,normal:new he,materialIndex:0};Pi.getNormal(ou,lu,cu,v.normal),x.face=v,x.barycoord=_}return x}class rb extends kn{constructor(t=null,i=1,r=1,c,f,d,h,m,g=Un,x=Un,_,v){super(null,d,h,m,g,x,c,f,_,v),this.isDataTexture=!0,this.image={data:t,width:i,height:r},this.generateMipmaps=!1,this.flipY=!1,this.unpackAlignment=1}}const qs=new Kp,ob=new Ut(.5,.5),hu=new he;class Qp{constructor(t=new _s,i=new _s,r=new _s,c=new _s,f=new _s,d=new _s){this.planes=[t,i,r,c,f,d]}set(t,i,r,c,f,d){const h=this.planes;return h[0].copy(t),h[1].copy(i),h[2].copy(r),h[3].copy(c),h[4].copy(f),h[5].copy(d),this}copy(t){const i=this.planes;for(let r=0;r<6;r++)i[r].copy(t.planes[r]);return this}setFromProjectionMatrix(t,i=aa,r=!1){const c=this.planes,f=t.elements,d=f[0],h=f[1],m=f[2],g=f[3],x=f[4],_=f[5],v=f[6],E=f[7],R=f[8],N=f[9],y=f[10],S=f[11],z=f[12],W=f[13],C=f[14],D=f[15];if(c[0].setComponents(g-d,E-x,S-R,D-z).normalize(),c[1].setComponents(g+d,E+x,S+R,D+z).normalize(),c[2].setComponents(g+h,E+_,S+N,D+W).normalize(),c[3].setComponents(g-h,E-_,S-N,D-W).normalize(),r)c[4].setComponents(m,v,y,C).normalize(),c[5].setComponents(g-m,E-v,S-y,D-C).normalize();else if(c[4].setComponents(g-m,E-v,S-y,D-C).normalize(),i===aa)c[5].setComponents(g+m,E+v,S+y,D+C).normalize();else if(i===Ml)c[5].setComponents(m,v,y,C).normalize();else throw new Error("THREE.Frustum.setFromProjectionMatrix(): Invalid coordinate system: "+i);return this}intersectsObject(t){if(t.boundingSphere!==void 0)t.boundingSphere===null&&t.computeBoundingSphere(),qs.copy(t.boundingSphere).applyMatrix4(t.matrixWorld);else{const i=t.geometry;i.boundingSphere===null&&i.computeBoundingSphere(),qs.copy(i.boundingSphere).applyMatrix4(t.matrixWorld)}return this.intersectsSphere(qs)}intersectsSprite(t){qs.center.set(0,0,0);const i=ob.distanceTo(t.center);return qs.radius=.7071067811865476+i,qs.applyMatrix4(t.matrixWorld),this.intersectsSphere(qs)}intersectsSphere(t){const i=this.planes,r=t.center,c=-t.radius;for(let f=0;f<6;f++)if(i[f].distanceToPoint(r)<c)return!1;return!0}intersectsBox(t){const i=this.planes;for(let r=0;r<6;r++){const c=i[r];if(hu.x=c.normal.x>0?t.max.x:t.min.x,hu.y=c.normal.y>0?t.max.y:t.min.y,hu.z=c.normal.z>0?t.max.z:t.min.z,c.distanceToPoint(hu)<0)return!1}return!0}containsPoint(t){const i=this.planes;for(let r=0;r<6;r++)if(i[r].distanceToPoint(t)<0)return!1;return!0}clone(){return new this.constructor().copy(this)}}class Xx extends kn{constructor(t=[],i=Qs,r,c,f,d,h,m,g,x){super(t,i,r,c,f,d,h,m,g,x),this.isCubeTexture=!0,this.flipY=!1}get images(){return this.image}set images(t){this.image=t}}class El extends kn{constructor(t,i,r=ra,c,f,d,h=Un,m=Un,g,x=Ia,_=1){if(x!==Ia&&x!==Zs)throw new Error("THREE.DepthTexture: format must be either THREE.DepthFormat or THREE.DepthStencilFormat");const v={width:t,height:i,depth:_};super(v,c,f,d,h,m,x,r,g),this.isDepthTexture=!0,this.flipY=!1,this.generateMipmaps=!1,this.compareFunction=null}copy(t){return super.copy(t),this.source=new Zp(Object.assign({},t.image)),this.compareFunction=t.compareFunction,this}toJSON(t){const i=super.toJSON(t);return i.compareFunction=this.compareFunction,i}}class lb extends El{constructor(t,i=ra,r=Qs,c,f,d=Un,h=Un,m,g=Ia){const x={width:t,height:t,depth:1},_=[x,x,x,x,x,x];super(t,t,i,r,c,f,d,h,m,g),this.image=_,this.isCubeDepthTexture=!0,this.isCubeTexture=!0}get images(){return this.image}set images(t){this.image=t}}class Wx extends kn{constructor(t=null){super(),this.sourceTexture=t,this.isExternalTexture=!0}copy(t){return super.copy(t),this.sourceTexture=t.sourceTexture,this}}class oo extends ua{constructor(t=1,i=1,r=1,c=1,f=1,d=1){super(),this.type="BoxGeometry",this.parameters={width:t,height:i,depth:r,widthSegments:c,heightSegments:f,depthSegments:d};const h=this;c=Math.floor(c),f=Math.floor(f),d=Math.floor(d);const m=[],g=[],x=[],_=[];let v=0,E=0;R("z","y","x",-1,-1,r,i,t,d,f,0),R("z","y","x",1,-1,r,i,-t,d,f,1),R("x","z","y",1,1,t,r,i,c,d,2),R("x","z","y",1,-1,t,r,-i,c,d,3),R("x","y","z",1,-1,t,i,r,c,f,4),R("x","y","z",-1,-1,t,i,-r,c,f,5),this.setIndex(m),this.setAttribute("position",new Ti(g,3)),this.setAttribute("normal",new Ti(x,3)),this.setAttribute("uv",new Ti(_,2));function R(N,y,S,z,W,C,D,U,I,b,L){const k=C/I,Y=D/b,$=C/2,O=D/2,q=U/2,ne=I+1,j=b+1;let Q=0,pe=0;const ce=new he;for(let me=0;me<j;me++){const be=me*Y-O;for(let Ze=0;Ze<ne;Ze++){const je=Ze*k-$;ce[N]=je*z,ce[y]=be*W,ce[S]=q,g.push(ce.x,ce.y,ce.z),ce[N]=0,ce[y]=0,ce[S]=U>0?1:-1,x.push(ce.x,ce.y,ce.z),_.push(Ze/I),_.push(1-me/b),Q+=1}}for(let me=0;me<b;me++)for(let be=0;be<I;be++){const Ze=v+be+ne*me,je=v+be+ne*(me+1),H=v+(be+1)+ne*(me+1),F=v+(be+1)+ne*me;m.push(Ze,je,F),m.push(je,H,F),pe+=6}h.addGroup(E,pe,L),E+=pe,v+=Q}}copy(t){return super.copy(t),this.parameters=Object.assign({},t.parameters),this}static fromJSON(t){return new oo(t.width,t.height,t.depth,t.widthSegments,t.heightSegments,t.depthSegments)}}class Cu extends ua{constructor(t=1,i=1,r=1,c=32,f=1,d=!1,h=0,m=Math.PI*2){super(),this.type="CylinderGeometry",this.parameters={radiusTop:t,radiusBottom:i,height:r,radialSegments:c,heightSegments:f,openEnded:d,thetaStart:h,thetaLength:m};const g=this;c=Math.floor(c),f=Math.floor(f);const x=[],_=[],v=[],E=[];let R=0;const N=[],y=r/2;let S=0;z(),d===!1&&(t>0&&W(!0),i>0&&W(!1)),this.setIndex(x),this.setAttribute("position",new Ti(_,3)),this.setAttribute("normal",new Ti(v,3)),this.setAttribute("uv",new Ti(E,2));function z(){const C=new he,D=new he;let U=0;const I=(i-t)/r;for(let b=0;b<=f;b++){const L=[],k=b/f,Y=k*(i-t)+t;for(let $=0;$<=c;$++){const O=$/c,q=O*m+h,ne=Math.sin(q),j=Math.cos(q);D.x=Y*ne,D.y=-k*r+y,D.z=Y*j,_.push(D.x,D.y,D.z),C.set(ne,I,j).normalize(),v.push(C.x,C.y,C.z),E.push(O,1-k),L.push(R++)}N.push(L)}for(let b=0;b<c;b++)for(let L=0;L<f;L++){const k=N[L][b],Y=N[L+1][b],$=N[L+1][b+1],O=N[L][b+1];(t>0||L!==0)&&(x.push(k,Y,O),U+=3),(i>0||L!==f-1)&&(x.push(Y,$,O),U+=3)}g.addGroup(S,U,0),S+=U}function W(C){const D=R,U=new Ut,I=new he;let b=0;const L=C===!0?t:i,k=C===!0?1:-1;for(let $=1;$<=c;$++)_.push(0,y*k,0),v.push(0,k,0),E.push(.5,.5),R++;const Y=R;for(let $=0;$<=c;$++){const q=$/c*m+h,ne=Math.cos(q),j=Math.sin(q);I.x=L*j,I.y=y*k,I.z=L*ne,_.push(I.x,I.y,I.z),v.push(0,k,0),U.x=ne*.5+.5,U.y=j*.5*k+.5,E.push(U.x,U.y),R++}for(let $=0;$<c;$++){const O=D+$,q=Y+$;C===!0?x.push(q,q+1,O):x.push(q+1,q,O),b+=3}g.addGroup(S,b,C===!0?1:2),S+=b}}copy(t){return super.copy(t),this.parameters=Object.assign({},t.parameters),this}static fromJSON(t){return new Cu(t.radiusTop,t.radiusBottom,t.height,t.radialSegments,t.heightSegments,t.openEnded,t.thetaStart,t.thetaLength)}}class Uu extends ua{constructor(t=1,i=1,r=1,c=1){super(),this.type="PlaneGeometry",this.parameters={width:t,height:i,widthSegments:r,heightSegments:c};const f=t/2,d=i/2,h=Math.floor(r),m=Math.floor(c),g=h+1,x=m+1,_=t/h,v=i/m,E=[],R=[],N=[],y=[];for(let S=0;S<x;S++){const z=S*v-d;for(let W=0;W<g;W++){const C=W*_-f;R.push(C,-z,0),N.push(0,0,1),y.push(W/h),y.push(1-S/m)}}for(let S=0;S<m;S++)for(let z=0;z<h;z++){const W=z+g*S,C=z+g*(S+1),D=z+1+g*(S+1),U=z+1+g*S;E.push(W,C,U),E.push(C,D,U)}this.setIndex(E),this.setAttribute("position",new Ti(R,3)),this.setAttribute("normal",new Ti(N,3)),this.setAttribute("uv",new Ti(y,2))}copy(t){return super.copy(t),this.parameters=Object.assign({},t.parameters),this}static fromJSON(t){return new Uu(t.width,t.height,t.widthSegments,t.heightSegments)}}function so(o){const t={};for(const i in o){t[i]={};for(const r in o[i]){const c=o[i][r];if(zv(c))c.isRenderTargetTexture?(ot("UniformsUtils: Textures of render targets cannot be cloned via cloneUniforms() or mergeUniforms()."),t[i][r]=null):t[i][r]=c.clone();else if(Array.isArray(c))if(zv(c[0])){const f=[];for(let d=0,h=c.length;d<h;d++)f[d]=c[d].clone();t[i][r]=f}else t[i][r]=c.slice();else t[i][r]=c}}return t}function Vn(o){const t={};for(let i=0;i<o.length;i++){const r=so(o[i]);for(const c in r)t[c]=r[c]}return t}function zv(o){return o&&(o.isColor||o.isMatrix3||o.isMatrix4||o.isVector2||o.isVector3||o.isVector4||o.isTexture||o.isQuaternion)}function cb(o){const t=[];for(let i=0;i<o.length;i++)t.push(o[i].clone());return t}function qx(o){const t=o.getRenderTarget();return t===null?o.outputColorSpace:t.isXRRenderTarget===!0?t.texture.colorSpace:Ct.workingColorSpace}const ub={clone:so,merge:Vn};var fb=`void main() {
	gl_Position = projectionMatrix * modelViewMatrix * vec4( position, 1.0 );
}`,db=`void main() {
	gl_FragColor = vec4( 1.0, 0.0, 0.0, 1.0 );
}`;class ca extends Al{constructor(t){super(),this.isShaderMaterial=!0,this.type="ShaderMaterial",this.defines={},this.uniforms={},this.uniformsGroups=[],this.vertexShader=fb,this.fragmentShader=db,this.linewidth=1,this.wireframe=!1,this.wireframeLinewidth=1,this.fog=!1,this.lights=!1,this.clipping=!1,this.forceSinglePass=!0,this.extensions={clipCullDistance:!1,multiDraw:!1},this.defaultAttributeValues={color:[1,1,1],uv:[0,0],uv1:[0,0]},this.index0AttributeName=void 0,this.uniformsNeedUpdate=!1,this.glslVersion=null,t!==void 0&&this.setValues(t)}copy(t){return super.copy(t),this.fragmentShader=t.fragmentShader,this.vertexShader=t.vertexShader,this.uniforms=so(t.uniforms),this.uniformsGroups=cb(t.uniformsGroups),this.defines=Object.assign({},t.defines),this.wireframe=t.wireframe,this.wireframeLinewidth=t.wireframeLinewidth,this.fog=t.fog,this.lights=t.lights,this.clipping=t.clipping,this.extensions=Object.assign({},t.extensions),this.glslVersion=t.glslVersion,this.defaultAttributeValues=Object.assign({},t.defaultAttributeValues),this.index0AttributeName=t.index0AttributeName,this.uniformsNeedUpdate=t.uniformsNeedUpdate,this}toJSON(t){const i=super.toJSON(t);i.glslVersion=this.glslVersion,i.uniforms={};for(const c in this.uniforms){const d=this.uniforms[c].value;d&&d.isTexture?i.uniforms[c]={type:"t",value:d.toJSON(t).uuid}:d&&d.isColor?i.uniforms[c]={type:"c",value:d.getHex()}:d&&d.isVector2?i.uniforms[c]={type:"v2",value:d.toArray()}:d&&d.isVector3?i.uniforms[c]={type:"v3",value:d.toArray()}:d&&d.isVector4?i.uniforms[c]={type:"v4",value:d.toArray()}:d&&d.isMatrix3?i.uniforms[c]={type:"m3",value:d.toArray()}:d&&d.isMatrix4?i.uniforms[c]={type:"m4",value:d.toArray()}:i.uniforms[c]={value:d}}Object.keys(this.defines).length>0&&(i.defines=this.defines),i.vertexShader=this.vertexShader,i.fragmentShader=this.fragmentShader,i.lights=this.lights,i.clipping=this.clipping;const r={};for(const c in this.extensions)this.extensions[c]===!0&&(r[c]=!0);return Object.keys(r).length>0&&(i.extensions=r),i}fromJSON(t,i){if(super.fromJSON(t,i),t.uniforms!==void 0)for(const r in t.uniforms){const c=t.uniforms[r];switch(this.uniforms[r]={},c.type){case"t":this.uniforms[r].value=i[c.value]||null;break;case"c":this.uniforms[r].value=new Dt().setHex(c.value);break;case"v2":this.uniforms[r].value=new Ut().fromArray(c.value);break;case"v3":this.uniforms[r].value=new he().fromArray(c.value);break;case"v4":this.uniforms[r].value=new sn().fromArray(c.value);break;case"m3":this.uniforms[r].value=new ft().fromArray(c.value);break;case"m4":this.uniforms[r].value=new un().fromArray(c.value);break;default:this.uniforms[r].value=c.value}}if(t.defines!==void 0&&(this.defines=t.defines),t.vertexShader!==void 0&&(this.vertexShader=t.vertexShader),t.fragmentShader!==void 0&&(this.fragmentShader=t.fragmentShader),t.glslVersion!==void 0&&(this.glslVersion=t.glslVersion),t.extensions!==void 0)for(const r in t.extensions)this.extensions[r]=t.extensions[r];return t.lights!==void 0&&(this.lights=t.lights),t.clipping!==void 0&&(this.clipping=t.clipping),this}}class hb extends ca{constructor(t){super(t),this.isRawShaderMaterial=!0,this.type="RawShaderMaterial"}}class Hh extends Al{constructor(t){super(),this.isMeshStandardMaterial=!0,this.type="MeshStandardMaterial",this.defines={STANDARD:""},this.color=new Dt(16777215),this.roughness=1,this.metalness=0,this.map=null,this.lightMap=null,this.lightMapIntensity=1,this.aoMap=null,this.aoMapIntensity=1,this.emissive=new Dt(0),this.emissiveIntensity=1,this.emissiveMap=null,this.bumpMap=null,this.bumpScale=1,this.normalMap=null,this.normalMapType=Op,this.normalScale=new Ut(1,1),this.displacementMap=null,this.displacementScale=1,this.displacementBias=0,this.roughnessMap=null,this.metalnessMap=null,this.alphaMap=null,this.envMap=null,this.envMapRotation=new xs,this.envMapIntensity=1,this.wireframe=!1,this.wireframeLinewidth=1,this.wireframeLinecap="round",this.wireframeLinejoin="round",this.flatShading=!1,this.fog=!0,this.setValues(t)}copy(t){return super.copy(t),this.defines={STANDARD:""},this.color.copy(t.color),this.roughness=t.roughness,this.metalness=t.metalness,this.map=t.map,this.lightMap=t.lightMap,this.lightMapIntensity=t.lightMapIntensity,this.aoMap=t.aoMap,this.aoMapIntensity=t.aoMapIntensity,this.emissive.copy(t.emissive),this.emissiveMap=t.emissiveMap,this.emissiveIntensity=t.emissiveIntensity,this.bumpMap=t.bumpMap,this.bumpScale=t.bumpScale,this.normalMap=t.normalMap,this.normalMapType=t.normalMapType,this.normalScale.copy(t.normalScale),this.displacementMap=t.displacementMap,this.displacementScale=t.displacementScale,this.displacementBias=t.displacementBias,this.roughnessMap=t.roughnessMap,this.metalnessMap=t.metalnessMap,this.alphaMap=t.alphaMap,this.envMap=t.envMap,this.envMapRotation.copy(t.envMapRotation),this.envMapIntensity=t.envMapIntensity,this.wireframe=t.wireframe,this.wireframeLinewidth=t.wireframeLinewidth,this.wireframeLinecap=t.wireframeLinecap,this.wireframeLinejoin=t.wireframeLinejoin,this.flatShading=t.flatShading,this.fog=t.fog,this}}class pb extends Al{constructor(t){super(),this.isMeshDepthMaterial=!0,this.type="MeshDepthMaterial",this.depthPacking=EE,this.map=null,this.alphaMap=null,this.displacementMap=null,this.displacementScale=1,this.displacementBias=0,this.wireframe=!1,this.wireframeLinewidth=1,this.setValues(t)}copy(t){return super.copy(t),this.depthPacking=t.depthPacking,this.map=t.map,this.alphaMap=t.alphaMap,this.displacementMap=t.displacementMap,this.displacementScale=t.displacementScale,this.displacementBias=t.displacementBias,this.wireframe=t.wireframe,this.wireframeLinewidth=t.wireframeLinewidth,this}}class mb extends Al{constructor(t){super(),this.isMeshDistanceMaterial=!0,this.type="MeshDistanceMaterial",this.map=null,this.alphaMap=null,this.displacementMap=null,this.displacementScale=1,this.displacementBias=0,this.setValues(t)}copy(t){return super.copy(t),this.map=t.map,this.alphaMap=t.alphaMap,this.displacementMap=t.displacementMap,this.displacementScale=t.displacementScale,this.displacementBias=t.displacementBias,this}}class Yx extends Qn{constructor(t,i=1){super(),this.isLight=!0,this.type="Light",this.color=new Dt(t),this.intensity=i}copy(t,i){return super.copy(t,i),this.color.copy(t.color),this.intensity=t.intensity,this}toJSON(t){const i=super.toJSON(t);return i.object.color=this.color.getHex(),i.object.intensity=this.intensity,i}}const Gh=new un,Fv=new he,Hv=new he;class gb{constructor(t){this.camera=t,this.intensity=1,this.bias=0,this.biasNode=null,this.normalBias=0,this.radius=1,this.blurSamples=8,this.mapSize=new Ut(512,512),this.mapType=fi,this.map=null,this.mapPass=null,this.matrix=new un,this.autoUpdate=!0,this.needsUpdate=!1,this._frustum=new Qp,this._frameExtents=new Ut(1,1),this._viewportCount=1,this._viewports=[new sn(0,0,1,1)]}getViewportCount(){return this._viewportCount}getCamera(){return this.camera}getFrustum(){return this._frustum}updateMatrices(t){const i=this.camera;Fv.setFromMatrixPosition(t.matrixWorld),i.position.copy(Fv),Hv.setFromMatrixPosition(t.target.matrixWorld),i.lookAt(Hv),i.updateMatrixWorld(),this._updateMatrix(i,this.matrix,this._frustum)}_updateMatrix(t,i,r,c){Gh.multiplyMatrices(t.projectionMatrix,t.matrixWorldInverse),r.setFromProjectionMatrix(Gh,t.coordinateSystem,t.reversedDepth);const f=this._frameExtents,d=c?c.z/f.x:1,h=c?c.w/f.y:1,m=c?c.x/f.x:0,g=c?c.y/f.y:0;t.coordinateSystem===Ml||t.reversedDepth?i.set(.5*d,0,0,.5*d+m,0,.5*h,0,.5*h+g,0,0,1,0,0,0,0,1):i.set(.5*d,0,0,.5*d+m,0,.5*h,0,.5*h+g,0,0,.5,.5,0,0,0,1),i.multiply(Gh)}getViewport(t){return this._viewports[t]}getFrameExtents(){return this._frameExtents}dispose(){this.map&&this.map.dispose(),this.mapPass&&this.mapPass.dispose()}copy(t){return this.camera=t.camera.clone(),this.intensity=t.intensity,this.bias=t.bias,this.radius=t.radius,this.autoUpdate=t.autoUpdate,this.needsUpdate=t.needsUpdate,this.normalBias=t.normalBias,this.blurSamples=t.blurSamples,this.mapSize.copy(t.mapSize),this.biasNode=t.biasNode,this}clone(){return new this.constructor().copy(this)}toJSON(){const t={};return t.intensity=this.intensity,t.bias=this.bias,t.normalBias=this.normalBias,t.radius=this.radius,t.blurSamples=this.blurSamples,t.mapSize=this.mapSize.toArray(),t.camera=this.camera.toJSON(!1).object,delete t.camera.matrix,t}}const pu=new he,mu=new ro,Ji=new he;class jx extends Qn{constructor(){super(),this.isCamera=!0,this.type="Camera",this.matrixWorldInverse=new un,this.projectionMatrix=new un,this.projectionMatrixInverse=new un,this.coordinateSystem=aa,this._reversedDepth=!1}get reversedDepth(){return this._reversedDepth}copy(t,i){return super.copy(t,i),this.matrixWorldInverse.copy(t.matrixWorldInverse),this.projectionMatrix.copy(t.projectionMatrix),this.projectionMatrixInverse.copy(t.projectionMatrixInverse),this.coordinateSystem=t.coordinateSystem,this}getWorldDirection(t){return super.getWorldDirection(t).negate()}updateMatrixWorld(t){super.updateMatrixWorld(t),this.matrixWorld.decompose(pu,mu,Ji),Ji.x===1&&Ji.y===1&&Ji.z===1?this.matrixWorldInverse.copy(this.matrixWorld).invert():this.matrixWorldInverse.compose(pu,mu,Ji.set(1,1,1)).invert()}updateWorldMatrix(t,i,r=!1){super.updateWorldMatrix(t,i,r),this.matrixWorld.decompose(pu,mu,Ji),Ji.x===1&&Ji.y===1&&Ji.z===1?this.matrixWorldInverse.copy(this.matrixWorld).invert():this.matrixWorldInverse.compose(pu,mu,Ji.set(1,1,1)).invert()}clone(){return new this.constructor().copy(this)}}const ms=new he,Gv=new Ut,Vv=new Ut;class ui extends jx{constructor(t=50,i=1,r=.1,c=2e3){super(),this.isPerspectiveCamera=!0,this.type="PerspectiveCamera",this.fov=t,this.zoom=1,this.near=r,this.far=c,this.focus=10,this.aspect=i,this.view=null,this.filmGauge=35,this.filmOffset=0,this.updateProjectionMatrix()}copy(t,i){return super.copy(t,i),this.fov=t.fov,this.zoom=t.zoom,this.near=t.near,this.far=t.far,this.focus=t.focus,this.aspect=t.aspect,this.view=t.view===null?null:Object.assign({},t.view),this.filmGauge=t.filmGauge,this.filmOffset=t.filmOffset,this}setFocalLength(t){const i=.5*this.getFilmHeight()/t;this.fov=Pp*2*Math.atan(i),this.updateProjectionMatrix()}getFocalLength(){const t=Math.tan(vh*.5*this.fov);return .5*this.getFilmHeight()/t}getEffectiveFOV(){return Pp*2*Math.atan(Math.tan(vh*.5*this.fov)/this.zoom)}getFilmWidth(){return this.filmGauge*Math.min(this.aspect,1)}getFilmHeight(){return this.filmGauge/Math.max(this.aspect,1)}getViewBounds(t,i,r){ms.set(-1,-1,.5).applyMatrix4(this.projectionMatrixInverse),i.set(ms.x,ms.y).multiplyScalar(-t/ms.z),ms.set(1,1,.5).applyMatrix4(this.projectionMatrixInverse),r.set(ms.x,ms.y).multiplyScalar(-t/ms.z)}getViewSize(t,i){return this.getViewBounds(t,Gv,Vv),i.subVectors(Vv,Gv)}setViewOffset(t,i,r,c,f,d){this.aspect=t/i,this.view===null&&(this.view={enabled:!0,fullWidth:1,fullHeight:1,offsetX:0,offsetY:0,width:1,height:1}),this.view.enabled=!0,this.view.fullWidth=t,this.view.fullHeight=i,this.view.offsetX=r,this.view.offsetY=c,this.view.width=f,this.view.height=d,this.updateProjectionMatrix()}clearViewOffset(){this.view!==null&&(this.view.enabled=!1),this.updateProjectionMatrix()}updateProjectionMatrix(){const t=this.near;let i=t*Math.tan(vh*.5*this.fov)/this.zoom,r=2*i,c=this.aspect*r,f=-.5*c;const d=this.view;if(this.view!==null&&this.view.enabled){const m=d.fullWidth,g=d.fullHeight;f+=d.offsetX*c/m,i-=d.offsetY*r/g,c*=d.width/m,r*=d.height/g}const h=this.filmOffset;h!==0&&(f+=t*h/this.getFilmWidth()),this.projectionMatrix.makePerspective(f,f+c,i,i-r,t,this.far,this.coordinateSystem,this.reversedDepth),this.projectionMatrixInverse.copy(this.projectionMatrix).invert()}toJSON(t){const i=super.toJSON(t);return i.object.fov=this.fov,i.object.zoom=this.zoom,i.object.near=this.near,i.object.far=this.far,i.object.focus=this.focus,i.object.aspect=this.aspect,this.view!==null&&(i.object.view=Object.assign({},this.view)),i.object.filmGauge=this.filmGauge,i.object.filmOffset=this.filmOffset,i}}class _b extends gb{constructor(){super(new ui(90,1,.5,500)),this.isPointLightShadow=!0}}class vb extends Yx{constructor(t,i,r=0,c=2){super(t,i),this.isPointLight=!0,this.type="PointLight",this.distance=r,this.decay=c,this.shadow=new _b}get power(){return this.intensity*4*Math.PI}set power(t){this.intensity=t/(4*Math.PI)}dispose(){super.dispose(),this.shadow.dispose()}copy(t,i){return super.copy(t,i),this.distance=t.distance,this.decay=t.decay,this.shadow=t.shadow.clone(),this}toJSON(t){const i=super.toJSON(t);return i.object.distance=this.distance,i.object.decay=this.decay,i.object.shadow=this.shadow.toJSON(),i}}class Zx extends jx{constructor(t=-1,i=1,r=1,c=-1,f=.1,d=2e3){super(),this.isOrthographicCamera=!0,this.type="OrthographicCamera",this.zoom=1,this.view=null,this.left=t,this.right=i,this.top=r,this.bottom=c,this.near=f,this.far=d,this.updateProjectionMatrix()}copy(t,i){return super.copy(t,i),this.left=t.left,this.right=t.right,this.top=t.top,this.bottom=t.bottom,this.near=t.near,this.far=t.far,this.zoom=t.zoom,this.view=t.view===null?null:Object.assign({},t.view),this}setViewOffset(t,i,r,c,f,d){this.view===null&&(this.view={enabled:!0,fullWidth:1,fullHeight:1,offsetX:0,offsetY:0,width:1,height:1}),this.view.enabled=!0,this.view.fullWidth=t,this.view.fullHeight=i,this.view.offsetX=r,this.view.offsetY=c,this.view.width=f,this.view.height=d,this.updateProjectionMatrix()}clearViewOffset(){this.view!==null&&(this.view.enabled=!1),this.updateProjectionMatrix()}updateProjectionMatrix(){const t=(this.right-this.left)/(2*this.zoom),i=(this.top-this.bottom)/(2*this.zoom),r=(this.right+this.left)/2,c=(this.top+this.bottom)/2;let f=r-t,d=r+t,h=c+i,m=c-i;if(this.view!==null&&this.view.enabled){const g=(this.right-this.left)/this.view.fullWidth/this.zoom,x=(this.top-this.bottom)/this.view.fullHeight/this.zoom;f+=g*this.view.offsetX,d=f+g*this.view.width,h-=x*this.view.offsetY,m=h-x*this.view.height}this.projectionMatrix.makeOrthographic(f,d,h,m,this.near,this.far,this.coordinateSystem,this.reversedDepth),this.projectionMatrixInverse.copy(this.projectionMatrix).invert()}toJSON(t){const i=super.toJSON(t);return i.object.zoom=this.zoom,i.object.left=this.left,i.object.right=this.right,i.object.top=this.top,i.object.bottom=this.bottom,i.object.near=this.near,i.object.far=this.far,this.view!==null&&(i.object.view=Object.assign({},this.view)),i}}class xb extends Yx{constructor(t,i){super(t,i),this.isAmbientLight=!0,this.type="AmbientLight"}}const Qr=-90,Jr=1;class Sb extends Qn{constructor(t,i,r){super(),this.type="CubeCamera",this.renderTarget=r,this.coordinateSystem=null,this.activeMipmapLevel=0;const c=new ui(Qr,Jr,t,i);c.layers=this.layers,this.add(c);const f=new ui(Qr,Jr,t,i);f.layers=this.layers,this.add(f);const d=new ui(Qr,Jr,t,i);d.layers=this.layers,this.add(d);const h=new ui(Qr,Jr,t,i);h.layers=this.layers,this.add(h);const m=new ui(Qr,Jr,t,i);m.layers=this.layers,this.add(m);const g=new ui(Qr,Jr,t,i);g.layers=this.layers,this.add(g)}updateCoordinateSystem(){const t=this.coordinateSystem,i=this.children.concat(),[r,c,f,d,h,m]=i;for(const g of i)this.remove(g);if(t===aa)r.up.set(0,1,0),r.lookAt(1,0,0),c.up.set(0,1,0),c.lookAt(-1,0,0),f.up.set(0,0,-1),f.lookAt(0,1,0),d.up.set(0,0,1),d.lookAt(0,-1,0),h.up.set(0,1,0),h.lookAt(0,0,1),m.up.set(0,1,0),m.lookAt(0,0,-1);else if(t===Ml)r.up.set(0,-1,0),r.lookAt(-1,0,0),c.up.set(0,-1,0),c.lookAt(1,0,0),f.up.set(0,0,1),f.lookAt(0,1,0),d.up.set(0,0,-1),d.lookAt(0,-1,0),h.up.set(0,-1,0),h.lookAt(0,0,1),m.up.set(0,-1,0),m.lookAt(0,0,-1);else throw new Error("THREE.CubeCamera.updateCoordinateSystem(): Invalid coordinate system: "+t);for(const g of i)this.add(g),g.updateMatrixWorld()}update(t,i){this.parent===null&&this.updateMatrixWorld();const{renderTarget:r,activeMipmapLevel:c}=this;this.coordinateSystem!==t.coordinateSystem&&(this.coordinateSystem=t.coordinateSystem,this.updateCoordinateSystem());const[f,d,h,m,g,x]=this.children,_=t.getRenderTarget(),v=t.getActiveCubeFace(),E=t.getActiveMipmapLevel(),R=t.xr.enabled;t.xr.enabled=!1;const N=r.texture.generateMipmaps;r.texture.generateMipmaps=!1;let y=!1;t.isWebGLRenderer===!0?y=t.state.buffers.depth.getReversed():y=t.reversedDepthBuffer,t.setRenderTarget(r,0,c),y&&t.autoClear===!1&&t.clearDepth(),t.render(i,f),t.setRenderTarget(r,1,c),y&&t.autoClear===!1&&t.clearDepth(),t.render(i,d),t.setRenderTarget(r,2,c),y&&t.autoClear===!1&&t.clearDepth(),t.render(i,h),t.setRenderTarget(r,3,c),y&&t.autoClear===!1&&t.clearDepth(),t.render(i,m),t.setRenderTarget(r,4,c),y&&t.autoClear===!1&&t.clearDepth(),t.render(i,g),r.texture.generateMipmaps=N,t.setRenderTarget(r,5,c),y&&t.autoClear===!1&&t.clearDepth(),t.render(i,x),t.setRenderTarget(_,v,E),t.xr.enabled=R,r.texture.needsPMREMUpdate=!0}}class yb extends ui{constructor(t=[]){super(),this.isArrayCamera=!0,this.isMultiViewCamera=!1,this.cameras=t}}const nm=class nm{constructor(t,i,r,c){this.elements=[1,0,0,1],t!==void 0&&this.set(t,i,r,c)}identity(){return this.set(1,0,0,1),this}fromArray(t,i=0){for(let r=0;r<4;r++)this.elements[r]=t[r+i];return this}set(t,i,r,c){const f=this.elements;return f[0]=t,f[2]=i,f[1]=r,f[3]=c,this}};nm.prototype.isMatrix2=!0;let kv=nm;function Xv(o,t,i,r){const c=Mb(r);switch(i){case Ox:return o*t;case Ix:return o*t/c.components*c.byteLength;case Xp:return o*t/c.components*c.byteLength;case Js:return o*t*2/c.components*c.byteLength;case Wp:return o*t*2/c.components*c.byteLength;case Px:return o*t*3/c.components*c.byteLength;case Ii:return o*t*4/c.components*c.byteLength;case qp:return o*t*4/c.components*c.byteLength;case xu:case Su:return Math.floor((o+3)/4)*Math.floor((t+3)/4)*8;case yu:case Mu:return Math.floor((o+3)/4)*Math.floor((t+3)/4)*16;case sp:case op:return Math.max(o,16)*Math.max(t,8)/4;case ap:case rp:return Math.max(o,8)*Math.max(t,8)/2;case lp:case cp:case fp:case dp:return Math.floor((o+3)/4)*Math.floor((t+3)/4)*8;case up:case bu:case hp:return Math.floor((o+3)/4)*Math.floor((t+3)/4)*16;case pp:return Math.floor((o+3)/4)*Math.floor((t+3)/4)*16;case mp:return Math.floor((o+4)/5)*Math.floor((t+3)/4)*16;case gp:return Math.floor((o+4)/5)*Math.floor((t+4)/5)*16;case _p:return Math.floor((o+5)/6)*Math.floor((t+4)/5)*16;case vp:return Math.floor((o+5)/6)*Math.floor((t+5)/6)*16;case xp:return Math.floor((o+7)/8)*Math.floor((t+4)/5)*16;case Sp:return Math.floor((o+7)/8)*Math.floor((t+5)/6)*16;case yp:return Math.floor((o+7)/8)*Math.floor((t+7)/8)*16;case Mp:return Math.floor((o+9)/10)*Math.floor((t+4)/5)*16;case Ep:return Math.floor((o+9)/10)*Math.floor((t+5)/6)*16;case bp:return Math.floor((o+9)/10)*Math.floor((t+7)/8)*16;case Tp:return Math.floor((o+9)/10)*Math.floor((t+9)/10)*16;case Ap:return Math.floor((o+11)/12)*Math.floor((t+9)/10)*16;case Rp:return Math.floor((o+11)/12)*Math.floor((t+11)/12)*16;case wp:case Cp:case Np:return Math.ceil(o/4)*Math.ceil(t/4)*16;case Dp:case Up:return Math.ceil(o/4)*Math.ceil(t/4)*8;case Tu:case Lp:return Math.ceil(o/4)*Math.ceil(t/4)*16}throw new Error(`Unable to determine texture byte length for ${i} format.`)}function Mb(o){switch(o){case fi:case Nx:return{byteLength:1,components:1};case Sl:case Dx:case oa:return{byteLength:2,components:1};case Vp:case kp:return{byteLength:2,components:4};case ra:case Gp:case ia:return{byteLength:4,components:1};case Ux:case Lx:return{byteLength:4,components:3}}throw new Error(`THREE.TextureUtils: Unknown texture type ${o}.`)}typeof __THREE_DEVTOOLS__<"u"&&__THREE_DEVTOOLS__.dispatchEvent(new CustomEvent("register",{detail:{revision:Hp}}));typeof window<"u"&&(window.__THREE__?ot("WARNING: Multiple instances of Three.js being imported."):window.__THREE__=Hp);/**
 * @license
 * Copyright 2010-2026 Three.js Authors
 * SPDX-License-Identifier: MIT
 */function Kx(){let o=null,t=!1,i=null,r=null;function c(f,d){r=o.requestAnimationFrame(c),i(f,d)}return{start:function(){t!==!0&&i!==null&&o!==null&&(r=o.requestAnimationFrame(c),t=!0)},stop:function(){o!==null&&o.cancelAnimationFrame(r),t=!1},setAnimationLoop:function(f){i=f},setContext:function(f){o=f}}}function Eb(o){const t=new WeakMap;function i(h,m){const g=h.array,x=h.usage,_=g.byteLength,v=o.createBuffer();o.bindBuffer(m,v),o.bufferData(m,g,x),h.onUploadCallback();let E;if(g instanceof Float32Array)E=o.FLOAT;else if(typeof Float16Array<"u"&&g instanceof Float16Array)E=o.HALF_FLOAT;else if(g instanceof Uint16Array)h.isFloat16BufferAttribute?E=o.HALF_FLOAT:E=o.UNSIGNED_SHORT;else if(g instanceof Int16Array)E=o.SHORT;else if(g instanceof Uint32Array)E=o.UNSIGNED_INT;else if(g instanceof Int32Array)E=o.INT;else if(g instanceof Int8Array)E=o.BYTE;else if(g instanceof Uint8Array)E=o.UNSIGNED_BYTE;else if(g instanceof Uint8ClampedArray)E=o.UNSIGNED_BYTE;else throw new Error("THREE.WebGLAttributes: Unsupported buffer data format: "+g);return{buffer:v,type:E,bytesPerElement:g.BYTES_PER_ELEMENT,version:h.version,size:_}}function r(h,m,g){const x=m.array,_=m.updateRanges;if(o.bindBuffer(g,h),_.length===0)o.bufferSubData(g,0,x);else{_.sort((E,R)=>E.start-R.start);let v=0;for(let E=1;E<_.length;E++){const R=_[v],N=_[E];N.start<=R.start+R.count+1?R.count=Math.max(R.count,N.start+N.count-R.start):(++v,_[v]=N)}_.length=v+1;for(let E=0,R=_.length;E<R;E++){const N=_[E];o.bufferSubData(g,N.start*x.BYTES_PER_ELEMENT,x,N.start,N.count)}m.clearUpdateRanges()}m.onUploadCallback()}function c(h){return h.isInterleavedBufferAttribute&&(h=h.data),t.get(h)}function f(h){h.isInterleavedBufferAttribute&&(h=h.data);const m=t.get(h);m&&(o.deleteBuffer(m.buffer),t.delete(h))}function d(h,m){if(h.isInterleavedBufferAttribute&&(h=h.data),h.isGLBufferAttribute){const x=t.get(h);(!x||x.version<h.version)&&t.set(h,{buffer:h.buffer,type:h.type,bytesPerElement:h.elementSize,version:h.version});return}const g=t.get(h);if(g===void 0)t.set(h,i(h,m));else if(g.version<h.version){if(g.size!==h.array.byteLength)throw new Error("THREE.WebGLAttributes: The size of the buffer attribute's array buffer does not match the original size. Resizing buffer attributes is not supported.");r(g.buffer,h,m),g.version=h.version}}return{get:c,remove:f,update:d}}var bb=`#ifdef USE_ALPHAHASH
	if ( diffuseColor.a < getAlphaHashThreshold( vPosition ) ) discard;
#endif`,Tb=`#ifdef USE_ALPHAHASH
	const float ALPHA_HASH_SCALE = 0.05;
	float hash2D( vec2 value ) {
		return fract( 1.0e4 * sin( 17.0 * value.x + 0.1 * value.y ) * ( 0.1 + abs( sin( 13.0 * value.y + value.x ) ) ) );
	}
	float hash3D( vec3 value ) {
		return hash2D( vec2( hash2D( value.xy ), value.z ) );
	}
	float getAlphaHashThreshold( vec3 position ) {
		float maxDeriv = max(
			length( dFdx( position.xyz ) ),
			length( dFdy( position.xyz ) )
		);
		float pixScale = 1.0 / ( ALPHA_HASH_SCALE * maxDeriv );
		vec2 pixScales = vec2(
			exp2( floor( log2( pixScale ) ) ),
			exp2( ceil( log2( pixScale ) ) )
		);
		vec2 alpha = vec2(
			hash3D( floor( pixScales.x * position.xyz ) ),
			hash3D( floor( pixScales.y * position.xyz ) )
		);
		float lerpFactor = fract( log2( pixScale ) );
		float x = ( 1.0 - lerpFactor ) * alpha.x + lerpFactor * alpha.y;
		float a = min( lerpFactor, 1.0 - lerpFactor );
		vec3 cases = vec3(
			x * x / ( 2.0 * a * ( 1.0 - a ) ),
			( x - 0.5 * a ) / ( 1.0 - a ),
			1.0 - ( ( 1.0 - x ) * ( 1.0 - x ) / ( 2.0 * a * ( 1.0 - a ) ) )
		);
		float threshold = ( x < ( 1.0 - a ) )
			? ( ( x < a ) ? cases.x : cases.y )
			: cases.z;
		return clamp( threshold , 1.0e-6, 1.0 );
	}
#endif`,Ab=`#ifdef USE_ALPHAMAP
	diffuseColor.a *= texture2D( alphaMap, vAlphaMapUv ).g;
#endif`,Rb=`#ifdef USE_ALPHAMAP
	uniform sampler2D alphaMap;
#endif`,wb=`#ifdef USE_ALPHATEST
	#ifdef ALPHA_TO_COVERAGE
	diffuseColor.a = smoothstep( alphaTest, alphaTest + fwidth( diffuseColor.a ), diffuseColor.a );
	if ( diffuseColor.a == 0.0 ) discard;
	#else
	if ( diffuseColor.a < alphaTest ) discard;
	#endif
#endif`,Cb=`#ifdef USE_ALPHATEST
	uniform float alphaTest;
#endif`,Nb=`#ifdef USE_AOMAP
	float ambientOcclusion = ( texture2D( aoMap, vAoMapUv ).r - 1.0 ) * aoMapIntensity + 1.0;
	reflectedLight.indirectDiffuse *= ambientOcclusion;
	#if defined( USE_CLEARCOAT ) 
		clearcoatSpecularIndirect *= ambientOcclusion;
	#endif
	#if defined( USE_SHEEN ) 
		sheenSpecularIndirect *= ambientOcclusion;
	#endif
	#if defined( USE_ENVMAP ) && defined( STANDARD )
		float dotNV = saturate( dot( geometryNormal, geometryViewDir ) );
		reflectedLight.indirectSpecular *= computeSpecularOcclusion( dotNV, ambientOcclusion, material.roughness );
	#endif
#endif`,Db=`#ifdef USE_AOMAP
	uniform sampler2D aoMap;
	uniform float aoMapIntensity;
#endif`,Ub=`#ifdef USE_BATCHING
	#if ! defined( GL_ANGLE_multi_draw )
	#define gl_DrawID _gl_DrawID
	uniform int _gl_DrawID;
	#endif
	uniform highp sampler2D batchingTexture;
	uniform highp usampler2D batchingIdTexture;
	mat4 getBatchingMatrix( const in float i ) {
		int size = textureSize( batchingTexture, 0 ).x;
		int j = int( i ) * 4;
		int x = j % size;
		int y = j / size;
		vec4 v1 = texelFetch( batchingTexture, ivec2( x, y ), 0 );
		vec4 v2 = texelFetch( batchingTexture, ivec2( x + 1, y ), 0 );
		vec4 v3 = texelFetch( batchingTexture, ivec2( x + 2, y ), 0 );
		vec4 v4 = texelFetch( batchingTexture, ivec2( x + 3, y ), 0 );
		return mat4( v1, v2, v3, v4 );
	}
	float getIndirectIndex( const in int i ) {
		int size = textureSize( batchingIdTexture, 0 ).x;
		int x = i % size;
		int y = i / size;
		return float( texelFetch( batchingIdTexture, ivec2( x, y ), 0 ).r );
	}
#endif
#ifdef USE_BATCHING_COLOR
	uniform sampler2D batchingColorTexture;
	vec4 getBatchingColor( const in float i ) {
		int size = textureSize( batchingColorTexture, 0 ).x;
		int j = int( i );
		int x = j % size;
		int y = j / size;
		return texelFetch( batchingColorTexture, ivec2( x, y ), 0 );
	}
#endif`,Lb=`#ifdef USE_BATCHING
	mat4 batchingMatrix = getBatchingMatrix( getIndirectIndex( gl_DrawID ) );
#endif`,Ob=`vec3 transformed = vec3( position );
#ifdef USE_ALPHAHASH
	vPosition = vec3( position );
#endif`,Pb=`vec3 objectNormal = vec3( normal );
#ifdef USE_TANGENT
	vec3 objectTangent = vec3( tangent.xyz );
#endif`,Ib=`float G_BlinnPhong_Implicit( ) {
	return 0.25;
}
float D_BlinnPhong( const in float shininess, const in float dotNH ) {
	return RECIPROCAL_PI * ( shininess * 0.5 + 1.0 ) * pow( dotNH, shininess );
}
vec3 BRDF_BlinnPhong( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, const in vec3 specularColor, const in float shininess ) {
	vec3 halfDir = normalize( lightDir + viewDir );
	float dotNH = saturate( dot( normal, halfDir ) );
	float dotVH = saturate( dot( viewDir, halfDir ) );
	vec3 F = F_Schlick( specularColor, 1.0, dotVH );
	float G = G_BlinnPhong_Implicit( );
	float D = D_BlinnPhong( shininess, dotNH );
	return F * ( G * D );
} // validated`,Bb=`#ifdef USE_IRIDESCENCE
	const mat3 XYZ_TO_REC709 = mat3(
		 3.2404542, -0.9692660,  0.0556434,
		-1.5371385,  1.8760108, -0.2040259,
		-0.4985314,  0.0415560,  1.0572252
	);
	vec3 Fresnel0ToIor( vec3 fresnel0 ) {
		vec3 sqrtF0 = sqrt( fresnel0 );
		return ( vec3( 1.0 ) + sqrtF0 ) / ( vec3( 1.0 ) - sqrtF0 );
	}
	vec3 IorToFresnel0( vec3 transmittedIor, float incidentIor ) {
		return pow2( ( transmittedIor - vec3( incidentIor ) ) / ( transmittedIor + vec3( incidentIor ) ) );
	}
	float IorToFresnel0( float transmittedIor, float incidentIor ) {
		return pow2( ( transmittedIor - incidentIor ) / ( transmittedIor + incidentIor ));
	}
	vec3 evalSensitivity( float OPD, vec3 shift ) {
		float phase = 2.0 * PI * OPD * 1.0e-9;
		vec3 val = vec3( 5.4856e-13, 4.4201e-13, 5.2481e-13 );
		vec3 pos = vec3( 1.6810e+06, 1.7953e+06, 2.2084e+06 );
		vec3 var = vec3( 4.3278e+09, 9.3046e+09, 6.6121e+09 );
		vec3 xyz = val * sqrt( 2.0 * PI * var ) * cos( pos * phase + shift ) * exp( - pow2( phase ) * var );
		xyz.x += 9.7470e-14 * sqrt( 2.0 * PI * 4.5282e+09 ) * cos( 2.2399e+06 * phase + shift[ 0 ] ) * exp( - 4.5282e+09 * pow2( phase ) );
		xyz /= 1.0685e-7;
		vec3 rgb = XYZ_TO_REC709 * xyz;
		return rgb;
	}
	vec3 evalIridescence( float outsideIOR, float eta2, float cosTheta1, float thinFilmThickness, vec3 baseF0 ) {
		vec3 I;
		float iridescenceIOR = mix( outsideIOR, eta2, smoothstep( 0.0, 0.03, thinFilmThickness ) );
		float sinTheta2Sq = pow2( outsideIOR / iridescenceIOR ) * ( 1.0 - pow2( cosTheta1 ) );
		float cosTheta2Sq = 1.0 - sinTheta2Sq;
		if ( cosTheta2Sq < 0.0 ) {
			return vec3( 1.0 );
		}
		float cosTheta2 = sqrt( cosTheta2Sq );
		float R0 = IorToFresnel0( iridescenceIOR, outsideIOR );
		float R12 = F_Schlick( R0, 1.0, cosTheta1 );
		float T121 = 1.0 - R12;
		float phi12 = 0.0;
		if ( iridescenceIOR < outsideIOR ) phi12 = PI;
		float phi21 = PI - phi12;
		vec3 baseIOR = Fresnel0ToIor( clamp( baseF0, 0.0, 0.9999 ) );		vec3 R1 = IorToFresnel0( baseIOR, iridescenceIOR );
		vec3 R23 = F_Schlick( R1, 1.0, cosTheta2 );
		vec3 phi23 = vec3( 0.0 );
		if ( baseIOR[ 0 ] < iridescenceIOR ) phi23[ 0 ] = PI;
		if ( baseIOR[ 1 ] < iridescenceIOR ) phi23[ 1 ] = PI;
		if ( baseIOR[ 2 ] < iridescenceIOR ) phi23[ 2 ] = PI;
		float OPD = 2.0 * iridescenceIOR * thinFilmThickness * cosTheta2;
		vec3 phi = vec3( phi21 ) + phi23;
		vec3 R123 = clamp( R12 * R23, 1e-5, 0.9999 );
		vec3 r123 = sqrt( R123 );
		vec3 Rs = pow2( T121 ) * R23 / ( vec3( 1.0 ) - R123 );
		vec3 C0 = R12 + Rs;
		I = C0;
		vec3 Cm = Rs - T121;
		for ( int m = 1; m <= 2; ++ m ) {
			Cm *= r123;
			vec3 Sm = 2.0 * evalSensitivity( float( m ) * OPD, float( m ) * phi );
			I += Cm * Sm;
		}
		return max( I, vec3( 0.0 ) );
	}
#endif`,zb=`#ifdef USE_BUMPMAP
	uniform sampler2D bumpMap;
	uniform float bumpScale;
	vec2 dHdxy_fwd() {
		vec2 dSTdx = dFdx( vBumpMapUv );
		vec2 dSTdy = dFdy( vBumpMapUv );
		float Hll = bumpScale * texture2D( bumpMap, vBumpMapUv ).x;
		float dBx = bumpScale * texture2D( bumpMap, vBumpMapUv + dSTdx ).x - Hll;
		float dBy = bumpScale * texture2D( bumpMap, vBumpMapUv + dSTdy ).x - Hll;
		return vec2( dBx, dBy );
	}
	vec3 perturbNormalArb( vec3 surf_pos, vec3 surf_norm, vec2 dHdxy, float faceDirection ) {
		vec3 vSigmaX = normalize( dFdx( surf_pos.xyz ) );
		vec3 vSigmaY = normalize( dFdy( surf_pos.xyz ) );
		vec3 vN = surf_norm;
		vec3 R1 = cross( vSigmaY, vN );
		vec3 R2 = cross( vN, vSigmaX );
		float fDet = dot( vSigmaX, R1 ) * faceDirection;
		vec3 vGrad = sign( fDet ) * ( dHdxy.x * R1 + dHdxy.y * R2 );
		return normalize( abs( fDet ) * surf_norm - vGrad );
	}
#endif`,Fb=`#if NUM_CLIPPING_PLANES > 0
	vec4 plane;
	#ifdef ALPHA_TO_COVERAGE
		float distanceToPlane, distanceGradient;
		float clipOpacity = 1.0;
		#pragma unroll_loop_start
		for ( int i = 0; i < UNION_CLIPPING_PLANES; i ++ ) {
			plane = clippingPlanes[ i ];
			distanceToPlane = - dot( vClipPosition, plane.xyz ) + plane.w;
			distanceGradient = fwidth( distanceToPlane ) / 2.0;
			clipOpacity *= smoothstep( - distanceGradient, distanceGradient, distanceToPlane );
			if ( clipOpacity == 0.0 ) discard;
		}
		#pragma unroll_loop_end
		#if UNION_CLIPPING_PLANES < NUM_CLIPPING_PLANES
			float unionClipOpacity = 1.0;
			#pragma unroll_loop_start
			for ( int i = UNION_CLIPPING_PLANES; i < NUM_CLIPPING_PLANES; i ++ ) {
				plane = clippingPlanes[ i ];
				distanceToPlane = - dot( vClipPosition, plane.xyz ) + plane.w;
				distanceGradient = fwidth( distanceToPlane ) / 2.0;
				unionClipOpacity *= 1.0 - smoothstep( - distanceGradient, distanceGradient, distanceToPlane );
			}
			#pragma unroll_loop_end
			clipOpacity *= 1.0 - unionClipOpacity;
		#endif
		diffuseColor.a *= clipOpacity;
		if ( diffuseColor.a == 0.0 ) discard;
	#else
		#pragma unroll_loop_start
		for ( int i = 0; i < UNION_CLIPPING_PLANES; i ++ ) {
			plane = clippingPlanes[ i ];
			if ( dot( vClipPosition, plane.xyz ) > plane.w ) discard;
		}
		#pragma unroll_loop_end
		#if UNION_CLIPPING_PLANES < NUM_CLIPPING_PLANES
			bool clipped = true;
			#pragma unroll_loop_start
			for ( int i = UNION_CLIPPING_PLANES; i < NUM_CLIPPING_PLANES; i ++ ) {
				plane = clippingPlanes[ i ];
				clipped = ( dot( vClipPosition, plane.xyz ) > plane.w ) && clipped;
			}
			#pragma unroll_loop_end
			if ( clipped ) discard;
		#endif
	#endif
#endif`,Hb=`#if NUM_CLIPPING_PLANES > 0
	varying vec3 vClipPosition;
	uniform vec4 clippingPlanes[ NUM_CLIPPING_PLANES ];
#endif`,Gb=`#if NUM_CLIPPING_PLANES > 0
	varying vec3 vClipPosition;
#endif`,Vb=`#if NUM_CLIPPING_PLANES > 0
	vClipPosition = - mvPosition.xyz;
#endif`,kb=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA )
	diffuseColor *= vColor;
#endif`,Xb=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA )
	varying vec4 vColor;
#endif`,Wb=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA ) || defined( USE_INSTANCING_COLOR ) || defined( USE_BATCHING_COLOR )
	varying vec4 vColor;
#endif`,qb=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA ) || defined( USE_INSTANCING_COLOR ) || defined( USE_BATCHING_COLOR )
	vColor = vec4( 1.0 );
#endif
#ifdef USE_COLOR_ALPHA
	vColor *= color;
#elif defined( USE_COLOR )
	vColor.rgb *= color;
#endif
#ifdef USE_INSTANCING_COLOR
	vColor.rgb *= instanceColor.rgb;
#endif
#ifdef USE_BATCHING_COLOR
	vColor *= getBatchingColor( getIndirectIndex( gl_DrawID ) );
#endif`,Yb=`#define PI 3.141592653589793
#define PI2 6.283185307179586
#define PI_HALF 1.5707963267948966
#define RECIPROCAL_PI 0.3183098861837907
#define RECIPROCAL_PI2 0.15915494309189535
#define EPSILON 1e-6
#ifndef saturate
#define saturate( a ) clamp( a, 0.0, 1.0 )
#endif
#define whiteComplement( a ) ( 1.0 - saturate( a ) )
float pow2( const in float x ) { return x*x; }
vec3 pow2( const in vec3 x ) { return x*x; }
float pow3( const in float x ) { return x*x*x; }
float pow4( const in float x ) { float x2 = x*x; return x2*x2; }
float max3( const in vec3 v ) { return max( max( v.x, v.y ), v.z ); }
float average( const in vec3 v ) { return dot( v, vec3( 0.3333333 ) ); }
highp float rand( const in vec2 uv ) {
	const highp float a = 12.9898, b = 78.233, c = 43758.5453;
	highp float dt = dot( uv.xy, vec2( a,b ) ), sn = mod( dt, PI );
	return fract( sin( sn ) * c );
}
#ifdef HIGH_PRECISION
	float precisionSafeLength( vec3 v ) { return length( v ); }
#else
	float precisionSafeLength( vec3 v ) {
		float maxComponent = max3( abs( v ) );
		return length( v / maxComponent ) * maxComponent;
	}
#endif
struct IncidentLight {
	vec3 color;
	vec3 direction;
	bool visible;
};
struct ReflectedLight {
	vec3 directDiffuse;
	vec3 directSpecular;
	vec3 indirectDiffuse;
	vec3 indirectSpecular;
};
#ifdef USE_ALPHAHASH
	varying vec3 vPosition;
#endif
vec3 transformDirection( in vec3 dir, in mat4 matrix ) {
	return normalize( ( matrix * vec4( dir, 0.0 ) ).xyz );
}
#define inverseTransformDirection transformDirectionByInverseViewMatrix
vec3 transformNormalByInverseViewMatrix( in vec3 normal, in mat4 viewMatrix ) {
	return normalize( ( vec4( normal, 0.0 ) * viewMatrix ).xyz );
}
vec3 transformDirectionByInverseViewMatrix( in vec3 dir, in mat4 viewMatrix ) {
	return normalize( ( vec4( dir, 0.0 ) * viewMatrix ).xyz );
}
bool isPerspectiveMatrix( mat4 m ) {
	return m[ 2 ][ 3 ] == - 1.0;
}
vec2 equirectUv( in vec3 dir ) {
	float u = atan( dir.z, dir.x ) * RECIPROCAL_PI2 + 0.5;
	float v = asin( clamp( dir.y, - 1.0, 1.0 ) ) * RECIPROCAL_PI + 0.5;
	return vec2( u, v );
}
vec3 BRDF_Lambert( const in vec3 diffuseColor ) {
	return RECIPROCAL_PI * diffuseColor;
}
vec3 F_Schlick( const in vec3 f0, const in float f90, const in float dotVH ) {
	float fresnel = exp2( ( - 5.55473 * dotVH - 6.98316 ) * dotVH );
	return f0 * ( 1.0 - fresnel ) + ( f90 * fresnel );
}
float F_Schlick( const in float f0, const in float f90, const in float dotVH ) {
	float fresnel = exp2( ( - 5.55473 * dotVH - 6.98316 ) * dotVH );
	return f0 * ( 1.0 - fresnel ) + ( f90 * fresnel );
} // validated`,jb=`#ifdef ENVMAP_TYPE_CUBE_UV
	#define cubeUV_minMipLevel 4.0
	#define cubeUV_minTileSize 16.0
	float getFace( vec3 direction ) {
		vec3 absDirection = abs( direction );
		float face = - 1.0;
		if ( absDirection.x > absDirection.z ) {
			if ( absDirection.x > absDirection.y )
				face = direction.x > 0.0 ? 0.0 : 3.0;
			else
				face = direction.y > 0.0 ? 1.0 : 4.0;
		} else {
			if ( absDirection.z > absDirection.y )
				face = direction.z > 0.0 ? 2.0 : 5.0;
			else
				face = direction.y > 0.0 ? 1.0 : 4.0;
		}
		return face;
	}
	vec2 getUV( vec3 direction, float face ) {
		vec2 uv;
		if ( face == 0.0 ) {
			uv = vec2( direction.z, direction.y ) / abs( direction.x );
		} else if ( face == 1.0 ) {
			uv = vec2( - direction.x, - direction.z ) / abs( direction.y );
		} else if ( face == 2.0 ) {
			uv = vec2( - direction.x, direction.y ) / abs( direction.z );
		} else if ( face == 3.0 ) {
			uv = vec2( - direction.z, direction.y ) / abs( direction.x );
		} else if ( face == 4.0 ) {
			uv = vec2( - direction.x, direction.z ) / abs( direction.y );
		} else {
			uv = vec2( direction.x, direction.y ) / abs( direction.z );
		}
		return 0.5 * ( uv + 1.0 );
	}
	vec3 bilinearCubeUV( sampler2D envMap, vec3 direction, float mipInt ) {
		float face = getFace( direction );
		float filterInt = max( cubeUV_minMipLevel - mipInt, 0.0 );
		mipInt = max( mipInt, cubeUV_minMipLevel );
		float faceSize = exp2( mipInt );
		highp vec2 uv = getUV( direction, face ) * ( faceSize - 2.0 ) + 1.0;
		if ( face > 2.0 ) {
			uv.y += faceSize;
			face -= 3.0;
		}
		uv.x += face * faceSize;
		uv.x += filterInt * 3.0 * cubeUV_minTileSize;
		uv.y += 4.0 * ( exp2( CUBEUV_MAX_MIP ) - faceSize );
		uv.x *= CUBEUV_TEXEL_WIDTH;
		uv.y *= CUBEUV_TEXEL_HEIGHT;
		#ifdef texture2DGradEXT
			return texture2DGradEXT( envMap, uv, vec2( 0.0 ), vec2( 0.0 ) ).rgb;
		#else
			return texture2D( envMap, uv ).rgb;
		#endif
	}
	#define cubeUV_r0 1.0
	#define cubeUV_m0 - 2.0
	#define cubeUV_r1 0.8
	#define cubeUV_m1 - 1.0
	#define cubeUV_r4 0.4
	#define cubeUV_m4 2.0
	#define cubeUV_r5 0.305
	#define cubeUV_m5 3.0
	#define cubeUV_r6 0.21
	#define cubeUV_m6 4.0
	float roughnessToMip( float roughness ) {
		float mip = 0.0;
		if ( roughness >= cubeUV_r1 ) {
			mip = ( cubeUV_r0 - roughness ) * ( cubeUV_m1 - cubeUV_m0 ) / ( cubeUV_r0 - cubeUV_r1 ) + cubeUV_m0;
		} else if ( roughness >= cubeUV_r4 ) {
			mip = ( cubeUV_r1 - roughness ) * ( cubeUV_m4 - cubeUV_m1 ) / ( cubeUV_r1 - cubeUV_r4 ) + cubeUV_m1;
		} else if ( roughness >= cubeUV_r5 ) {
			mip = ( cubeUV_r4 - roughness ) * ( cubeUV_m5 - cubeUV_m4 ) / ( cubeUV_r4 - cubeUV_r5 ) + cubeUV_m4;
		} else if ( roughness >= cubeUV_r6 ) {
			mip = ( cubeUV_r5 - roughness ) * ( cubeUV_m6 - cubeUV_m5 ) / ( cubeUV_r5 - cubeUV_r6 ) + cubeUV_m5;
		} else {
			mip = - 2.0 * log2( 1.16 * roughness );		}
		return mip;
	}
	vec4 textureCubeUV( sampler2D envMap, vec3 sampleDir, float roughness ) {
		float mip = clamp( roughnessToMip( roughness ), cubeUV_m0, CUBEUV_MAX_MIP );
		float mipF = fract( mip );
		float mipInt = floor( mip );
		vec3 color0 = bilinearCubeUV( envMap, sampleDir, mipInt );
		if ( mipF == 0.0 ) {
			return vec4( color0, 1.0 );
		} else {
			vec3 color1 = bilinearCubeUV( envMap, sampleDir, mipInt + 1.0 );
			return vec4( mix( color0, color1, mipF ), 1.0 );
		}
	}
#endif`,Zb=`vec3 transformedNormal = objectNormal;
#ifdef USE_TANGENT
	vec3 transformedTangent = objectTangent;
#endif
#ifdef USE_BATCHING
	mat3 bm = mat3( batchingMatrix );
	transformedNormal /= vec3( dot( bm[ 0 ], bm[ 0 ] ), dot( bm[ 1 ], bm[ 1 ] ), dot( bm[ 2 ], bm[ 2 ] ) );
	transformedNormal = bm * transformedNormal;
	#ifdef USE_TANGENT
		transformedTangent = bm * transformedTangent;
	#endif
#endif
#ifdef USE_INSTANCING
	mat3 im = mat3( instanceMatrix );
	transformedNormal /= vec3( dot( im[ 0 ], im[ 0 ] ), dot( im[ 1 ], im[ 1 ] ), dot( im[ 2 ], im[ 2 ] ) );
	transformedNormal = im * transformedNormal;
	#ifdef USE_TANGENT
		transformedTangent = im * transformedTangent;
	#endif
#endif
transformedNormal = normalMatrix * transformedNormal;
#ifdef FLIP_SIDED
	transformedNormal = - transformedNormal;
#endif
#ifdef USE_TANGENT
	transformedTangent = ( modelViewMatrix * vec4( transformedTangent, 0.0 ) ).xyz;
#endif`,Kb=`#ifdef USE_DISPLACEMENTMAP
	uniform sampler2D displacementMap;
	uniform float displacementScale;
	uniform float displacementBias;
#endif`,Qb=`#ifdef USE_DISPLACEMENTMAP
	transformed += normalize( objectNormal ) * ( texture2D( displacementMap, vDisplacementMapUv ).x * displacementScale + displacementBias );
#endif`,Jb=`#ifdef USE_EMISSIVEMAP
	vec4 emissiveColor = texture2D( emissiveMap, vEmissiveMapUv );
	#ifdef DECODE_VIDEO_TEXTURE_EMISSIVE
		emissiveColor = sRGBTransferEOTF( emissiveColor );
	#endif
	totalEmissiveRadiance *= emissiveColor.rgb;
#endif`,$b=`#ifdef USE_EMISSIVEMAP
	uniform sampler2D emissiveMap;
#endif`,eT="gl_FragColor = linearToOutputTexel( gl_FragColor );",tT=`vec4 LinearTransferOETF( in vec4 value ) {
	return value;
}
vec4 sRGBTransferEOTF( in vec4 value ) {
	return vec4( mix( pow( value.rgb * 0.9478672986 + vec3( 0.0521327014 ), vec3( 2.4 ) ), value.rgb * 0.0773993808, vec3( lessThanEqual( value.rgb, vec3( 0.04045 ) ) ) ), value.a );
}
vec4 sRGBTransferOETF( in vec4 value ) {
	return vec4( mix( pow( value.rgb, vec3( 0.41666 ) ) * 1.055 - vec3( 0.055 ), value.rgb * 12.92, vec3( lessThanEqual( value.rgb, vec3( 0.0031308 ) ) ) ), value.a );
}`,nT=`#ifdef USE_ENVMAP
	#ifdef ENV_WORLDPOS
		vec3 cameraToFrag;
		if ( isOrthographic ) {
			cameraToFrag = normalize( vec3( - viewMatrix[ 0 ][ 2 ], - viewMatrix[ 1 ][ 2 ], - viewMatrix[ 2 ][ 2 ] ) );
		} else {
			cameraToFrag = normalize( vWorldPosition - cameraPosition );
		}
		vec3 worldNormal = transformNormalByInverseViewMatrix( normal, viewMatrix );
		#ifdef ENVMAP_MODE_REFLECTION
			vec3 reflectVec = reflect( cameraToFrag, worldNormal );
		#else
			vec3 reflectVec = refract( cameraToFrag, worldNormal, refractionRatio );
		#endif
	#else
		vec3 reflectVec = vReflect;
	#endif
	#ifdef ENVMAP_TYPE_CUBE
		vec4 envColor = textureCube( envMap, envMapRotation * reflectVec );
		#ifdef ENVMAP_BLENDING_MULTIPLY
			outgoingLight = mix( outgoingLight, outgoingLight * envColor.xyz, specularStrength * reflectivity );
		#elif defined( ENVMAP_BLENDING_MIX )
			outgoingLight = mix( outgoingLight, envColor.xyz, specularStrength * reflectivity );
		#elif defined( ENVMAP_BLENDING_ADD )
			outgoingLight += envColor.xyz * specularStrength * reflectivity;
		#endif
	#endif
#endif`,iT=`#ifdef USE_ENVMAP
	uniform float envMapIntensity;
	uniform mat3 envMapRotation;
	#ifdef ENVMAP_TYPE_CUBE
		uniform samplerCube envMap;
	#else
		uniform sampler2D envMap;
	#endif
#endif`,aT=`#ifdef USE_ENVMAP
	uniform float reflectivity;
	#if defined( USE_BUMPMAP ) || defined( USE_NORMALMAP ) || defined( PHONG ) || defined( LAMBERT )
		#define ENV_WORLDPOS
	#endif
	#ifdef ENV_WORLDPOS
		varying vec3 vWorldPosition;
		uniform float refractionRatio;
	#else
		varying vec3 vReflect;
	#endif
#endif`,sT=`#ifdef USE_ENVMAP
	#if defined( USE_BUMPMAP ) || defined( USE_NORMALMAP ) || defined( PHONG ) || defined( LAMBERT )
		#define ENV_WORLDPOS
	#endif
	#ifdef ENV_WORLDPOS
		
		varying vec3 vWorldPosition;
	#else
		varying vec3 vReflect;
		uniform float refractionRatio;
	#endif
#endif`,rT=`#ifdef USE_ENVMAP
	#ifdef ENV_WORLDPOS
		vWorldPosition = worldPosition.xyz;
	#else
		vec3 cameraToVertex;
		if ( isOrthographic ) {
			cameraToVertex = normalize( vec3( - viewMatrix[ 0 ][ 2 ], - viewMatrix[ 1 ][ 2 ], - viewMatrix[ 2 ][ 2 ] ) );
		} else {
			cameraToVertex = normalize( worldPosition.xyz - cameraPosition );
		}
		vec3 worldNormal = transformNormalByInverseViewMatrix( transformedNormal, viewMatrix );
		#ifdef ENVMAP_MODE_REFLECTION
			vReflect = reflect( cameraToVertex, worldNormal );
		#else
			vReflect = refract( cameraToVertex, worldNormal, refractionRatio );
		#endif
	#endif
#endif`,oT=`#ifdef USE_FOG
	vFogDepth = - mvPosition.z;
#endif`,lT=`#ifdef USE_FOG
	varying float vFogDepth;
#endif`,cT=`#ifdef USE_FOG
	#ifdef FOG_EXP2
		float fogFactor = 1.0 - exp( - fogDensity * fogDensity * vFogDepth * vFogDepth );
	#else
		float fogFactor = smoothstep( fogNear, fogFar, vFogDepth );
	#endif
	gl_FragColor.rgb = mix( gl_FragColor.rgb, fogColor, fogFactor );
#endif`,uT=`#ifdef USE_FOG
	uniform vec3 fogColor;
	varying float vFogDepth;
	#ifdef FOG_EXP2
		uniform float fogDensity;
	#else
		uniform float fogNear;
		uniform float fogFar;
	#endif
#endif`,fT=`#ifdef USE_GRADIENTMAP
	uniform sampler2D gradientMap;
#endif
vec3 getGradientIrradiance( vec3 normal, vec3 lightDirection ) {
	float dotNL = dot( normal, lightDirection );
	vec2 coord = vec2( dotNL * 0.5 + 0.5, 0.0 );
	#ifdef USE_GRADIENTMAP
		return vec3( texture2D( gradientMap, coord ).r );
	#else
		vec2 fw = fwidth( coord ) * 0.5;
		return mix( vec3( 0.7 ), vec3( 1.0 ), smoothstep( 0.7 - fw.x, 0.7 + fw.x, coord.x ) );
	#endif
}`,dT=`#ifdef USE_LIGHTMAP
	uniform sampler2D lightMap;
	uniform float lightMapIntensity;
#endif`,hT=`LambertMaterial material;
material.diffuseColor = diffuseColor.rgb;
material.specularStrength = specularStrength;`,pT=`varying vec3 vViewPosition;
struct LambertMaterial {
	vec3 diffuseColor;
	float specularStrength;
};
void RE_Direct_Lambert( const in IncidentLight directLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in LambertMaterial material, inout ReflectedLight reflectedLight ) {
	float dotNL = saturate( dot( geometryNormal, directLight.direction ) );
	vec3 irradiance = dotNL * directLight.color;
	reflectedLight.directDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
void RE_IndirectDiffuse_Lambert( const in vec3 irradiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in LambertMaterial material, inout ReflectedLight reflectedLight ) {
	reflectedLight.indirectDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
#define RE_Direct				RE_Direct_Lambert
#define RE_IndirectDiffuse		RE_IndirectDiffuse_Lambert`,mT=`uniform bool receiveShadow;
uniform vec3 ambientLightColor;
#if defined( USE_LIGHT_PROBES )
	uniform vec3 lightProbe[ 9 ];
#endif
vec3 shGetIrradianceAt( in vec3 normal, in vec3 shCoefficients[ 9 ] ) {
	float x = normal.x, y = normal.y, z = normal.z;
	vec3 result = shCoefficients[ 0 ] * 0.886227;
	result += shCoefficients[ 1 ] * 2.0 * 0.511664 * y;
	result += shCoefficients[ 2 ] * 2.0 * 0.511664 * z;
	result += shCoefficients[ 3 ] * 2.0 * 0.511664 * x;
	result += shCoefficients[ 4 ] * 2.0 * 0.429043 * x * y;
	result += shCoefficients[ 5 ] * 2.0 * 0.429043 * y * z;
	result += shCoefficients[ 6 ] * ( 0.743125 * z * z - 0.247708 );
	result += shCoefficients[ 7 ] * 2.0 * 0.429043 * x * z;
	result += shCoefficients[ 8 ] * 0.429043 * ( x * x - y * y );
	return result;
}
vec3 getLightProbeIrradiance( const in vec3 lightProbe[ 9 ], const in vec3 normal ) {
	vec3 worldNormal = transformNormalByInverseViewMatrix( normal, viewMatrix );
	vec3 irradiance = shGetIrradianceAt( worldNormal, lightProbe );
	return irradiance;
}
vec3 getAmbientLightIrradiance( const in vec3 ambientLightColor ) {
	vec3 irradiance = ambientLightColor;
	return irradiance;
}
float getDistanceAttenuation( const in float lightDistance, const in float cutoffDistance, const in float decayExponent ) {
	float distanceFalloff = 1.0 / max( pow( lightDistance, decayExponent ), 0.01 );
	if ( cutoffDistance > 0.0 ) {
		distanceFalloff *= pow2( saturate( 1.0 - pow4( lightDistance / cutoffDistance ) ) );
	}
	return distanceFalloff;
}
float getSpotAttenuation( const in float coneCosine, const in float penumbraCosine, const in float angleCosine ) {
	return smoothstep( coneCosine, penumbraCosine, angleCosine );
}
#if NUM_SUN_LIGHTS > 0
	struct SunLight {
		vec3 direction;
		vec3 color;
	};
	uniform SunLight sunLights[ NUM_SUN_LIGHTS ];
	void getSunLightInfo( const in SunLight sunLight, out IncidentLight light ) {
		light.color = sunLight.color;
		light.direction = sunLight.direction;
		light.visible = true;
	}
#endif
#if NUM_DIR_LIGHTS > 0
	struct DirectionalLight {
		vec3 direction;
		vec3 color;
	};
	uniform DirectionalLight directionalLights[ NUM_DIR_LIGHTS ];
	void getDirectionalLightInfo( const in DirectionalLight directionalLight, out IncidentLight light ) {
		light.color = directionalLight.color;
		light.direction = directionalLight.direction;
		light.visible = true;
	}
#endif
#if NUM_POINT_LIGHTS > 0
	struct PointLight {
		vec3 position;
		vec3 color;
		float distance;
		float decay;
	};
	uniform PointLight pointLights[ NUM_POINT_LIGHTS ];
	void getPointLightInfo( const in PointLight pointLight, const in vec3 geometryPosition, out IncidentLight light ) {
		vec3 lVector = pointLight.position - geometryPosition;
		light.direction = normalize( lVector );
		float lightDistance = length( lVector );
		light.color = pointLight.color;
		light.color *= getDistanceAttenuation( lightDistance, pointLight.distance, pointLight.decay );
		light.visible = ( light.color != vec3( 0.0 ) );
	}
#endif
#if NUM_SPOT_LIGHTS > 0
	struct SpotLight {
		vec3 position;
		vec3 direction;
		vec3 color;
		float distance;
		float decay;
		float coneCos;
		float penumbraCos;
	};
	uniform SpotLight spotLights[ NUM_SPOT_LIGHTS ];
	void getSpotLightInfo( const in SpotLight spotLight, const in vec3 geometryPosition, out IncidentLight light ) {
		vec3 lVector = spotLight.position - geometryPosition;
		light.direction = normalize( lVector );
		float angleCos = dot( light.direction, spotLight.direction );
		float spotAttenuation = getSpotAttenuation( spotLight.coneCos, spotLight.penumbraCos, angleCos );
		if ( spotAttenuation > 0.0 ) {
			float lightDistance = length( lVector );
			light.color = spotLight.color * spotAttenuation;
			light.color *= getDistanceAttenuation( lightDistance, spotLight.distance, spotLight.decay );
			light.visible = ( light.color != vec3( 0.0 ) );
		} else {
			light.color = vec3( 0.0 );
			light.visible = false;
		}
	}
#endif
#if NUM_RECT_AREA_LIGHTS > 0
	struct RectAreaLight {
		vec3 color;
		vec3 position;
		vec3 halfWidth;
		vec3 halfHeight;
	};
	uniform sampler2D ltc_1;	uniform sampler2D ltc_2;
	uniform RectAreaLight rectAreaLights[ NUM_RECT_AREA_LIGHTS ];
#endif
#if NUM_HEMI_LIGHTS > 0
	struct HemisphereLight {
		vec3 direction;
		vec3 skyColor;
		vec3 groundColor;
	};
	uniform HemisphereLight hemisphereLights[ NUM_HEMI_LIGHTS ];
	vec3 getHemisphereLightIrradiance( const in HemisphereLight hemiLight, const in vec3 normal ) {
		float dotNL = dot( normal, hemiLight.direction );
		float hemiDiffuseWeight = 0.5 * dotNL + 0.5;
		vec3 irradiance = mix( hemiLight.groundColor, hemiLight.skyColor, hemiDiffuseWeight );
		return irradiance;
	}
#endif
#include <lightprobes_pars_fragment>`,gT=`#ifdef USE_ENVMAP
	vec3 getIBLIrradiance( const in vec3 normal ) {
		#ifdef ENVMAP_TYPE_CUBE_UV
			vec3 worldNormal = transformNormalByInverseViewMatrix( normal, viewMatrix );
			vec4 envMapColor = textureCubeUV( envMap, envMapRotation * worldNormal, 1.0 );
			return PI * envMapColor.rgb * envMapIntensity;
		#else
			return vec3( 0.0 );
		#endif
	}
	vec3 getIBLRadiance( const in vec3 viewDir, const in vec3 normal, const in float roughness ) {
		#ifdef ENVMAP_TYPE_CUBE_UV
			vec3 reflectVec = reflect( - viewDir, normal );
			reflectVec = normalize( mix( reflectVec, normal, pow4( roughness ) ) );
			reflectVec = transformDirectionByInverseViewMatrix( reflectVec, viewMatrix );
			vec4 envMapColor = textureCubeUV( envMap, envMapRotation * reflectVec, roughness );
			return envMapColor.rgb * envMapIntensity;
		#else
			return vec3( 0.0 );
		#endif
	}
	#ifdef USE_RETROREFLECTION
		vec3 getIBLRetroRadiance( const in vec3 viewDir, const in vec3 normal, const in float roughness ) {
			#ifdef ENVMAP_TYPE_CUBE_UV
				vec3 retroVec = normalize( mix( viewDir, normal, pow4( roughness ) ) );
				retroVec = transformDirectionByInverseViewMatrix( retroVec, viewMatrix );
				vec4 envMapColor = textureCubeUV( envMap, envMapRotation * retroVec, roughness );
				return envMapColor.rgb * envMapIntensity;
			#else
				return vec3( 0.0 );
			#endif
		}
	#endif
	#ifdef USE_ANISOTROPY
		vec3 getIBLAnisotropyRadiance( const in vec3 viewDir, const in vec3 normal, const in float roughness, const in vec3 bitangent, const in float anisotropy ) {
			#ifdef ENVMAP_TYPE_CUBE_UV
				vec3 bentNormal = cross( bitangent, viewDir );
				bentNormal = normalize( cross( bentNormal, bitangent ) );
				bentNormal = normalize( mix( bentNormal, normal, pow2( pow2( 1.0 - anisotropy * ( 1.0 - roughness ) ) ) ) );
				return getIBLRadiance( viewDir, bentNormal, roughness );
			#else
				return vec3( 0.0 );
			#endif
		}
		#ifdef USE_RETROREFLECTION
			vec3 getIBLAnisotropyRetroRadiance( const in vec3 viewDir, const in vec3 normal, const in float roughness, const in vec3 bitangent, const in float anisotropy ) {
				#ifdef ENVMAP_TYPE_CUBE_UV
					vec3 bentNormal = cross( bitangent, viewDir );
					bentNormal = normalize( cross( bentNormal, bitangent ) );
					bentNormal = normalize( mix( bentNormal, normal, pow2( pow2( 1.0 - anisotropy * ( 1.0 - roughness ) ) ) ) );
					return getIBLRetroRadiance( viewDir, bentNormal, roughness );
				#else
					return vec3( 0.0 );
				#endif
			}
		#endif
	#endif
#endif`,_T=`ToonMaterial material;
material.diffuseColor = diffuseColor.rgb;`,vT=`varying vec3 vViewPosition;
struct ToonMaterial {
	vec3 diffuseColor;
};
void RE_Direct_Toon( const in IncidentLight directLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in ToonMaterial material, inout ReflectedLight reflectedLight ) {
	vec3 irradiance = getGradientIrradiance( geometryNormal, directLight.direction ) * directLight.color;
	reflectedLight.directDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
void RE_IndirectDiffuse_Toon( const in vec3 irradiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in ToonMaterial material, inout ReflectedLight reflectedLight ) {
	reflectedLight.indirectDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
#define RE_Direct				RE_Direct_Toon
#define RE_IndirectDiffuse		RE_IndirectDiffuse_Toon`,xT=`BlinnPhongMaterial material;
material.diffuseColor = diffuseColor.rgb;
material.specularColor = specular;
material.specularShininess = shininess;
material.specularStrength = specularStrength;`,ST=`varying vec3 vViewPosition;
struct BlinnPhongMaterial {
	vec3 diffuseColor;
	vec3 specularColor;
	float specularShininess;
	float specularStrength;
};
void RE_Direct_BlinnPhong( const in IncidentLight directLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in BlinnPhongMaterial material, inout ReflectedLight reflectedLight ) {
	float dotNL = saturate( dot( geometryNormal, directLight.direction ) );
	vec3 irradiance = dotNL * directLight.color;
	reflectedLight.directDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
	reflectedLight.directSpecular += irradiance * BRDF_BlinnPhong( directLight.direction, geometryViewDir, geometryNormal, material.specularColor, material.specularShininess ) * material.specularStrength;
}
void RE_IndirectDiffuse_BlinnPhong( const in vec3 irradiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in BlinnPhongMaterial material, inout ReflectedLight reflectedLight ) {
	reflectedLight.indirectDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
#define RE_Direct				RE_Direct_BlinnPhong
#define RE_IndirectDiffuse		RE_IndirectDiffuse_BlinnPhong`,yT=`PhysicalMaterial material;
material.diffuseColor = diffuseColor.rgb;
material.diffuseContribution = diffuseColor.rgb * ( 1.0 - metalnessFactor );
material.metalness = metalnessFactor;
vec3 dxy = max( abs( dFdx( nonPerturbedNormal ) ), abs( dFdy( nonPerturbedNormal ) ) );
float geometryRoughness = max( max( dxy.x, dxy.y ), dxy.z );
material.roughness = max( roughnessFactor, 0.0525 );material.roughness += geometryRoughness;
material.roughness = min( material.roughness, 1.0 );
#ifdef IOR
	material.ior = ior;
	#ifdef USE_SPECULAR
		float specularIntensityFactor = specularIntensity;
		vec3 specularColorFactor = specularColor;
		#ifdef USE_SPECULAR_COLORMAP
			specularColorFactor *= texture2D( specularColorMap, vSpecularColorMapUv ).rgb;
		#endif
		#ifdef USE_SPECULAR_INTENSITYMAP
			specularIntensityFactor *= texture2D( specularIntensityMap, vSpecularIntensityMapUv ).a;
		#endif
		material.specularF90 = mix( specularIntensityFactor, 1.0, metalnessFactor );
	#else
		float specularIntensityFactor = 1.0;
		vec3 specularColorFactor = vec3( 1.0 );
		material.specularF90 = 1.0;
	#endif
	material.specularColor = min( pow2( ( material.ior - 1.0 ) / ( material.ior + 1.0 ) ) * specularColorFactor, vec3( 1.0 ) ) * specularIntensityFactor;
	material.specularColorBlended = mix( material.specularColor, diffuseColor.rgb, metalnessFactor );
#else
	material.specularColor = vec3( 0.04 );
	material.specularColorBlended = mix( material.specularColor, diffuseColor.rgb, metalnessFactor );
	material.specularF90 = 1.0;
#endif
#ifdef USE_CLEARCOAT
	material.clearcoat = clearcoat;
	material.clearcoatRoughness = clearcoatRoughness;
	material.clearcoatF0 = vec3( 0.04 );
	material.clearcoatF90 = 1.0;
	#ifdef USE_CLEARCOATMAP
		material.clearcoat *= texture2D( clearcoatMap, vClearcoatMapUv ).x;
	#endif
	#ifdef USE_CLEARCOAT_ROUGHNESSMAP
		material.clearcoatRoughness *= texture2D( clearcoatRoughnessMap, vClearcoatRoughnessMapUv ).y;
	#endif
	material.clearcoat = saturate( material.clearcoat );	material.clearcoatRoughness = max( material.clearcoatRoughness, 0.0525 );
	material.clearcoatRoughness += geometryRoughness;
	material.clearcoatRoughness = min( material.clearcoatRoughness, 1.0 );
#endif
#ifdef USE_DISPERSION
	material.dispersion = dispersion;
#endif
#ifdef USE_RETROREFLECTION
	material.retroreflectivity = retroreflectivity;
#endif
#ifdef USE_IRIDESCENCE
	material.iridescence = iridescence;
	material.iridescenceIOR = iridescenceIOR;
	#ifdef USE_IRIDESCENCEMAP
		material.iridescence *= texture2D( iridescenceMap, vIridescenceMapUv ).r;
	#endif
	#ifdef USE_IRIDESCENCE_THICKNESSMAP
		material.iridescenceThickness = (iridescenceThicknessMaximum - iridescenceThicknessMinimum) * texture2D( iridescenceThicknessMap, vIridescenceThicknessMapUv ).g + iridescenceThicknessMinimum;
	#else
		material.iridescenceThickness = iridescenceThicknessMaximum;
	#endif
#endif
#ifdef USE_SHEEN
	material.sheenColor = sheenColor;
	#ifdef USE_SHEEN_COLORMAP
		material.sheenColor *= texture2D( sheenColorMap, vSheenColorMapUv ).rgb;
	#endif
	material.sheenRoughness = clamp( sheenRoughness, 0.0001, 1.0 );
	#ifdef USE_SHEEN_ROUGHNESSMAP
		material.sheenRoughness *= texture2D( sheenRoughnessMap, vSheenRoughnessMapUv ).a;
	#endif
#endif
#ifdef USE_ANISOTROPY
	#ifdef USE_ANISOTROPYMAP
		mat2 anisotropyMat = mat2( anisotropyVector.x, anisotropyVector.y, - anisotropyVector.y, anisotropyVector.x );
		vec3 anisotropyPolar = texture2D( anisotropyMap, vAnisotropyMapUv ).rgb;
		vec2 anisotropyV = anisotropyMat * normalize( 2.0 * anisotropyPolar.rg - vec2( 1.0 ) ) * anisotropyPolar.b;
	#else
		vec2 anisotropyV = anisotropyVector;
	#endif
	material.anisotropy = length( anisotropyV );
	if( material.anisotropy == 0.0 ) {
		anisotropyV = vec2( 1.0, 0.0 );
	} else {
		anisotropyV /= material.anisotropy;
		material.anisotropy = saturate( material.anisotropy );
	}
	material.alphaT = mix( pow2( material.roughness ), 1.0, pow2( material.anisotropy ) );
	material.anisotropyT = tbn[ 0 ] * anisotropyV.x + tbn[ 1 ] * anisotropyV.y;
	material.anisotropyB = tbn[ 1 ] * anisotropyV.x - tbn[ 0 ] * anisotropyV.y;
#endif`,MT=`uniform sampler2D dfgLUT;
struct PhysicalMaterial {
	vec3 diffuseColor;
	vec3 diffuseContribution;
	vec3 specularColor;
	vec3 specularColorBlended;
	float roughness;
	float metalness;
	float specularF90;
	float dispersion;
	vec2 dfg;
	vec3 multiScatteringCompensation;
	#ifdef USE_RETROREFLECTION
		float retroreflectivity;
	#endif
	#ifdef USE_CLEARCOAT
		float clearcoat;
		float clearcoatRoughness;
		vec3 clearcoatF0;
		float clearcoatF90;
	#endif
	#ifdef USE_IRIDESCENCE
		float iridescence;
		float iridescenceIOR;
		float iridescenceThickness;
		vec3 iridescenceFresnel;
		vec3 iridescenceF0Dielectric;
		vec3 iridescenceF0Metallic;
	#endif
	#ifdef USE_SHEEN
		vec3 sheenColor;
		float sheenRoughness;
	#endif
	#ifdef IOR
		float ior;
	#endif
	#ifdef USE_TRANSMISSION
		float transmission;
		float transmissionAlpha;
		float thickness;
		float attenuationDistance;
		vec3 attenuationColor;
	#endif
	#ifdef USE_ANISOTROPY
		float anisotropy;
		float alphaT;
		vec3 anisotropyT;
		vec3 anisotropyB;
	#endif
};
vec3 clearcoatSpecularDirect = vec3( 0.0 );
vec3 clearcoatSpecularIndirect = vec3( 0.0 );
vec3 sheenSpecularDirect = vec3( 0.0 );
vec3 sheenSpecularIndirect = vec3(0.0 );
vec3 Schlick_to_F0( const in vec3 f, const in float f90, const in float dotVH ) {
    float x = clamp( 1.0 - dotVH, 0.0, 1.0 );
    float x2 = x * x;
    float x5 = clamp( x * x2 * x2, 0.0, 0.9999 );
    return ( f - vec3( f90 ) * x5 ) / ( 1.0 - x5 );
}
float V_GGX_SmithCorrelated( const in float alpha, const in float dotNL, const in float dotNV ) {
	float a2 = pow2( alpha );
	float gv = dotNL * sqrt( a2 + ( 1.0 - a2 ) * pow2( dotNV ) );
	float gl = dotNV * sqrt( a2 + ( 1.0 - a2 ) * pow2( dotNL ) );
	return 0.5 / max( gv + gl, EPSILON );
}
float D_GGX( const in float alpha, const in float dotNH ) {
	float a2 = pow2( alpha );
	float denom = pow2( dotNH ) * ( a2 - 1.0 ) + 1.0;
	return RECIPROCAL_PI * a2 / pow2( denom );
}
#ifdef USE_ANISOTROPY
	float V_GGX_SmithCorrelated_Anisotropic( const in float alphaT, const in float alphaB, const in float dotTV, const in float dotBV, const in float dotTL, const in float dotBL, const in float dotNV, const in float dotNL ) {
		float gv = dotNL * length( vec3( alphaT * dotTV, alphaB * dotBV, dotNV ) );
		float gl = dotNV * length( vec3( alphaT * dotTL, alphaB * dotBL, dotNL ) );
		return 0.5 / max( gv + gl, EPSILON );
	}
	float D_GGX_Anisotropic( const in float alphaT, const in float alphaB, const in float dotNH, const in float dotTH, const in float dotBH ) {
		float a2 = alphaT * alphaB;
		highp vec3 v = vec3( alphaB * dotTH, alphaT * dotBH, a2 * dotNH );
		highp float v2 = dot( v, v );
		float w2 = a2 / v2;
		return RECIPROCAL_PI * a2 * pow2 ( w2 );
	}
#endif
#ifdef USE_CLEARCOAT
	vec3 BRDF_GGX_Clearcoat( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, const in PhysicalMaterial material) {
		vec3 f0 = material.clearcoatF0;
		float f90 = material.clearcoatF90;
		float roughness = material.clearcoatRoughness;
		float alpha = pow2( roughness );
		vec3 halfDir = normalize( lightDir + viewDir );
		float dotNL = saturate( dot( normal, lightDir ) );
		float dotNV = saturate( dot( normal, viewDir ) );
		float dotNH = saturate( dot( normal, halfDir ) );
		float dotVH = saturate( dot( viewDir, halfDir ) );
		vec3 F = F_Schlick( f0, f90, dotVH );
		float V = V_GGX_SmithCorrelated( alpha, dotNL, dotNV );
		float D = D_GGX( alpha, dotNH );
		return F * ( V * D );
	}
#endif
vec3 BRDF_GGX( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, const in PhysicalMaterial material ) {
	vec3 f0 = material.specularColorBlended;
	float f90 = material.specularF90;
	float roughness = material.roughness;
	float alpha = pow2( roughness );
	vec3 halfDir = normalize( lightDir + viewDir );
	float dotNL = saturate( dot( normal, lightDir ) );
	float dotNV = saturate( dot( normal, viewDir ) );
	float dotNH = saturate( dot( normal, halfDir ) );
	float dotVH = saturate( dot( viewDir, halfDir ) );
	vec3 F = F_Schlick( f0, f90, dotVH );
	#ifdef USE_IRIDESCENCE
		F = mix( F, material.iridescenceFresnel, material.iridescence );
	#endif
	#ifdef USE_ANISOTROPY
		float dotTL = dot( material.anisotropyT, lightDir );
		float dotTV = dot( material.anisotropyT, viewDir );
		float dotTH = dot( material.anisotropyT, halfDir );
		float dotBL = dot( material.anisotropyB, lightDir );
		float dotBV = dot( material.anisotropyB, viewDir );
		float dotBH = dot( material.anisotropyB, halfDir );
		float V = V_GGX_SmithCorrelated_Anisotropic( material.alphaT, alpha, dotTV, dotBV, dotTL, dotBL, dotNV, dotNL );
		float D = D_GGX_Anisotropic( material.alphaT, alpha, dotNH, dotTH, dotBH );
	#else
		float V = V_GGX_SmithCorrelated( alpha, dotNL, dotNV );
		float D = D_GGX( alpha, dotNH );
	#endif
	return F * ( V * D );
}
vec2 LTC_Uv( const in vec3 N, const in vec3 V, const in float roughness ) {
	const float LUT_SIZE = 64.0;
	const float LUT_SCALE = ( LUT_SIZE - 1.0 ) / LUT_SIZE;
	const float LUT_BIAS = 0.5 / LUT_SIZE;
	float dotNV = saturate( dot( N, V ) );
	vec2 uv = vec2( roughness, sqrt( 1.0 - dotNV ) );
	uv = uv * LUT_SCALE + LUT_BIAS;
	return uv;
}
float LTC_ClippedSphereFormFactor( const in vec3 f ) {
	float l = length( f );
	return max( ( l * l + f.z ) / ( l + 1.0 ), 0.0 );
}
vec3 LTC_EdgeVectorFormFactor( const in vec3 v1, const in vec3 v2 ) {
	float x = dot( v1, v2 );
	float y = abs( x );
	float a = 0.8543985 + ( 0.4965155 + 0.0145206 * y ) * y;
	float b = 3.4175940 + ( 4.1616724 + y ) * y;
	float v = a / b;
	float theta_sintheta = ( x > 0.0 ) ? v : 0.5 * inversesqrt( max( 1.0 - x * x, 1e-7 ) ) - v;
	return cross( v1, v2 ) * theta_sintheta;
}
vec3 LTC_Evaluate( const in vec3 N, const in vec3 V, const in vec3 P, const in mat3 mInv, const in vec3 rectCoords[ 4 ] ) {
	vec3 v1 = rectCoords[ 1 ] - rectCoords[ 0 ];
	vec3 v2 = rectCoords[ 3 ] - rectCoords[ 0 ];
	vec3 lightNormal = cross( v1, v2 );
	if( dot( lightNormal, P - rectCoords[ 0 ] ) < 0.0 ) return vec3( 0.0 );
	vec3 T1, T2;
	T1 = normalize( V - N * dot( V, N ) );
	T2 = - cross( N, T1 );
	mat3 mat = mInv * transpose( mat3( T1, T2, N ) );
	vec3 coords[ 4 ];
	coords[ 0 ] = mat * ( rectCoords[ 0 ] - P );
	coords[ 1 ] = mat * ( rectCoords[ 1 ] - P );
	coords[ 2 ] = mat * ( rectCoords[ 2 ] - P );
	coords[ 3 ] = mat * ( rectCoords[ 3 ] - P );
	coords[ 0 ] = normalize( coords[ 0 ] );
	coords[ 1 ] = normalize( coords[ 1 ] );
	coords[ 2 ] = normalize( coords[ 2 ] );
	coords[ 3 ] = normalize( coords[ 3 ] );
	vec3 vectorFormFactor = vec3( 0.0 );
	vectorFormFactor += LTC_EdgeVectorFormFactor( coords[ 0 ], coords[ 1 ] );
	vectorFormFactor += LTC_EdgeVectorFormFactor( coords[ 1 ], coords[ 2 ] );
	vectorFormFactor += LTC_EdgeVectorFormFactor( coords[ 2 ], coords[ 3 ] );
	vectorFormFactor += LTC_EdgeVectorFormFactor( coords[ 3 ], coords[ 0 ] );
	float result = LTC_ClippedSphereFormFactor( vectorFormFactor );
	return vec3( result );
}
#if defined( USE_SHEEN )
float D_Charlie( float roughness, float dotNH ) {
	float alpha = pow2( roughness );
	float invAlpha = 1.0 / alpha;
	float cos2h = dotNH * dotNH;
	float sin2h = max( 1.0 - cos2h, 0.0078125 );
	return ( 2.0 + invAlpha ) * pow( sin2h, invAlpha * 0.5 ) / ( 2.0 * PI );
}
float V_Neubelt( float dotNV, float dotNL ) {
	return saturate( 1.0 / ( 4.0 * ( dotNL + dotNV - dotNL * dotNV ) ) );
}
vec3 BRDF_Sheen( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, vec3 sheenColor, const in float sheenRoughness ) {
	vec3 halfDir = normalize( lightDir + viewDir );
	float dotNL = saturate( dot( normal, lightDir ) );
	float dotNV = saturate( dot( normal, viewDir ) );
	float dotNH = saturate( dot( normal, halfDir ) );
	float D = D_Charlie( sheenRoughness, dotNH );
	float V = V_Neubelt( dotNV, dotNL );
	return sheenColor * ( D * V );
}
#endif
float IBLSheenBRDF( const in vec3 normal, const in vec3 viewDir, const in float roughness ) {
	float dotNV = saturate( dot( normal, viewDir ) );
	float r2 = roughness * roughness;
	float rInv = 1.0 / ( roughness + 0.1 );
	float a = -1.9362 + 1.0678 * roughness + 0.4573 * r2 - 0.8469 * rInv;
	float b = -0.6014 + 0.5538 * roughness - 0.4670 * r2 - 0.1255 * rInv;
	float DG = exp( a * dotNV + b );
	return saturate( DG );
}
vec3 EnvironmentBRDF( const in vec3 normal, const in vec3 viewDir, const in vec3 specularColor, const in float specularF90, const in float roughness ) {
	float dotNV = saturate( dot( normal, viewDir ) );
	vec2 fab = texture2D( dfgLUT, vec2( roughness, dotNV ) ).rg;
	return specularColor * fab.x + specularF90 * fab.y;
}
#ifdef USE_IRIDESCENCE
void computeMultiscatteringIridescence( const in vec2 fab, const in vec3 specularColor, const in float specularF90, const in float iridescence, const in vec3 iridescenceF0, inout vec3 singleScatter, inout vec3 multiScatter ) {
#else
void computeMultiscattering( const in vec2 fab, const in vec3 specularColor, const in float specularF90, inout vec3 singleScatter, inout vec3 multiScatter ) {
#endif
	#ifdef USE_IRIDESCENCE
		vec3 Fr = mix( specularColor, iridescenceF0, iridescence );
	#else
		vec3 Fr = specularColor;
	#endif
	vec3 FssEss = Fr * fab.x + specularF90 * fab.y;
	float Ess = fab.x + fab.y;
	float Ems = 1.0 - Ess;
	vec3 Favg = Fr + ( 1.0 - Fr ) * 0.047619;	vec3 Fms = FssEss * Favg / ( 1.0 - Ems * Favg );
	singleScatter += FssEss;
	multiScatter += Fms * Ems;
}
#if NUM_RECT_AREA_LIGHTS > 0
	void RE_Direct_RectArea_Physical( const in RectAreaLight rectAreaLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in PhysicalMaterial material, inout ReflectedLight reflectedLight ) {
		vec3 normal = geometryNormal;
		vec3 viewDir = geometryViewDir;
		vec3 position = geometryPosition;
		vec3 lightPos = rectAreaLight.position;
		vec3 halfWidth = rectAreaLight.halfWidth;
		vec3 halfHeight = rectAreaLight.halfHeight;
		vec3 lightColor = rectAreaLight.color;
		float roughness = material.roughness;
		vec3 rectCoords[ 4 ];
		rectCoords[ 0 ] = lightPos + halfWidth - halfHeight;		rectCoords[ 1 ] = lightPos - halfWidth - halfHeight;
		rectCoords[ 2 ] = lightPos - halfWidth + halfHeight;
		rectCoords[ 3 ] = lightPos + halfWidth + halfHeight;
		vec2 uv = LTC_Uv( normal, viewDir, roughness );
		vec4 t1 = texture2D( ltc_1, uv );
		vec4 t2 = texture2D( ltc_2, uv );
		mat3 mInv = mat3(
			vec3( t1.x, 0, t1.y ),
			vec3(    0, 1,    0 ),
			vec3( t1.z, 0, t1.w )
		);
		vec3 fresnel = ( material.specularColorBlended * t2.x + ( material.specularF90 - material.specularColorBlended ) * t2.y );
		reflectedLight.directSpecular += lightColor * fresnel * LTC_Evaluate( normal, viewDir, position, mInv, rectCoords );
		reflectedLight.directDiffuse += lightColor * material.diffuseContribution * LTC_Evaluate( normal, viewDir, position, mat3( 1.0 ), rectCoords );
		#ifdef USE_CLEARCOAT
			vec3 Ncc = geometryClearcoatNormal;
			vec2 uvClearcoat = LTC_Uv( Ncc, viewDir, material.clearcoatRoughness );
			vec4 t1Clearcoat = texture2D( ltc_1, uvClearcoat );
			vec4 t2Clearcoat = texture2D( ltc_2, uvClearcoat );
			mat3 mInvClearcoat = mat3(
				vec3( t1Clearcoat.x, 0, t1Clearcoat.y ),
				vec3(             0, 1,             0 ),
				vec3( t1Clearcoat.z, 0, t1Clearcoat.w )
			);
			vec3 fresnelClearcoat = material.clearcoatF0 * t2Clearcoat.x + ( material.clearcoatF90 - material.clearcoatF0 ) * t2Clearcoat.y;
			clearcoatSpecularDirect += lightColor * fresnelClearcoat * LTC_Evaluate( Ncc, viewDir, position, mInvClearcoat, rectCoords );
		#endif
	}
#endif
void RE_Direct_Physical( const in IncidentLight directLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in PhysicalMaterial material, inout ReflectedLight reflectedLight ) {
	float dotNL = saturate( dot( geometryNormal, directLight.direction ) );
	vec3 irradiance = dotNL * directLight.color;
	#ifdef USE_CLEARCOAT
		float dotNLcc = saturate( dot( geometryClearcoatNormal, directLight.direction ) );
		vec3 ccIrradiance = dotNLcc * directLight.color;
		clearcoatSpecularDirect += ccIrradiance * BRDF_GGX_Clearcoat( directLight.direction, geometryViewDir, geometryClearcoatNormal, material );
	#endif
	#ifdef USE_SHEEN
 
 		sheenSpecularDirect += irradiance * BRDF_Sheen( directLight.direction, geometryViewDir, geometryNormal, material.sheenColor, material.sheenRoughness );
 
 		float sheenAlbedoV = IBLSheenBRDF( geometryNormal, geometryViewDir, material.sheenRoughness );
 		float sheenAlbedoL = IBLSheenBRDF( geometryNormal, directLight.direction, material.sheenRoughness );
 
 		float sheenEnergyComp = 1.0 - max3( material.sheenColor ) * max( sheenAlbedoV, sheenAlbedoL );
 
 		irradiance *= sheenEnergyComp;
 
 	#endif
	vec3 specularBRDF = BRDF_GGX( directLight.direction, geometryViewDir, geometryNormal, material );
	#ifdef USE_RETROREFLECTION
		vec3 retroViewDir = reflect( - geometryViewDir, geometryNormal );
		vec3 retroSpecularBRDF = BRDF_GGX( directLight.direction, retroViewDir, geometryNormal, material );
		specularBRDF = mix( specularBRDF, retroSpecularBRDF, saturate( material.retroreflectivity ) );
	#endif
	reflectedLight.directSpecular += irradiance * specularBRDF * material.multiScatteringCompensation;
	vec3 halfDir = normalize( directLight.direction + geometryViewDir );
	float dotVH = saturate( dot( geometryViewDir, halfDir ) );
	vec3 F = F_Schlick( material.specularColor, material.specularF90, dotVH );
	#ifdef USE_RETROREFLECTION
		vec3 retroHalfDir = normalize( directLight.direction + retroViewDir );
		float dotRetroVH = saturate( dot( retroViewDir, retroHalfDir ) );
		vec3 retroF = F_Schlick( material.specularColor, material.specularF90, dotRetroVH );
		F = mix( F, retroF, saturate( material.retroreflectivity ) );
	#endif
	reflectedLight.directDiffuse += irradiance * BRDF_Lambert( material.diffuseContribution ) * ( 1.0 - F );
}
void RE_IndirectDiffuse_Physical( const in vec3 irradiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in PhysicalMaterial material, inout ReflectedLight reflectedLight ) {
	vec3 singleScattering = vec3( 0.0 );
	vec3 multiScattering = vec3( 0.0 );
	#ifdef USE_IRIDESCENCE
		computeMultiscatteringIridescence( material.dfg, material.specularColor, material.specularF90, material.iridescence, material.iridescenceF0Dielectric, singleScattering, multiScattering );
	#else
		computeMultiscattering( material.dfg, material.specularColor, material.specularF90, singleScattering, multiScattering );
	#endif
	vec3 diffuse = irradiance * BRDF_Lambert( material.diffuseContribution ) * ( 1.0 - singleScattering - multiScattering );
	#ifdef USE_SHEEN
		float sheenAlbedo = IBLSheenBRDF( geometryNormal, geometryViewDir, material.sheenRoughness );
		sheenSpecularIndirect += irradiance * material.sheenColor * sheenAlbedo * RECIPROCAL_PI;
		float sheenEnergyComp = 1.0 - max3( material.sheenColor ) * sheenAlbedo;
		diffuse *= sheenEnergyComp;
	#endif
	reflectedLight.indirectDiffuse += diffuse;
}
void RE_IndirectSpecular_Physical( const in vec3 radiance, const in vec3 irradiance, const in vec3 clearcoatRadiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in PhysicalMaterial material, inout ReflectedLight reflectedLight) {
	#ifdef USE_CLEARCOAT
		clearcoatSpecularIndirect += clearcoatRadiance * EnvironmentBRDF( geometryClearcoatNormal, geometryViewDir, material.clearcoatF0, material.clearcoatF90, material.clearcoatRoughness );
	#endif
	#ifdef USE_SHEEN
		sheenSpecularIndirect += irradiance * material.sheenColor * IBLSheenBRDF( geometryNormal, geometryViewDir, material.sheenRoughness ) * RECIPROCAL_PI;
 	#endif
	vec3 singleScatteringDielectric = vec3( 0.0 );
	vec3 multiScatteringDielectric = vec3( 0.0 );
	vec3 singleScatteringMetallic = vec3( 0.0 );
	vec3 multiScatteringMetallic = vec3( 0.0 );
	#ifdef USE_IRIDESCENCE
		computeMultiscatteringIridescence( material.dfg, material.specularColor, material.specularF90, material.iridescence, material.iridescenceF0Dielectric, singleScatteringDielectric, multiScatteringDielectric );
		computeMultiscatteringIridescence( material.dfg, material.diffuseColor, material.specularF90, material.iridescence, material.iridescenceF0Metallic, singleScatteringMetallic, multiScatteringMetallic );
	#else
		computeMultiscattering( material.dfg, material.specularColor, material.specularF90, singleScatteringDielectric, multiScatteringDielectric );
		computeMultiscattering( material.dfg, material.diffuseColor, material.specularF90, singleScatteringMetallic, multiScatteringMetallic );
	#endif
	vec3 singleScattering = mix( singleScatteringDielectric, singleScatteringMetallic, material.metalness );
	vec3 multiScattering = mix( multiScatteringDielectric, multiScatteringMetallic, material.metalness );
	vec3 totalScatteringDielectric = singleScatteringDielectric + multiScatteringDielectric;
	vec3 diffuse = material.diffuseContribution * ( 1.0 - totalScatteringDielectric );
	vec3 cosineWeightedIrradiance = irradiance * RECIPROCAL_PI;
	vec3 indirectSpecular = radiance * singleScattering;
	indirectSpecular += multiScattering * cosineWeightedIrradiance;
	vec3 indirectDiffuse = diffuse * cosineWeightedIrradiance;
	#ifdef USE_SHEEN
		float sheenAlbedo = IBLSheenBRDF( geometryNormal, geometryViewDir, material.sheenRoughness );
		float sheenEnergyComp = 1.0 - max3( material.sheenColor ) * sheenAlbedo;
		indirectSpecular *= sheenEnergyComp;
		indirectDiffuse *= sheenEnergyComp;
	#endif
	reflectedLight.indirectSpecular += indirectSpecular;
	reflectedLight.indirectDiffuse += indirectDiffuse;
}
#define RE_Direct				RE_Direct_Physical
#define RE_Direct_RectArea		RE_Direct_RectArea_Physical
#define RE_IndirectDiffuse		RE_IndirectDiffuse_Physical
#define RE_IndirectSpecular		RE_IndirectSpecular_Physical
float computeSpecularOcclusion( const in float dotNV, const in float ambientOcclusion, const in float roughness ) {
	return saturate( pow( dotNV + ambientOcclusion, exp2( - 16.0 * roughness - 1.0 ) ) - 1.0 + ambientOcclusion );
}`,ET=`
vec3 geometryPosition = - vViewPosition;
vec3 geometryNormal = normal;
vec3 geometryViewDir = ( isOrthographic ) ? vec3( 0, 0, 1 ) : normalize( vViewPosition );
vec3 geometryClearcoatNormal = vec3( 0.0 );
#ifdef USE_CLEARCOAT
	geometryClearcoatNormal = clearcoatNormal;
#endif
#ifdef USE_IRIDESCENCE
	float dotNVi = saturate( dot( normal, geometryViewDir ) );
	if ( material.iridescenceThickness == 0.0 ) {
		material.iridescence = 0.0;
	} else {
		material.iridescence = saturate( material.iridescence );
	}
	if ( material.iridescence > 0.0 ) {
		vec3 iridescenceFresnelDielectric = evalIridescence( 1.0, material.iridescenceIOR, dotNVi, material.iridescenceThickness, material.specularColor );
		vec3 iridescenceFresnelMetallic = evalIridescence( 1.0, material.iridescenceIOR, dotNVi, material.iridescenceThickness, material.diffuseColor );
		material.iridescenceFresnel = mix( iridescenceFresnelDielectric, iridescenceFresnelMetallic, material.metalness );
		material.iridescenceF0Dielectric = Schlick_to_F0( iridescenceFresnelDielectric, 1.0, dotNVi );
		material.iridescenceF0Metallic = Schlick_to_F0( iridescenceFresnelMetallic, 1.0, dotNVi );
	}
#endif
#ifdef STANDARD
	float dotNVms = saturate( dot( geometryNormal, geometryViewDir ) );
	material.dfg = texture2D( dfgLUT, vec2( material.roughness, dotNVms ) ).rg;
	#if ( NUM_SUN_LIGHTS > 0 || NUM_DIR_LIGHTS > 0 || NUM_POINT_LIGHTS > 0 || NUM_SPOT_LIGHTS > 0 )
		float EssMs = material.dfg.x + material.dfg.y;
		material.multiScatteringCompensation = 1.0 + material.specularColorBlended * ( 1.0 / EssMs - 1.0 );
	#endif
#endif
IncidentLight directLight;
#if ( NUM_POINT_LIGHTS > 0 ) && defined( RE_Direct )
	PointLight pointLight;
	#if defined( USE_SHADOWMAP ) && NUM_POINT_LIGHT_SHADOWS > 0
	PointLightShadow pointLightShadow;
	#endif
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_POINT_LIGHTS; i ++ ) {
		pointLight = pointLights[ i ];
		getPointLightInfo( pointLight, geometryPosition, directLight );
		#if defined( USE_SHADOWMAP ) && ( UNROLLED_LOOP_INDEX < NUM_POINT_LIGHT_SHADOWS ) && ( defined( SHADOWMAP_TYPE_PCF ) || defined( SHADOWMAP_TYPE_BASIC ) )
		pointLightShadow = pointLightShadows[ i ];
		directLight.color *= ( directLight.visible && receiveShadow ) ? getPointShadow( pointShadowMap[ i ], pointLightShadow.shadowMapSize, pointLightShadow.shadowIntensity, pointLightShadow.shadowBias, pointLightShadow.shadowRadius, vPointShadowCoord[ i ], pointLightShadow.shadowCameraNear, pointLightShadow.shadowCameraFar ) : 1.0;
		#endif
		RE_Direct( directLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if ( NUM_SPOT_LIGHTS > 0 ) && defined( RE_Direct )
	SpotLight spotLight;
	vec4 spotColor;
	vec3 spotLightCoord;
	bool inSpotLightMap;
	#if defined( USE_SHADOWMAP ) && NUM_SPOT_LIGHT_SHADOWS > 0
	SpotLightShadow spotLightShadow;
	#endif
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_SPOT_LIGHTS; i ++ ) {
		spotLight = spotLights[ i ];
		getSpotLightInfo( spotLight, geometryPosition, directLight );
		#if ( UNROLLED_LOOP_INDEX < NUM_SPOT_LIGHT_SHADOWS_WITH_MAPS )
		#define SPOT_LIGHT_MAP_INDEX UNROLLED_LOOP_INDEX
		#elif ( UNROLLED_LOOP_INDEX < NUM_SPOT_LIGHT_SHADOWS )
		#define SPOT_LIGHT_MAP_INDEX NUM_SPOT_LIGHT_MAPS
		#else
		#define SPOT_LIGHT_MAP_INDEX ( UNROLLED_LOOP_INDEX - NUM_SPOT_LIGHT_SHADOWS + NUM_SPOT_LIGHT_SHADOWS_WITH_MAPS )
		#endif
		#if ( SPOT_LIGHT_MAP_INDEX < NUM_SPOT_LIGHT_MAPS )
			spotLightCoord = vSpotLightCoord[ i ].xyz / vSpotLightCoord[ i ].w;
			inSpotLightMap = all( lessThan( abs( spotLightCoord * 2. - 1. ), vec3( 1.0 ) ) );
			spotColor = texture2D( spotLightMap[ SPOT_LIGHT_MAP_INDEX ], spotLightCoord.xy );
			directLight.color = inSpotLightMap ? directLight.color * spotColor.rgb : directLight.color;
		#endif
		#undef SPOT_LIGHT_MAP_INDEX
		#if defined( USE_SHADOWMAP ) && ( UNROLLED_LOOP_INDEX < NUM_SPOT_LIGHT_SHADOWS )
		spotLightShadow = spotLightShadows[ i ];
		directLight.color *= ( directLight.visible && receiveShadow ) ? getShadow( spotShadowMap[ i ], spotLightShadow.shadowMapSize, spotLightShadow.shadowIntensity, spotLightShadow.shadowBias, spotLightShadow.shadowRadius, vSpotLightCoord[ i ] ) : 1.0;
		#endif
		RE_Direct( directLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if ( NUM_SUN_LIGHTS > 0 ) && defined( RE_Direct )
	SunLight sunLight;
	#if defined( USE_SHADOWMAP ) && NUM_SUN_LIGHT_SHADOWS > 0
	SunLightShadow sunLightShadow;
	#endif
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_SUN_LIGHTS; i ++ ) {
		sunLight = sunLights[ i ];
		getSunLightInfo( sunLight, directLight );
		#if defined( USE_SHADOWMAP ) && ( UNROLLED_LOOP_INDEX < NUM_SUN_LIGHT_SHADOWS )
		sunLightShadow = sunLightShadows[ i ];
		directLight.color *= ( directLight.visible && receiveShadow ) ? getSunShadow( sunShadowMap[ i ], sunLightShadow, UNROLLED_LOOP_INDEX ) : 1.0;
		#endif
		RE_Direct( directLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if ( NUM_DIR_LIGHTS > 0 ) && defined( RE_Direct )
	DirectionalLight directionalLight;
	#if defined( USE_SHADOWMAP ) && NUM_DIR_LIGHT_SHADOWS > 0
	DirectionalLightShadow directionalLightShadow;
	#endif
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_DIR_LIGHTS; i ++ ) {
		directionalLight = directionalLights[ i ];
		getDirectionalLightInfo( directionalLight, directLight );
		#if defined( USE_SHADOWMAP ) && ( UNROLLED_LOOP_INDEX < NUM_DIR_LIGHT_SHADOWS )
		directionalLightShadow = directionalLightShadows[ i ];
		directLight.color *= ( directLight.visible && receiveShadow ) ? getShadow( directionalShadowMap[ i ], directionalLightShadow.shadowMapSize, directionalLightShadow.shadowIntensity, directionalLightShadow.shadowBias, directionalLightShadow.shadowRadius, vDirectionalShadowCoord[ i ] ) : 1.0;
		#endif
		RE_Direct( directLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if ( NUM_RECT_AREA_LIGHTS > 0 ) && defined( RE_Direct_RectArea )
	RectAreaLight rectAreaLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_RECT_AREA_LIGHTS; i ++ ) {
		rectAreaLight = rectAreaLights[ i ];
		RE_Direct_RectArea( rectAreaLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if defined( RE_IndirectDiffuse )
	vec3 iblIrradiance = vec3( 0.0 );
	vec3 irradiance = getAmbientLightIrradiance( ambientLightColor );
	#if defined( USE_LIGHT_PROBES )
		irradiance += getLightProbeIrradiance( lightProbe, geometryNormal );
	#endif
	#if ( NUM_HEMI_LIGHTS > 0 )
		#pragma unroll_loop_start
		for ( int i = 0; i < NUM_HEMI_LIGHTS; i ++ ) {
			irradiance += getHemisphereLightIrradiance( hemisphereLights[ i ], geometryNormal );
		}
		#pragma unroll_loop_end
	#endif
	#ifdef USE_LIGHT_PROBES_GRID
		vec3 probeWorldPos = ( ( vec4( geometryPosition, 1.0 ) - viewMatrix[ 3 ] ) * viewMatrix ).xyz;
		vec3 probeWorldNormal = transformNormalByInverseViewMatrix( geometryNormal, viewMatrix );
		irradiance += getLightProbeGridIrradiance( probeWorldPos, probeWorldNormal );
	#endif
#endif
#if defined( RE_IndirectSpecular )
	vec3 radiance = vec3( 0.0 );
	vec3 clearcoatRadiance = vec3( 0.0 );
#endif`,bT=`#if defined( RE_IndirectDiffuse )
	#ifdef USE_LIGHTMAP
		vec4 lightMapTexel = texture2D( lightMap, vLightMapUv );
		vec3 lightMapIrradiance = lightMapTexel.rgb * lightMapIntensity;
		irradiance += lightMapIrradiance;
	#endif
	#if defined( USE_ENVMAP ) && defined( ENVMAP_TYPE_CUBE_UV )
		#if defined( STANDARD ) || defined( LAMBERT ) || defined( PHONG )
			iblIrradiance += getIBLIrradiance( geometryNormal );
		#endif
	#endif
#endif
#if defined( USE_ENVMAP ) && defined( RE_IndirectSpecular )
	#ifdef USE_ANISOTROPY
		vec3 iblRadiance = getIBLAnisotropyRadiance( geometryViewDir, geometryNormal, material.roughness, material.anisotropyB, material.anisotropy );
	#else
		vec3 iblRadiance = getIBLRadiance( geometryViewDir, geometryNormal, material.roughness );
	#endif
	#ifdef USE_RETROREFLECTION
		#ifdef USE_ANISOTROPY
			vec3 retroIBLRadiance = getIBLAnisotropyRetroRadiance( geometryViewDir, geometryNormal, material.roughness, material.anisotropyB, material.anisotropy );
		#else
			vec3 retroIBLRadiance = getIBLRetroRadiance( geometryViewDir, geometryNormal, material.roughness );
		#endif
		iblRadiance = mix( iblRadiance, retroIBLRadiance, saturate( material.retroreflectivity ) );
	#endif
	radiance += iblRadiance;
	#ifdef USE_CLEARCOAT
		clearcoatRadiance += getIBLRadiance( geometryViewDir, geometryClearcoatNormal, material.clearcoatRoughness );
	#endif
#endif`,TT=`#if defined( RE_IndirectDiffuse )
	#if defined( LAMBERT ) || defined( PHONG )
		irradiance += iblIrradiance;
	#endif
	RE_IndirectDiffuse( irradiance, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
#endif
#if defined( RE_IndirectSpecular )
	RE_IndirectSpecular( radiance, iblIrradiance, clearcoatRadiance, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
#endif`,AT=`#ifdef USE_LIGHT_PROBES_GRID
uniform highp sampler3D probesSH;
uniform vec3 probesMin;
uniform vec3 probesMax;
uniform vec3 probesResolution;
vec3 getLightProbeGridIrradiance( vec3 worldPos, vec3 worldNormal ) {
	vec3 res = probesResolution;
	vec3 gridRange = probesMax - probesMin;
	vec3 resMinusOne = res - 1.0;
	vec3 probeSpacing = gridRange / resMinusOne;
	vec3 samplePos = worldPos + worldNormal * probeSpacing * 0.5;
	vec3 uvw = clamp( ( samplePos - probesMin ) / gridRange, 0.0, 1.0 );
	uvw = uvw * resMinusOne / res + 0.5 / res;
	float nz          = res.z;
	float paddedSlices = nz + 2.0;
	float atlasDepth  = 7.0 * paddedSlices;
	float uvZBase     = uvw.z * nz + 1.0;
	vec4 s0 = texture( probesSH, vec3( uvw.xy, ( uvZBase                       ) / atlasDepth ) );
	vec4 s1 = texture( probesSH, vec3( uvw.xy, ( uvZBase +       paddedSlices   ) / atlasDepth ) );
	vec4 s2 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 2.0 * paddedSlices   ) / atlasDepth ) );
	vec4 s3 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 3.0 * paddedSlices   ) / atlasDepth ) );
	vec4 s4 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 4.0 * paddedSlices   ) / atlasDepth ) );
	vec4 s5 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 5.0 * paddedSlices   ) / atlasDepth ) );
	vec4 s6 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 6.0 * paddedSlices   ) / atlasDepth ) );
	vec3 c0 = s0.xyz;
	vec3 c1 = vec3( s0.w, s1.xy );
	vec3 c2 = vec3( s1.zw, s2.x );
	vec3 c3 = s2.yzw;
	vec3 c4 = s3.xyz;
	vec3 c5 = vec3( s3.w, s4.xy );
	vec3 c6 = vec3( s4.zw, s5.x );
	vec3 c7 = s5.yzw;
	vec3 c8 = s6.xyz;
	float x = worldNormal.x, y = worldNormal.y, z = worldNormal.z;
	vec3 result = c0 * 0.886227;
	result += c1 * 2.0 * 0.511664 * y;
	result += c2 * 2.0 * 0.511664 * z;
	result += c3 * 2.0 * 0.511664 * x;
	result += c4 * 2.0 * 0.429043 * x * y;
	result += c5 * 2.0 * 0.429043 * y * z;
	result += c6 * ( 0.743125 * z * z - 0.247708 );
	result += c7 * 2.0 * 0.429043 * x * z;
	result += c8 * 0.429043 * ( x * x - y * y );
	return max( result, vec3( 0.0 ) );
}
#endif`,RT=`#if defined( USE_LOGARITHMIC_DEPTH_BUFFER )
	gl_FragDepth = vIsPerspective == 0.0 ? gl_FragCoord.z : log2( vFragDepth ) * logDepthBufFC * 0.5;
#endif`,wT=`#if defined( USE_LOGARITHMIC_DEPTH_BUFFER )
	uniform float logDepthBufFC;
	varying float vFragDepth;
	varying float vIsPerspective;
#endif`,CT=`#ifdef USE_LOGARITHMIC_DEPTH_BUFFER
	varying float vFragDepth;
	varying float vIsPerspective;
#endif`,NT=`#ifdef USE_LOGARITHMIC_DEPTH_BUFFER
	vFragDepth = 1.0 + gl_Position.w;
	vIsPerspective = float( isPerspectiveMatrix( projectionMatrix ) );
#endif`,DT=`#ifdef USE_MAP
	vec4 sampledDiffuseColor = texture2D( map, vMapUv );
	#ifdef DECODE_VIDEO_TEXTURE
		sampledDiffuseColor = sRGBTransferEOTF( sampledDiffuseColor );
	#endif
	diffuseColor *= sampledDiffuseColor;
#endif`,UT=`#ifdef USE_MAP
	uniform sampler2D map;
#endif`,LT=`#if defined( USE_MAP ) || defined( USE_ALPHAMAP )
	#if defined( USE_POINTS_UV )
		vec2 uv = vUv;
	#else
		vec2 uv = ( uvTransform * vec3( gl_PointCoord.x, 1.0 - gl_PointCoord.y, 1 ) ).xy;
	#endif
#endif
#ifdef USE_MAP
	diffuseColor *= texture2D( map, uv );
#endif
#ifdef USE_ALPHAMAP
	diffuseColor.a *= texture2D( alphaMap, uv ).g;
#endif`,OT=`#if defined( USE_POINTS_UV )
	varying vec2 vUv;
#else
	#if defined( USE_MAP ) || defined( USE_ALPHAMAP )
		uniform mat3 uvTransform;
	#endif
#endif
#ifdef USE_MAP
	uniform sampler2D map;
#endif
#ifdef USE_ALPHAMAP
	uniform sampler2D alphaMap;
#endif`,PT=`float metalnessFactor = metalness;
#ifdef USE_METALNESSMAP
	vec4 texelMetalness = texture2D( metalnessMap, vMetalnessMapUv );
	metalnessFactor *= texelMetalness.b;
#endif`,IT=`#ifdef USE_METALNESSMAP
	uniform sampler2D metalnessMap;
#endif`,BT=`#ifdef USE_INSTANCING_MORPH
	float morphTargetInfluences[ MORPHTARGETS_COUNT ];
	float morphTargetBaseInfluence = texelFetch( morphTexture, ivec2( 0, gl_InstanceID ), 0 ).r;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		morphTargetInfluences[i] =  texelFetch( morphTexture, ivec2( i + 1, gl_InstanceID ), 0 ).r;
	}
#endif`,zT=`#if defined( USE_MORPHCOLORS )
	vColor *= morphTargetBaseInfluence;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		#if defined( USE_COLOR_ALPHA )
			if ( morphTargetInfluences[ i ] != 0.0 ) vColor += getMorph( gl_VertexID, i, 2 ) * morphTargetInfluences[ i ];
		#elif defined( USE_COLOR )
			if ( morphTargetInfluences[ i ] != 0.0 ) vColor += getMorph( gl_VertexID, i, 2 ).rgb * morphTargetInfluences[ i ];
		#endif
	}
#endif`,FT=`#ifdef USE_MORPHNORMALS
	objectNormal *= morphTargetBaseInfluence;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		if ( morphTargetInfluences[ i ] != 0.0 ) objectNormal += getMorph( gl_VertexID, i, 1 ).xyz * morphTargetInfluences[ i ];
	}
#endif`,HT=`#ifdef USE_MORPHTARGETS
	#ifndef USE_INSTANCING_MORPH
		uniform float morphTargetBaseInfluence;
		uniform float morphTargetInfluences[ MORPHTARGETS_COUNT ];
	#endif
	uniform sampler2DArray morphTargetsTexture;
	uniform ivec2 morphTargetsTextureSize;
	vec4 getMorph( const in int vertexIndex, const in int morphTargetIndex, const in int offset ) {
		int texelIndex = vertexIndex * MORPHTARGETS_TEXTURE_STRIDE + offset;
		int y = texelIndex / morphTargetsTextureSize.x;
		int x = texelIndex - y * morphTargetsTextureSize.x;
		ivec3 morphUV = ivec3( x, y, morphTargetIndex );
		return texelFetch( morphTargetsTexture, morphUV, 0 );
	}
#endif`,GT=`#ifdef USE_MORPHTARGETS
	transformed *= morphTargetBaseInfluence;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		if ( morphTargetInfluences[ i ] != 0.0 ) transformed += getMorph( gl_VertexID, i, 0 ).xyz * morphTargetInfluences[ i ];
	}
#endif`,VT=`float faceDirection = gl_FrontFacing ? 1.0 : - 1.0;
#ifdef FLAT_SHADED
	vec3 fdx = dFdx( vViewPosition );
	vec3 fdy = dFdy( vViewPosition );
	vec3 normal = normalize( cross( fdx, fdy ) );
#else
	vec3 normal = normalize( vNormal );
	#ifdef DOUBLE_SIDED
		normal *= faceDirection;
	#endif
#endif
#if defined( USE_NORMALMAP_TANGENTSPACE ) || defined( USE_CLEARCOAT_NORMALMAP ) || defined( USE_ANISOTROPY )
	#ifdef USE_TANGENT
		mat3 tbn = mat3( normalize( vTangent ), normalize( vBitangent ), normal );
	#else
		mat3 tbn = getTangentFrame( - vViewPosition, normal,
		#if defined( USE_NORMALMAP )
			vNormalMapUv
		#elif defined( USE_CLEARCOAT_NORMALMAP )
			vClearcoatNormalMapUv
		#else
			vUv
		#endif
		);
	#endif
	#ifdef DOUBLE_SIDED
		tbn[0] *= faceDirection;
		tbn[1] *= faceDirection;
	#endif
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	#ifdef USE_TANGENT
		mat3 tbn2 = mat3( normalize( vTangent ), normalize( vBitangent ), normal );
	#else
		mat3 tbn2 = getTangentFrame( - vViewPosition, normal, vClearcoatNormalMapUv );
	#endif
	#ifdef DOUBLE_SIDED
		tbn2[0] *= faceDirection;
		tbn2[1] *= faceDirection;
	#endif
#endif
vec3 nonPerturbedNormal = normal;`,kT=`#ifdef USE_NORMALMAP_OBJECTSPACE
	normal = texture2D( normalMap, vNormalMapUv ).xyz * 2.0 - 1.0;
	#ifdef FLIP_SIDED
		normal = - normal;
	#endif
	#ifdef DOUBLE_SIDED
		normal = normal * faceDirection;
	#endif
	normal = normalize( normalMatrix * normal );
#elif defined( USE_NORMALMAP_TANGENTSPACE )
	vec3 mapN = texture2D( normalMap, vNormalMapUv ).xyz * 2.0 - 1.0;
	#if defined( USE_PACKED_NORMALMAP )
		mapN = vec3( mapN.xy, sqrt( saturate( 1.0 - dot( mapN.xy, mapN.xy ) ) ) );
	#endif
	mapN.xy *= normalScale;
	normal = normalize( tbn * mapN );
#elif defined( USE_BUMPMAP )
	normal = perturbNormalArb( - vViewPosition, normal, dHdxy_fwd(), faceDirection );
#endif`,XT=`#ifndef FLAT_SHADED
	varying vec3 vNormal;
	#ifdef USE_TANGENT
		varying vec3 vTangent;
		varying vec3 vBitangent;
	#endif
#endif`,WT=`#ifndef FLAT_SHADED
	varying vec3 vNormal;
	#ifdef USE_TANGENT
		varying vec3 vTangent;
		varying vec3 vBitangent;
	#endif
#endif`,qT=`#ifndef FLAT_SHADED
	vNormal = normalize( transformedNormal );
	#ifdef USE_TANGENT
		vTangent = normalize( transformedTangent );
		vBitangent = normalize( cross( vNormal, vTangent ) * tangent.w );
		#ifdef FLIP_SIDED
			vBitangent = - vBitangent;
		#endif
	#endif
#endif`,YT=`#ifdef USE_NORMALMAP
	uniform sampler2D normalMap;
	uniform vec2 normalScale;
#endif
#ifdef USE_NORMALMAP_OBJECTSPACE
	uniform mat3 normalMatrix;
#endif
#if ! defined ( USE_TANGENT ) && ( defined ( USE_NORMALMAP_TANGENTSPACE ) || defined ( USE_CLEARCOAT_NORMALMAP ) || defined( USE_ANISOTROPY ) )
	mat3 getTangentFrame( vec3 eye_pos, vec3 surf_norm, vec2 uv ) {
		vec3 q0 = dFdx( eye_pos.xyz );
		vec3 q1 = dFdy( eye_pos.xyz );
		vec2 st0 = dFdx( uv.st );
		vec2 st1 = dFdy( uv.st );
		vec3 N = surf_norm;
		vec3 q1perp = cross( q1, N );
		vec3 q0perp = cross( N, q0 );
		vec3 T = q1perp * st0.x + q0perp * st1.x;
		vec3 B = q1perp * st0.y + q0perp * st1.y;
		float det = max( dot( T, T ), dot( B, B ) );
		float scale = ( det == 0.0 ) ? 0.0 : inversesqrt( det );
		return mat3( T * scale, B * scale, N );
	}
#endif`,jT=`#ifdef USE_CLEARCOAT
	vec3 clearcoatNormal = nonPerturbedNormal;
#endif`,ZT=`#ifdef USE_CLEARCOAT_NORMALMAP
	vec3 clearcoatMapN = texture2D( clearcoatNormalMap, vClearcoatNormalMapUv ).xyz * 2.0 - 1.0;
	clearcoatMapN.xy *= clearcoatNormalScale;
	clearcoatNormal = normalize( tbn2 * clearcoatMapN );
#endif`,KT=`#ifdef USE_CLEARCOATMAP
	uniform sampler2D clearcoatMap;
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	uniform sampler2D clearcoatNormalMap;
	uniform vec2 clearcoatNormalScale;
#endif
#ifdef USE_CLEARCOAT_ROUGHNESSMAP
	uniform sampler2D clearcoatRoughnessMap;
#endif`,QT=`#ifdef USE_IRIDESCENCEMAP
	uniform sampler2D iridescenceMap;
#endif
#ifdef USE_IRIDESCENCE_THICKNESSMAP
	uniform sampler2D iridescenceThicknessMap;
#endif`,JT=`#ifdef OPAQUE
diffuseColor.a = 1.0;
#endif
#ifdef USE_TRANSMISSION
diffuseColor.a *= material.transmissionAlpha;
#endif
gl_FragColor = vec4( outgoingLight, diffuseColor.a );`,$T=`vec3 packNormalToRGB( const in vec3 normal ) {
	return normalize( normal ) * 0.5 + 0.5;
}
vec3 unpackRGBToNormal( const in vec3 rgb ) {
	return 2.0 * rgb.xyz - 1.0;
}
const float PackUpscale = 256. / 255.;const float UnpackDownscale = 255. / 256.;const float ShiftRight8 = 1. / 256.;
const float Inv255 = 1. / 255.;
const vec4 PackFactors = vec4( 1.0, 256.0, 256.0 * 256.0, 256.0 * 256.0 * 256.0 );
const vec2 UnpackFactors2 = vec2( UnpackDownscale, 1.0 / PackFactors.g );
const vec3 UnpackFactors3 = vec3( UnpackDownscale / PackFactors.rg, 1.0 / PackFactors.b );
const vec4 UnpackFactors4 = vec4( UnpackDownscale / PackFactors.rgb, 1.0 / PackFactors.a );
vec4 packDepthToRGBA( const in float v ) {
	if( v <= 0.0 )
		return vec4( 0., 0., 0., 0. );
	if( v >= 1.0 )
		return vec4( 1., 1., 1., 1. );
	float vuf;
	float af = modf( v * PackFactors.a, vuf );
	float bf = modf( vuf * ShiftRight8, vuf );
	float gf = modf( vuf * ShiftRight8, vuf );
	return vec4( vuf * Inv255, gf * PackUpscale, bf * PackUpscale, af );
}
vec3 packDepthToRGB( const in float v ) {
	if( v <= 0.0 )
		return vec3( 0., 0., 0. );
	if( v >= 1.0 )
		return vec3( 1., 1., 1. );
	float vuf;
	float bf = modf( v * PackFactors.b, vuf );
	float gf = modf( vuf * ShiftRight8, vuf );
	return vec3( vuf * Inv255, gf * PackUpscale, bf );
}
vec2 packDepthToRG( const in float v ) {
	if( v <= 0.0 )
		return vec2( 0., 0. );
	if( v >= 1.0 )
		return vec2( 1., 1. );
	float vuf;
	float gf = modf( v * 256., vuf );
	return vec2( vuf * Inv255, gf );
}
float unpackRGBAToDepth( const in vec4 v ) {
	return dot( v, UnpackFactors4 );
}
float unpackRGBToDepth( const in vec3 v ) {
	return dot( v, UnpackFactors3 );
}
float unpackRGToDepth( const in vec2 v ) {
	return v.r * UnpackFactors2.r + v.g * UnpackFactors2.g;
}
vec4 pack2HalfToRGBA( const in vec2 v ) {
	vec4 r = vec4( v.x, fract( v.x * 255.0 ), v.y, fract( v.y * 255.0 ) );
	return vec4( r.x - r.y / 255.0, r.y, r.z - r.w / 255.0, r.w );
}
vec2 unpackRGBATo2Half( const in vec4 v ) {
	return vec2( v.x + ( v.y / 255.0 ), v.z + ( v.w / 255.0 ) );
}
float viewZToOrthographicDepth( const in float viewZ, const in float near, const in float far ) {
	return ( viewZ + near ) / ( near - far );
}
float orthographicDepthToViewZ( const in float depth, const in float near, const in float far ) {
	#ifdef USE_REVERSED_DEPTH_BUFFER
	
		return depth * ( far - near ) - far;
	#else
		return depth * ( near - far ) - near;
	#endif
}
float viewZToPerspectiveDepth( const in float viewZ, const in float near, const in float far ) {
	return ( ( near + viewZ ) * far ) / ( ( far - near ) * viewZ );
}
float perspectiveDepthToViewZ( const in float depth, const in float near, const in float far ) {
	
	#ifdef USE_REVERSED_DEPTH_BUFFER
		return ( near * far ) / ( ( near - far ) * depth - near );
	#else
		return ( near * far ) / ( ( far - near ) * depth - far );
	#endif
}`,eA=`#ifdef PREMULTIPLIED_ALPHA
	gl_FragColor.rgb *= gl_FragColor.a;
#endif`,tA=`vec4 mvPosition = vec4( transformed, 1.0 );
#ifdef USE_BATCHING
	mvPosition = batchingMatrix * mvPosition;
#endif
#ifdef USE_INSTANCING
	mvPosition = instanceMatrix * mvPosition;
#endif
mvPosition = modelViewMatrix * mvPosition;
gl_Position = projectionMatrix * mvPosition;`,nA=`#ifdef DITHERING
	gl_FragColor.rgb = dithering( gl_FragColor.rgb );
#endif`,iA=`#ifdef DITHERING
	vec3 dithering( vec3 color ) {
		float grid_position = rand( gl_FragCoord.xy );
		vec3 dither_shift_RGB = vec3( 0.25 / 255.0, -0.25 / 255.0, 0.25 / 255.0 );
		dither_shift_RGB = mix( 2.0 * dither_shift_RGB, -2.0 * dither_shift_RGB, grid_position );
		return color + dither_shift_RGB;
	}
#endif`,aA=`float roughnessFactor = roughness;
#ifdef USE_ROUGHNESSMAP
	vec4 texelRoughness = texture2D( roughnessMap, vRoughnessMapUv );
	roughnessFactor *= texelRoughness.g;
#endif`,sA=`#ifdef USE_ROUGHNESSMAP
	uniform sampler2D roughnessMap;
#endif`,rA=`#if NUM_SPOT_LIGHT_COORDS > 0
	varying vec4 vSpotLightCoord[ NUM_SPOT_LIGHT_COORDS ];
#endif
#if NUM_SPOT_LIGHT_MAPS > 0
	uniform sampler2D spotLightMap[ NUM_SPOT_LIGHT_MAPS ];
#endif
#ifdef USE_SHADOWMAP
	#if NUM_SUN_LIGHT_SHADOWS > 0
		#define SUN_LIGHT_CASCADES 2
		#if defined( SHADOWMAP_TYPE_PCF )
			uniform sampler2DShadow sunShadowMap[ NUM_SUN_LIGHT_SHADOWS ];
		#else
			uniform sampler2D sunShadowMap[ NUM_SUN_LIGHT_SHADOWS ];
		#endif
		uniform mat4 sunShadowMatrix[ NUM_SUN_LIGHT_SHADOWS * SUN_LIGHT_CASCADES ];
		uniform vec4 sunShadowCascade[ NUM_SUN_LIGHT_SHADOWS * SUN_LIGHT_CASCADES ];
		varying vec4 vSunShadowWorldPosition;
		varying vec3 vSunShadowWorldNormal;
		struct SunLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform SunLightShadow sunLightShadows[ NUM_SUN_LIGHT_SHADOWS ];
	#endif
	#if NUM_DIR_LIGHT_SHADOWS > 0
		#if defined( SHADOWMAP_TYPE_PCF )
			uniform sampler2DShadow directionalShadowMap[ NUM_DIR_LIGHT_SHADOWS ];
		#else
			uniform sampler2D directionalShadowMap[ NUM_DIR_LIGHT_SHADOWS ];
		#endif
		varying vec4 vDirectionalShadowCoord[ NUM_DIR_LIGHT_SHADOWS ];
		struct DirectionalLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform DirectionalLightShadow directionalLightShadows[ NUM_DIR_LIGHT_SHADOWS ];
	#endif
	#if NUM_SPOT_LIGHT_SHADOWS > 0
		#if defined( SHADOWMAP_TYPE_PCF )
			uniform sampler2DShadow spotShadowMap[ NUM_SPOT_LIGHT_SHADOWS ];
		#else
			uniform sampler2D spotShadowMap[ NUM_SPOT_LIGHT_SHADOWS ];
		#endif
		struct SpotLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform SpotLightShadow spotLightShadows[ NUM_SPOT_LIGHT_SHADOWS ];
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0
		#if defined( SHADOWMAP_TYPE_PCF )
			uniform samplerCubeShadow pointShadowMap[ NUM_POINT_LIGHT_SHADOWS ];
		#elif defined( SHADOWMAP_TYPE_BASIC )
			uniform samplerCube pointShadowMap[ NUM_POINT_LIGHT_SHADOWS ];
		#endif
		varying vec4 vPointShadowCoord[ NUM_POINT_LIGHT_SHADOWS ];
		struct PointLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
			float shadowCameraNear;
			float shadowCameraFar;
		};
		uniform PointLightShadow pointLightShadows[ NUM_POINT_LIGHT_SHADOWS ];
	#endif
	#if defined( SHADOWMAP_TYPE_PCF )
		float interleavedGradientNoise( vec2 position ) {
			return fract( 52.9829189 * fract( dot( position, vec2( 0.06711056, 0.00583715 ) ) ) );
		}
		vec2 vogelDiskSample( int sampleIndex, int samplesCount, float phi ) {
			const float goldenAngle = 2.399963229728653;
			float r = sqrt( ( float( sampleIndex ) + 0.5 ) / float( samplesCount ) );
			float theta = float( sampleIndex ) * goldenAngle + phi;
			return vec2( cos( theta ), sin( theta ) ) * r;
		}
	#endif
	#if defined( SHADOWMAP_TYPE_PCF )
		float getShadow( sampler2DShadow shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord ) {
			float shadow = 1.0;
			shadowCoord.xyz /= shadowCoord.w;
			shadowCoord.z += shadowBias;
			bool inFrustum = shadowCoord.x >= 0.0 && shadowCoord.x <= 1.0 && shadowCoord.y >= 0.0 && shadowCoord.y <= 1.0;
			bool frustumTest = inFrustum && shadowCoord.z <= 1.0;
			if ( frustumTest ) {
				vec2 texelSize = vec2( 1.0 ) / shadowMapSize;
				float radius = shadowRadius * texelSize.x;
				float phi = interleavedGradientNoise( gl_FragCoord.xy ) * PI2;
				shadow = (
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 0, 5, phi ) * radius, shadowCoord.z ) ) +
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 1, 5, phi ) * radius, shadowCoord.z ) ) +
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 2, 5, phi ) * radius, shadowCoord.z ) ) +
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 3, 5, phi ) * radius, shadowCoord.z ) ) +
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 4, 5, phi ) * radius, shadowCoord.z ) )
				) * 0.2;
			}
			return mix( 1.0, shadow, shadowIntensity );
		}
	#elif defined( SHADOWMAP_TYPE_VSM )
		float getShadow( sampler2D shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord ) {
			float shadow = 1.0;
			shadowCoord.xyz /= shadowCoord.w;
			#ifdef USE_REVERSED_DEPTH_BUFFER
				shadowCoord.z -= shadowBias;
			#else
				shadowCoord.z += shadowBias;
			#endif
			bool inFrustum = shadowCoord.x >= 0.0 && shadowCoord.x <= 1.0 && shadowCoord.y >= 0.0 && shadowCoord.y <= 1.0;
			bool frustumTest = inFrustum && shadowCoord.z <= 1.0;
			if ( frustumTest ) {
				vec2 distribution = texture2D( shadowMap, shadowCoord.xy ).rg;
				float mean = distribution.x;
				float variance = distribution.y * distribution.y;
				#ifdef USE_REVERSED_DEPTH_BUFFER
					float hard_shadow = step( mean, shadowCoord.z );
				#else
					float hard_shadow = step( shadowCoord.z, mean );
				#endif
				
				if ( hard_shadow == 1.0 ) {
					shadow = 1.0;
				} else {
					variance = max( variance, 0.0000001 );
					float d = shadowCoord.z - mean;
					float p_max = variance / ( variance + d * d );
					p_max = clamp( ( p_max - 0.3 ) / 0.65, 0.0, 1.0 );
					shadow = max( hard_shadow, p_max );
				}
			}
			return mix( 1.0, shadow, shadowIntensity );
		}
	#else
		float getShadow( sampler2D shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord ) {
			float shadow = 1.0;
			shadowCoord.xyz /= shadowCoord.w;
			#ifdef USE_REVERSED_DEPTH_BUFFER
				shadowCoord.z -= shadowBias;
			#else
				shadowCoord.z += shadowBias;
			#endif
			bool inFrustum = shadowCoord.x >= 0.0 && shadowCoord.x <= 1.0 && shadowCoord.y >= 0.0 && shadowCoord.y <= 1.0;
			bool frustumTest = inFrustum && shadowCoord.z <= 1.0;
			if ( frustumTest ) {
				float depth = texture2D( shadowMap, shadowCoord.xy ).r;
				#ifdef USE_REVERSED_DEPTH_BUFFER
					shadow = step( depth, shadowCoord.z );
				#else
					shadow = step( shadowCoord.z, depth );
				#endif
			}
			return mix( 1.0, shadow, shadowIntensity );
		}
	#endif
	#if NUM_SUN_LIGHT_SHADOWS > 0
		float getSunShadow(
			#if defined( SHADOWMAP_TYPE_PCF )
				sampler2DShadow shadowMap,
			#else
				sampler2D shadowMap,
			#endif
			SunLightShadow sunLightShadow,
			int shadowIndex
		) {
			vec4 shadowWorldPosition = vec4( vSunShadowWorldPosition.xyz + vSunShadowWorldNormal * sunLightShadow.shadowNormalBias, 1.0 );
			float viewDepth = vSunShadowWorldPosition.w;
			int cascadeOffset = shadowIndex * SUN_LIGHT_CASCADES;
			float shadow = 1.0;
			for ( int i = SUN_LIGHT_CASCADES - 1; i >= 0; i -- ) {
				vec4 cascade = sunShadowCascade[ cascadeOffset + i ];
				if ( viewDepth >= cascade.x && viewDepth < cascade.y ) {
					float cascadeShadow = getShadow(
						shadowMap,
						sunLightShadow.shadowMapSize,
						sunLightShadow.shadowIntensity,
						sunLightShadow.shadowBias,
						sunLightShadow.shadowRadius,
						sunShadowMatrix[ cascadeOffset + i ] * shadowWorldPosition
					);
					shadow = mix( cascadeShadow, shadow, smoothstep( cascade.z, cascade.y, viewDepth ) );
				}
			}
			return shadow;
		}
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0
	#if defined( SHADOWMAP_TYPE_PCF )
	float getPointShadow( samplerCubeShadow shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord, float shadowCameraNear, float shadowCameraFar ) {
		float shadow = 1.0;
		vec3 lightToPosition = shadowCoord.xyz;
		vec3 bd3D = normalize( lightToPosition );
		vec3 absVec = abs( lightToPosition );
		float viewSpaceZ = max( max( absVec.x, absVec.y ), absVec.z );
		if ( viewSpaceZ - shadowCameraFar <= 0.0 && viewSpaceZ - shadowCameraNear >= 0.0 ) {
			#ifdef USE_REVERSED_DEPTH_BUFFER
				float dp = ( shadowCameraNear * ( shadowCameraFar - viewSpaceZ ) ) / ( viewSpaceZ * ( shadowCameraFar - shadowCameraNear ) );
				dp -= shadowBias;
			#else
				float dp = ( shadowCameraFar * ( viewSpaceZ - shadowCameraNear ) ) / ( viewSpaceZ * ( shadowCameraFar - shadowCameraNear ) );
				dp += shadowBias;
			#endif
			float texelSize = shadowRadius / shadowMapSize.x;
			vec3 absDir = abs( bd3D );
			vec3 tangent = absDir.x > absDir.z ? vec3( 0.0, 1.0, 0.0 ) : vec3( 1.0, 0.0, 0.0 );
			tangent = normalize( cross( bd3D, tangent ) );
			vec3 bitangent = cross( bd3D, tangent );
			float phi = interleavedGradientNoise( gl_FragCoord.xy ) * PI2;
			vec2 sample0 = vogelDiskSample( 0, 5, phi );
			vec2 sample1 = vogelDiskSample( 1, 5, phi );
			vec2 sample2 = vogelDiskSample( 2, 5, phi );
			vec2 sample3 = vogelDiskSample( 3, 5, phi );
			vec2 sample4 = vogelDiskSample( 4, 5, phi );
			shadow = (
				texture( shadowMap, vec4( bd3D + ( tangent * sample0.x + bitangent * sample0.y ) * texelSize, dp ) ) +
				texture( shadowMap, vec4( bd3D + ( tangent * sample1.x + bitangent * sample1.y ) * texelSize, dp ) ) +
				texture( shadowMap, vec4( bd3D + ( tangent * sample2.x + bitangent * sample2.y ) * texelSize, dp ) ) +
				texture( shadowMap, vec4( bd3D + ( tangent * sample3.x + bitangent * sample3.y ) * texelSize, dp ) ) +
				texture( shadowMap, vec4( bd3D + ( tangent * sample4.x + bitangent * sample4.y ) * texelSize, dp ) )
			) * 0.2;
		}
		return mix( 1.0, shadow, shadowIntensity );
	}
	#elif defined( SHADOWMAP_TYPE_BASIC )
	float getPointShadow( samplerCube shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord, float shadowCameraNear, float shadowCameraFar ) {
		float shadow = 1.0;
		vec3 lightToPosition = shadowCoord.xyz;
		vec3 absVec = abs( lightToPosition );
		float viewSpaceZ = max( max( absVec.x, absVec.y ), absVec.z );
		if ( viewSpaceZ - shadowCameraFar <= 0.0 && viewSpaceZ - shadowCameraNear >= 0.0 ) {
			float dp = ( shadowCameraFar * ( viewSpaceZ - shadowCameraNear ) ) / ( viewSpaceZ * ( shadowCameraFar - shadowCameraNear ) );
			dp += shadowBias;
			vec3 bd3D = normalize( lightToPosition );
			float depth = textureCube( shadowMap, bd3D ).r;
			#ifdef USE_REVERSED_DEPTH_BUFFER
				depth = 1.0 - depth;
			#endif
			shadow = step( dp, depth );
		}
		return mix( 1.0, shadow, shadowIntensity );
	}
	#endif
	#endif
#endif`,oA=`#if NUM_SPOT_LIGHT_COORDS > 0
	uniform mat4 spotLightMatrix[ NUM_SPOT_LIGHT_COORDS ];
	varying vec4 vSpotLightCoord[ NUM_SPOT_LIGHT_COORDS ];
#endif
#ifdef USE_SHADOWMAP
	#if NUM_SUN_LIGHT_SHADOWS > 0
		varying vec4 vSunShadowWorldPosition;
		varying vec3 vSunShadowWorldNormal;
	#endif
	#if NUM_DIR_LIGHT_SHADOWS > 0
		uniform mat4 directionalShadowMatrix[ NUM_DIR_LIGHT_SHADOWS ];
		varying vec4 vDirectionalShadowCoord[ NUM_DIR_LIGHT_SHADOWS ];
		struct DirectionalLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform DirectionalLightShadow directionalLightShadows[ NUM_DIR_LIGHT_SHADOWS ];
	#endif
	#if NUM_SPOT_LIGHT_SHADOWS > 0
		struct SpotLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform SpotLightShadow spotLightShadows[ NUM_SPOT_LIGHT_SHADOWS ];
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0
		uniform mat4 pointShadowMatrix[ NUM_POINT_LIGHT_SHADOWS ];
		varying vec4 vPointShadowCoord[ NUM_POINT_LIGHT_SHADOWS ];
		struct PointLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
			float shadowCameraNear;
			float shadowCameraFar;
		};
		uniform PointLightShadow pointLightShadows[ NUM_POINT_LIGHT_SHADOWS ];
	#endif
#endif`,lA=`#if ( defined( USE_SHADOWMAP ) && ( NUM_DIR_LIGHT_SHADOWS > 0 || NUM_SUN_LIGHT_SHADOWS > 0 || NUM_POINT_LIGHT_SHADOWS > 0 ) ) || ( NUM_SPOT_LIGHT_COORDS > 0 )
	#ifdef HAS_NORMAL
		vec3 shadowWorldNormal = transformNormalByInverseViewMatrix( transformedNormal, viewMatrix );
	#else
		vec3 shadowWorldNormal = vec3( 0.0 );
	#endif
	vec4 shadowWorldPosition;
#endif
#if defined( USE_SHADOWMAP )
	#if NUM_SUN_LIGHT_SHADOWS > 0
		vSunShadowWorldPosition = vec4( worldPosition.xyz, - mvPosition.z );
		vSunShadowWorldNormal = shadowWorldNormal;
	#endif
	#if NUM_DIR_LIGHT_SHADOWS > 0
		#pragma unroll_loop_start
		for ( int i = 0; i < NUM_DIR_LIGHT_SHADOWS; i ++ ) {
			shadowWorldPosition = worldPosition + vec4( shadowWorldNormal * directionalLightShadows[ i ].shadowNormalBias, 0 );
			vDirectionalShadowCoord[ i ] = directionalShadowMatrix[ i ] * shadowWorldPosition;
		}
		#pragma unroll_loop_end
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0
		#pragma unroll_loop_start
		for ( int i = 0; i < NUM_POINT_LIGHT_SHADOWS; i ++ ) {
			shadowWorldPosition = worldPosition + vec4( shadowWorldNormal * pointLightShadows[ i ].shadowNormalBias, 0 );
			vPointShadowCoord[ i ] = pointShadowMatrix[ i ] * shadowWorldPosition;
		}
		#pragma unroll_loop_end
	#endif
#endif
#if NUM_SPOT_LIGHT_COORDS > 0
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_SPOT_LIGHT_COORDS; i ++ ) {
		shadowWorldPosition = worldPosition;
		#if ( defined( USE_SHADOWMAP ) && UNROLLED_LOOP_INDEX < NUM_SPOT_LIGHT_SHADOWS )
			shadowWorldPosition.xyz += shadowWorldNormal * spotLightShadows[ i ].shadowNormalBias;
		#endif
		vSpotLightCoord[ i ] = spotLightMatrix[ i ] * shadowWorldPosition;
	}
	#pragma unroll_loop_end
#endif`,cA=`float getShadowMask() {
	float shadow = 1.0;
	#ifdef USE_SHADOWMAP
	#if NUM_SUN_LIGHT_SHADOWS > 0
	SunLightShadow sunLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_SUN_LIGHT_SHADOWS; i ++ ) {
		sunLight = sunLightShadows[ i ];
		shadow *= receiveShadow ? getSunShadow( sunShadowMap[ i ], sunLight, UNROLLED_LOOP_INDEX ) : 1.0;
	}
	#pragma unroll_loop_end
	#endif
	#if NUM_DIR_LIGHT_SHADOWS > 0
	DirectionalLightShadow directionalLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_DIR_LIGHT_SHADOWS; i ++ ) {
		directionalLight = directionalLightShadows[ i ];
		shadow *= receiveShadow ? getShadow( directionalShadowMap[ i ], directionalLight.shadowMapSize, directionalLight.shadowIntensity, directionalLight.shadowBias, directionalLight.shadowRadius, vDirectionalShadowCoord[ i ] ) : 1.0;
	}
	#pragma unroll_loop_end
	#endif
	#if NUM_SPOT_LIGHT_SHADOWS > 0
	SpotLightShadow spotLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_SPOT_LIGHT_SHADOWS; i ++ ) {
		spotLight = spotLightShadows[ i ];
		shadow *= receiveShadow ? getShadow( spotShadowMap[ i ], spotLight.shadowMapSize, spotLight.shadowIntensity, spotLight.shadowBias, spotLight.shadowRadius, vSpotLightCoord[ i ] ) : 1.0;
	}
	#pragma unroll_loop_end
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0 && ( defined( SHADOWMAP_TYPE_PCF ) || defined( SHADOWMAP_TYPE_BASIC ) )
	PointLightShadow pointLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_POINT_LIGHT_SHADOWS; i ++ ) {
		pointLight = pointLightShadows[ i ];
		shadow *= receiveShadow ? getPointShadow( pointShadowMap[ i ], pointLight.shadowMapSize, pointLight.shadowIntensity, pointLight.shadowBias, pointLight.shadowRadius, vPointShadowCoord[ i ], pointLight.shadowCameraNear, pointLight.shadowCameraFar ) : 1.0;
	}
	#pragma unroll_loop_end
	#endif
	#endif
	return shadow;
}`,uA=`#ifdef USE_SKINNING
	mat4 boneMatX = getBoneMatrix( skinIndex.x );
	mat4 boneMatY = getBoneMatrix( skinIndex.y );
	mat4 boneMatZ = getBoneMatrix( skinIndex.z );
	mat4 boneMatW = getBoneMatrix( skinIndex.w );
#endif`,fA=`#ifdef USE_SKINNING
	uniform mat4 bindMatrix;
	uniform mat4 bindMatrixInverse;
	uniform highp sampler2D boneTexture;
	mat4 getBoneMatrix( const in float i ) {
		int size = textureSize( boneTexture, 0 ).x;
		int j = int( i ) * 4;
		int x = j % size;
		int y = j / size;
		vec4 v1 = texelFetch( boneTexture, ivec2( x, y ), 0 );
		vec4 v2 = texelFetch( boneTexture, ivec2( x + 1, y ), 0 );
		vec4 v3 = texelFetch( boneTexture, ivec2( x + 2, y ), 0 );
		vec4 v4 = texelFetch( boneTexture, ivec2( x + 3, y ), 0 );
		return mat4( v1, v2, v3, v4 );
	}
#endif`,dA=`#ifdef USE_SKINNING
	vec4 skinVertex = bindMatrix * vec4( transformed, 1.0 );
	vec4 skinned = vec4( 0.0 );
	skinned += boneMatX * skinVertex * skinWeight.x;
	skinned += boneMatY * skinVertex * skinWeight.y;
	skinned += boneMatZ * skinVertex * skinWeight.z;
	skinned += boneMatW * skinVertex * skinWeight.w;
	transformed = ( bindMatrixInverse * skinned ).xyz;
#endif`,hA=`#ifdef USE_SKINNING
	mat4 skinMatrix = mat4( 0.0 );
	skinMatrix += skinWeight.x * boneMatX;
	skinMatrix += skinWeight.y * boneMatY;
	skinMatrix += skinWeight.z * boneMatZ;
	skinMatrix += skinWeight.w * boneMatW;
	skinMatrix = bindMatrixInverse * skinMatrix * bindMatrix;
	objectNormal = vec4( skinMatrix * vec4( objectNormal, 0.0 ) ).xyz;
	#ifdef USE_TANGENT
		objectTangent = vec4( skinMatrix * vec4( objectTangent, 0.0 ) ).xyz;
	#endif
#endif`,pA=`float specularStrength;
#ifdef USE_SPECULARMAP
	vec4 texelSpecular = texture2D( specularMap, vSpecularMapUv );
	specularStrength = texelSpecular.r;
#else
	specularStrength = 1.0;
#endif`,mA=`#ifdef USE_SPECULARMAP
	uniform sampler2D specularMap;
#endif`,gA=`#if defined( TONE_MAPPING )
	gl_FragColor.rgb = toneMapping( gl_FragColor.rgb );
#endif`,_A=`#ifndef saturate
#define saturate( a ) clamp( a, 0.0, 1.0 )
#endif
uniform float toneMappingExposure;
vec3 LinearToneMapping( vec3 color ) {
	return saturate( toneMappingExposure * color );
}
vec3 ReinhardToneMapping( vec3 color ) {
	color *= toneMappingExposure;
	return saturate( color / ( vec3( 1.0 ) + color ) );
}
vec3 CineonToneMapping( vec3 color ) {
	color *= toneMappingExposure;
	color = max( vec3( 0.0 ), color - 0.004 );
	return pow( ( color * ( 6.2 * color + 0.5 ) ) / ( color * ( 6.2 * color + 1.7 ) + 0.06 ), vec3( 2.2 ) );
}
vec3 RRTAndODTFit( vec3 v ) {
	vec3 a = v * ( v + 0.0245786 ) - 0.000090537;
	vec3 b = v * ( 0.983729 * v + 0.4329510 ) + 0.238081;
	return a / b;
}
vec3 ACESFilmicToneMapping( vec3 color ) {
	const mat3 ACESInputMat = mat3(
		vec3( 0.59719, 0.07600, 0.02840 ),		vec3( 0.35458, 0.90834, 0.13383 ),
		vec3( 0.04823, 0.01566, 0.83777 )
	);
	const mat3 ACESOutputMat = mat3(
		vec3(  1.60475, -0.10208, -0.00327 ),		vec3( -0.53108,  1.10813, -0.07276 ),
		vec3( -0.07367, -0.00605,  1.07602 )
	);
	color *= toneMappingExposure / 0.6;
	color = ACESInputMat * color;
	color = RRTAndODTFit( color );
	color = ACESOutputMat * color;
	return saturate( color );
}
const mat3 LINEAR_REC2020_TO_LINEAR_SRGB = mat3(
	vec3( 1.6605, - 0.1246, - 0.0182 ),
	vec3( - 0.5876, 1.1329, - 0.1006 ),
	vec3( - 0.0728, - 0.0083, 1.1187 )
);
const mat3 LINEAR_SRGB_TO_LINEAR_REC2020 = mat3(
	vec3( 0.6274, 0.0691, 0.0164 ),
	vec3( 0.3293, 0.9195, 0.0880 ),
	vec3( 0.0433, 0.0113, 0.8956 )
);
vec3 agxDefaultContrastApprox( vec3 x ) {
	vec3 x2 = x * x;
	vec3 x4 = x2 * x2;
	return + 15.5 * x4 * x2
		- 40.14 * x4 * x
		+ 31.96 * x4
		- 6.868 * x2 * x
		+ 0.4298 * x2
		+ 0.1191 * x
		- 0.00232;
}
vec3 AgXToneMapping( vec3 color ) {
	const mat3 AgXInsetMatrix = mat3(
		vec3( 0.856627153315983, 0.137318972929847, 0.11189821299995 ),
		vec3( 0.0951212405381588, 0.761241990602591, 0.0767994186031903 ),
		vec3( 0.0482516061458583, 0.101439036467562, 0.811302368396859 )
	);
	const mat3 AgXOutsetMatrix = mat3(
		vec3( 1.1271005818144368, - 0.1413297634984383, - 0.14132976349843826 ),
		vec3( - 0.11060664309660323, 1.157823702216272, - 0.11060664309660294 ),
		vec3( - 0.016493938717834573, - 0.016493938717834257, 1.2519364065950405 )
	);
	const float AgxMinEv = - 12.47393;	const float AgxMaxEv = 4.026069;
	color *= toneMappingExposure;
	color = LINEAR_SRGB_TO_LINEAR_REC2020 * color;
	color = AgXInsetMatrix * color;
	color = max( color, 1e-10 );	color = log2( color );
	color = ( color - AgxMinEv ) / ( AgxMaxEv - AgxMinEv );
	color = clamp( color, 0.0, 1.0 );
	color = agxDefaultContrastApprox( color );
	color = AgXOutsetMatrix * color;
	color = pow( max( vec3( 0.0 ), color ), vec3( 2.2 ) );
	color = LINEAR_REC2020_TO_LINEAR_SRGB * color;
	color = clamp( color, 0.0, 1.0 );
	return color;
}
vec3 NeutralToneMapping( vec3 color ) {
	const float StartCompression = 0.8 - 0.04;
	const float Desaturation = 0.15;
	color *= toneMappingExposure;
	float x = min( color.r, min( color.g, color.b ) );
	float offset = x < 0.08 ? x - 6.25 * x * x : 0.04;
	color -= offset;
	float peak = max( color.r, max( color.g, color.b ) );
	if ( peak < StartCompression ) return color;
	float d = 1. - StartCompression;
	float newPeak = 1. - d * d / ( peak + d - StartCompression );
	color *= newPeak / peak;
	float g = 1. - 1. / ( Desaturation * ( peak - newPeak ) + 1. );
	return mix( color, vec3( newPeak ), g );
}
vec3 CustomToneMapping( vec3 color ) { return color; }`,vA=`#ifdef USE_TRANSMISSION
	material.transmission = transmission;
	material.transmissionAlpha = 1.0;
	material.thickness = thickness;
	material.attenuationDistance = attenuationDistance;
	material.attenuationColor = attenuationColor;
	#ifdef USE_TRANSMISSIONMAP
		material.transmission *= texture2D( transmissionMap, vTransmissionMapUv ).r;
	#endif
	#ifdef USE_THICKNESSMAP
		material.thickness *= texture2D( thicknessMap, vThicknessMapUv ).g;
	#endif
	vec3 pos = vWorldPosition;
	vec3 v = normalize( cameraPosition - pos );
	vec3 n = transformNormalByInverseViewMatrix( normal, viewMatrix );
	vec4 transmitted = getIBLVolumeRefraction(
		n, v, material.roughness, material.diffuseContribution, material.specularColorBlended, material.specularF90,
		pos, modelMatrix, viewMatrix, projectionMatrix, material.dispersion, material.ior, material.thickness,
		material.attenuationColor, material.attenuationDistance );
	material.transmissionAlpha = mix( material.transmissionAlpha, transmitted.a, material.transmission );
	totalDiffuse = mix( totalDiffuse, transmitted.rgb, material.transmission );
#endif`,xA=`#ifdef USE_TRANSMISSION
	uniform float transmission;
	uniform float thickness;
	uniform float attenuationDistance;
	uniform vec3 attenuationColor;
	#ifdef USE_TRANSMISSIONMAP
		uniform sampler2D transmissionMap;
	#endif
	#ifdef USE_THICKNESSMAP
		uniform sampler2D thicknessMap;
	#endif
	uniform vec2 transmissionSamplerSize;
	uniform sampler2D transmissionSamplerMap;
	uniform mat4 modelMatrix;
	uniform mat4 projectionMatrix;
	varying vec3 vWorldPosition;
	float w0( float a ) {
		return ( 1.0 / 6.0 ) * ( a * ( a * ( - a + 3.0 ) - 3.0 ) + 1.0 );
	}
	float w1( float a ) {
		return ( 1.0 / 6.0 ) * ( a *  a * ( 3.0 * a - 6.0 ) + 4.0 );
	}
	float w2( float a ){
		return ( 1.0 / 6.0 ) * ( a * ( a * ( - 3.0 * a + 3.0 ) + 3.0 ) + 1.0 );
	}
	float w3( float a ) {
		return ( 1.0 / 6.0 ) * ( a * a * a );
	}
	float g0( float a ) {
		return w0( a ) + w1( a );
	}
	float g1( float a ) {
		return w2( a ) + w3( a );
	}
	float h0( float a ) {
		return - 1.0 + w1( a ) / ( w0( a ) + w1( a ) );
	}
	float h1( float a ) {
		return 1.0 + w3( a ) / ( w2( a ) + w3( a ) );
	}
	vec4 bicubic( sampler2D tex, vec2 uv, vec4 texelSize, float lod ) {
		uv = uv * texelSize.zw + 0.5;
		vec2 iuv = floor( uv );
		vec2 fuv = fract( uv );
		float g0x = g0( fuv.x );
		float g1x = g1( fuv.x );
		float h0x = h0( fuv.x );
		float h1x = h1( fuv.x );
		float h0y = h0( fuv.y );
		float h1y = h1( fuv.y );
		vec2 p0 = ( vec2( iuv.x + h0x, iuv.y + h0y ) - 0.5 ) * texelSize.xy;
		vec2 p1 = ( vec2( iuv.x + h1x, iuv.y + h0y ) - 0.5 ) * texelSize.xy;
		vec2 p2 = ( vec2( iuv.x + h0x, iuv.y + h1y ) - 0.5 ) * texelSize.xy;
		vec2 p3 = ( vec2( iuv.x + h1x, iuv.y + h1y ) - 0.5 ) * texelSize.xy;
		return g0( fuv.y ) * ( g0x * textureLod( tex, p0, lod ) + g1x * textureLod( tex, p1, lod ) ) +
			g1( fuv.y ) * ( g0x * textureLod( tex, p2, lod ) + g1x * textureLod( tex, p3, lod ) );
	}
	vec4 textureBicubic( sampler2D sampler, vec2 uv, float lod ) {
		vec2 fLodSize = vec2( textureSize( sampler, int( lod ) ) );
		vec2 cLodSize = vec2( textureSize( sampler, int( lod + 1.0 ) ) );
		vec2 fLodSizeInv = 1.0 / fLodSize;
		vec2 cLodSizeInv = 1.0 / cLodSize;
		vec4 fSample = bicubic( sampler, uv, vec4( fLodSizeInv, fLodSize ), floor( lod ) );
		vec4 cSample = bicubic( sampler, uv, vec4( cLodSizeInv, cLodSize ), ceil( lod ) );
		return mix( fSample, cSample, fract( lod ) );
	}
	vec3 getVolumeTransmissionRay( const in vec3 n, const in vec3 v, const in float thickness, const in float ior, const in mat4 modelMatrix ) {
		vec3 refractionVector = refract( - v, normalize( n ), 1.0 / ior );
		vec3 modelScale;
		modelScale.x = length( vec3( modelMatrix[ 0 ].xyz ) );
		modelScale.y = length( vec3( modelMatrix[ 1 ].xyz ) );
		modelScale.z = length( vec3( modelMatrix[ 2 ].xyz ) );
		return normalize( refractionVector ) * thickness * modelScale;
	}
	float applyIorToRoughness( const in float roughness, const in float ior ) {
		return roughness * clamp( ior * 2.0 - 2.0, 0.0, 1.0 );
	}
	vec4 getTransmissionSample( const in vec2 fragCoord, const in float roughness, const in float ior ) {
		float lod = log2( transmissionSamplerSize.x ) * applyIorToRoughness( roughness, ior );
		return textureBicubic( transmissionSamplerMap, fragCoord.xy, lod );
	}
	vec3 volumeAttenuation( const in float transmissionDistance, const in vec3 attenuationColor, const in float attenuationDistance ) {
		if ( isinf( attenuationDistance ) ) {
			return vec3( 1.0 );
		} else {
			vec3 attenuationCoefficient = -log( attenuationColor ) / attenuationDistance;
			vec3 transmittance = exp( - attenuationCoefficient * transmissionDistance );			return transmittance;
		}
	}
	vec4 getIBLVolumeRefraction( const in vec3 n, const in vec3 v, const in float roughness, const in vec3 diffuseColor,
		const in vec3 specularColor, const in float specularF90, const in vec3 position, const in mat4 modelMatrix,
		const in mat4 viewMatrix, const in mat4 projMatrix, const in float dispersion, const in float ior, const in float thickness,
		const in vec3 attenuationColor, const in float attenuationDistance ) {
		vec4 transmittedLight;
		vec3 transmittance;
		#ifdef USE_DISPERSION
			float halfSpread = ( ior - 1.0 ) * 0.025 * dispersion;
			vec3 iors = vec3( ior - halfSpread, ior, ior + halfSpread );
			for ( int i = 0; i < 3; i ++ ) {
				vec3 transmissionRay = getVolumeTransmissionRay( n, v, thickness, iors[ i ], modelMatrix );
				vec3 refractedRayExit = position + transmissionRay;
				vec4 ndcPos = projMatrix * viewMatrix * vec4( refractedRayExit, 1.0 );
				vec2 refractionCoords = ndcPos.xy / ndcPos.w;
				refractionCoords += 1.0;
				refractionCoords /= 2.0;
				vec4 transmissionSample = getTransmissionSample( refractionCoords, roughness, iors[ i ] );
				transmittedLight[ i ] = transmissionSample[ i ];
				transmittedLight.a += transmissionSample.a;
				transmittance[ i ] = diffuseColor[ i ] * volumeAttenuation( length( transmissionRay ), attenuationColor, attenuationDistance )[ i ];
			}
			transmittedLight.a /= 3.0;
		#else
			vec3 transmissionRay = getVolumeTransmissionRay( n, v, thickness, ior, modelMatrix );
			vec3 refractedRayExit = position + transmissionRay;
			vec4 ndcPos = projMatrix * viewMatrix * vec4( refractedRayExit, 1.0 );
			vec2 refractionCoords = ndcPos.xy / ndcPos.w;
			refractionCoords += 1.0;
			refractionCoords /= 2.0;
			transmittedLight = getTransmissionSample( refractionCoords, roughness, ior );
			transmittance = diffuseColor * volumeAttenuation( length( transmissionRay ), attenuationColor, attenuationDistance );
		#endif
		vec3 attenuatedColor = transmittance * transmittedLight.rgb;
		vec3 F = EnvironmentBRDF( n, v, specularColor, specularF90, roughness );
		float transmittanceFactor = ( transmittance.r + transmittance.g + transmittance.b ) / 3.0;
		return vec4( ( 1.0 - F ) * attenuatedColor, 1.0 - ( 1.0 - transmittedLight.a ) * transmittanceFactor );
	}
#endif`,SA=`#if defined( USE_UV ) || defined( USE_ANISOTROPY )
	varying vec2 vUv;
#endif
#ifdef USE_MAP
	varying vec2 vMapUv;
#endif
#ifdef USE_ALPHAMAP
	varying vec2 vAlphaMapUv;
#endif
#ifdef USE_LIGHTMAP
	varying vec2 vLightMapUv;
#endif
#ifdef USE_AOMAP
	varying vec2 vAoMapUv;
#endif
#ifdef USE_BUMPMAP
	varying vec2 vBumpMapUv;
#endif
#ifdef USE_NORMALMAP
	varying vec2 vNormalMapUv;
#endif
#ifdef USE_EMISSIVEMAP
	varying vec2 vEmissiveMapUv;
#endif
#ifdef USE_METALNESSMAP
	varying vec2 vMetalnessMapUv;
#endif
#ifdef USE_ROUGHNESSMAP
	varying vec2 vRoughnessMapUv;
#endif
#ifdef USE_ANISOTROPYMAP
	varying vec2 vAnisotropyMapUv;
#endif
#ifdef USE_CLEARCOATMAP
	varying vec2 vClearcoatMapUv;
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	varying vec2 vClearcoatNormalMapUv;
#endif
#ifdef USE_CLEARCOAT_ROUGHNESSMAP
	varying vec2 vClearcoatRoughnessMapUv;
#endif
#ifdef USE_IRIDESCENCEMAP
	varying vec2 vIridescenceMapUv;
#endif
#ifdef USE_IRIDESCENCE_THICKNESSMAP
	varying vec2 vIridescenceThicknessMapUv;
#endif
#ifdef USE_SHEEN_COLORMAP
	varying vec2 vSheenColorMapUv;
#endif
#ifdef USE_SHEEN_ROUGHNESSMAP
	varying vec2 vSheenRoughnessMapUv;
#endif
#ifdef USE_SPECULARMAP
	varying vec2 vSpecularMapUv;
#endif
#ifdef USE_SPECULAR_COLORMAP
	varying vec2 vSpecularColorMapUv;
#endif
#ifdef USE_SPECULAR_INTENSITYMAP
	varying vec2 vSpecularIntensityMapUv;
#endif
#ifdef USE_TRANSMISSIONMAP
	uniform mat3 transmissionMapTransform;
	varying vec2 vTransmissionMapUv;
#endif
#ifdef USE_THICKNESSMAP
	uniform mat3 thicknessMapTransform;
	varying vec2 vThicknessMapUv;
#endif`,yA=`#if defined( USE_UV ) || defined( USE_ANISOTROPY )
	varying vec2 vUv;
#endif
#ifdef USE_MAP
	uniform mat3 mapTransform;
	varying vec2 vMapUv;
#endif
#ifdef USE_ALPHAMAP
	uniform mat3 alphaMapTransform;
	varying vec2 vAlphaMapUv;
#endif
#ifdef USE_LIGHTMAP
	uniform mat3 lightMapTransform;
	varying vec2 vLightMapUv;
#endif
#ifdef USE_AOMAP
	uniform mat3 aoMapTransform;
	varying vec2 vAoMapUv;
#endif
#ifdef USE_BUMPMAP
	uniform mat3 bumpMapTransform;
	varying vec2 vBumpMapUv;
#endif
#ifdef USE_NORMALMAP
	uniform mat3 normalMapTransform;
	varying vec2 vNormalMapUv;
#endif
#ifdef USE_DISPLACEMENTMAP
	uniform mat3 displacementMapTransform;
	varying vec2 vDisplacementMapUv;
#endif
#ifdef USE_EMISSIVEMAP
	uniform mat3 emissiveMapTransform;
	varying vec2 vEmissiveMapUv;
#endif
#ifdef USE_METALNESSMAP
	uniform mat3 metalnessMapTransform;
	varying vec2 vMetalnessMapUv;
#endif
#ifdef USE_ROUGHNESSMAP
	uniform mat3 roughnessMapTransform;
	varying vec2 vRoughnessMapUv;
#endif
#ifdef USE_ANISOTROPYMAP
	uniform mat3 anisotropyMapTransform;
	varying vec2 vAnisotropyMapUv;
#endif
#ifdef USE_CLEARCOATMAP
	uniform mat3 clearcoatMapTransform;
	varying vec2 vClearcoatMapUv;
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	uniform mat3 clearcoatNormalMapTransform;
	varying vec2 vClearcoatNormalMapUv;
#endif
#ifdef USE_CLEARCOAT_ROUGHNESSMAP
	uniform mat3 clearcoatRoughnessMapTransform;
	varying vec2 vClearcoatRoughnessMapUv;
#endif
#ifdef USE_SHEEN_COLORMAP
	uniform mat3 sheenColorMapTransform;
	varying vec2 vSheenColorMapUv;
#endif
#ifdef USE_SHEEN_ROUGHNESSMAP
	uniform mat3 sheenRoughnessMapTransform;
	varying vec2 vSheenRoughnessMapUv;
#endif
#ifdef USE_IRIDESCENCEMAP
	uniform mat3 iridescenceMapTransform;
	varying vec2 vIridescenceMapUv;
#endif
#ifdef USE_IRIDESCENCE_THICKNESSMAP
	uniform mat3 iridescenceThicknessMapTransform;
	varying vec2 vIridescenceThicknessMapUv;
#endif
#ifdef USE_SPECULARMAP
	uniform mat3 specularMapTransform;
	varying vec2 vSpecularMapUv;
#endif
#ifdef USE_SPECULAR_COLORMAP
	uniform mat3 specularColorMapTransform;
	varying vec2 vSpecularColorMapUv;
#endif
#ifdef USE_SPECULAR_INTENSITYMAP
	uniform mat3 specularIntensityMapTransform;
	varying vec2 vSpecularIntensityMapUv;
#endif
#ifdef USE_TRANSMISSIONMAP
	uniform mat3 transmissionMapTransform;
	varying vec2 vTransmissionMapUv;
#endif
#ifdef USE_THICKNESSMAP
	uniform mat3 thicknessMapTransform;
	varying vec2 vThicknessMapUv;
#endif`,MA=`#if defined( USE_UV ) || defined( USE_ANISOTROPY )
	vUv = vec3( uv, 1 ).xy;
#endif
#ifdef USE_MAP
	vMapUv = ( mapTransform * vec3( MAP_UV, 1 ) ).xy;
#endif
#ifdef USE_ALPHAMAP
	vAlphaMapUv = ( alphaMapTransform * vec3( ALPHAMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_LIGHTMAP
	vLightMapUv = ( lightMapTransform * vec3( LIGHTMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_AOMAP
	vAoMapUv = ( aoMapTransform * vec3( AOMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_BUMPMAP
	vBumpMapUv = ( bumpMapTransform * vec3( BUMPMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_NORMALMAP
	vNormalMapUv = ( normalMapTransform * vec3( NORMALMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_DISPLACEMENTMAP
	vDisplacementMapUv = ( displacementMapTransform * vec3( DISPLACEMENTMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_EMISSIVEMAP
	vEmissiveMapUv = ( emissiveMapTransform * vec3( EMISSIVEMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_METALNESSMAP
	vMetalnessMapUv = ( metalnessMapTransform * vec3( METALNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_ROUGHNESSMAP
	vRoughnessMapUv = ( roughnessMapTransform * vec3( ROUGHNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_ANISOTROPYMAP
	vAnisotropyMapUv = ( anisotropyMapTransform * vec3( ANISOTROPYMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_CLEARCOATMAP
	vClearcoatMapUv = ( clearcoatMapTransform * vec3( CLEARCOATMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	vClearcoatNormalMapUv = ( clearcoatNormalMapTransform * vec3( CLEARCOAT_NORMALMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_CLEARCOAT_ROUGHNESSMAP
	vClearcoatRoughnessMapUv = ( clearcoatRoughnessMapTransform * vec3( CLEARCOAT_ROUGHNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_IRIDESCENCEMAP
	vIridescenceMapUv = ( iridescenceMapTransform * vec3( IRIDESCENCEMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_IRIDESCENCE_THICKNESSMAP
	vIridescenceThicknessMapUv = ( iridescenceThicknessMapTransform * vec3( IRIDESCENCE_THICKNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SHEEN_COLORMAP
	vSheenColorMapUv = ( sheenColorMapTransform * vec3( SHEEN_COLORMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SHEEN_ROUGHNESSMAP
	vSheenRoughnessMapUv = ( sheenRoughnessMapTransform * vec3( SHEEN_ROUGHNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SPECULARMAP
	vSpecularMapUv = ( specularMapTransform * vec3( SPECULARMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SPECULAR_COLORMAP
	vSpecularColorMapUv = ( specularColorMapTransform * vec3( SPECULAR_COLORMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SPECULAR_INTENSITYMAP
	vSpecularIntensityMapUv = ( specularIntensityMapTransform * vec3( SPECULAR_INTENSITYMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_TRANSMISSIONMAP
	vTransmissionMapUv = ( transmissionMapTransform * vec3( TRANSMISSIONMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_THICKNESSMAP
	vThicknessMapUv = ( thicknessMapTransform * vec3( THICKNESSMAP_UV, 1 ) ).xy;
#endif`,EA=`#if defined( USE_ENVMAP ) || defined( DISTANCE ) || defined ( USE_SHADOWMAP ) || defined ( USE_TRANSMISSION ) || NUM_SPOT_LIGHT_COORDS > 0
	vec4 worldPosition = vec4( transformed, 1.0 );
	#ifdef USE_BATCHING
		worldPosition = batchingMatrix * worldPosition;
	#endif
	#ifdef USE_INSTANCING
		worldPosition = instanceMatrix * worldPosition;
	#endif
	worldPosition = modelMatrix * worldPosition;
#endif`;const bA=`varying vec2 vUv;
uniform mat3 uvTransform;
void main() {
	vUv = ( uvTransform * vec3( uv, 1 ) ).xy;
	gl_Position = vec4( position.xy, 1.0, 1.0 );
}`,TA=`uniform sampler2D t2D;
uniform float backgroundIntensity;
varying vec2 vUv;
void main() {
	vec4 texColor = texture2D( t2D, vUv );
	#ifdef DECODE_VIDEO_TEXTURE
		texColor = vec4( mix( pow( texColor.rgb * 0.9478672986 + vec3( 0.0521327014 ), vec3( 2.4 ) ), texColor.rgb * 0.0773993808, vec3( lessThanEqual( texColor.rgb, vec3( 0.04045 ) ) ) ), texColor.w );
	#endif
	texColor.rgb *= backgroundIntensity;
	gl_FragColor = texColor;
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,AA=`varying vec3 vWorldDirection;
#include <common>
void main() {
	vWorldDirection = transformDirection( position, modelMatrix );
	#include <begin_vertex>
	#include <project_vertex>
	gl_Position.z = gl_Position.w;
}`,RA=`#ifdef ENVMAP_TYPE_CUBE
	uniform samplerCube envMap;
#elif defined( ENVMAP_TYPE_CUBE_UV )
	uniform sampler2D envMap;
#endif
uniform float backgroundBlurriness;
uniform float backgroundIntensity;
uniform mat3 backgroundRotation;
varying vec3 vWorldDirection;
#include <cube_uv_reflection_fragment>
void main() {
	#ifdef ENVMAP_TYPE_CUBE
		vec4 texColor = textureCube( envMap, backgroundRotation * vWorldDirection );
	#elif defined( ENVMAP_TYPE_CUBE_UV )
		vec4 texColor = textureCubeUV( envMap, backgroundRotation * vWorldDirection, backgroundBlurriness );
	#else
		vec4 texColor = vec4( 0.0, 0.0, 0.0, 1.0 );
	#endif
	texColor.rgb *= backgroundIntensity;
	gl_FragColor = texColor;
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,wA=`varying vec3 vWorldDirection;
#include <common>
void main() {
	vWorldDirection = transformDirection( position, modelMatrix );
	#include <begin_vertex>
	#include <project_vertex>
	gl_Position.z = gl_Position.w;
}`,CA=`uniform samplerCube tCube;
uniform float tFlip;
uniform float opacity;
varying vec3 vWorldDirection;
void main() {
	vec4 texColor = textureCube( tCube, vec3( tFlip * vWorldDirection.x, vWorldDirection.yz ) );
	gl_FragColor = texColor;
	gl_FragColor.a *= opacity;
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,NA=`#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
varying vec2 vHighPrecisionZW;
void main() {
	#include <uv_vertex>
	#include <batching_vertex>
	#include <skinbase_vertex>
	#include <morphinstance_vertex>
	#ifdef USE_DISPLACEMENTMAP
		#include <beginnormal_vertex>
		#include <morphnormal_vertex>
		#include <skinnormal_vertex>
	#endif
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vHighPrecisionZW = gl_Position.zw;
}`,DA=`#if DEPTH_PACKING == 3200
	uniform float opacity;
#endif
#include <common>
#include <packing>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
varying vec2 vHighPrecisionZW;
void main() {
	vec4 diffuseColor = vec4( 1.0 );
	#include <clipping_planes_fragment>
	#if DEPTH_PACKING == 3200
		diffuseColor.a = opacity;
	#endif
	#include <map_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <logdepthbuf_fragment>
	#ifdef USE_REVERSED_DEPTH_BUFFER
		float fragCoordZ = vHighPrecisionZW[ 0 ] / vHighPrecisionZW[ 1 ];
	#else
		float fragCoordZ = 0.5 * vHighPrecisionZW[ 0 ] / vHighPrecisionZW[ 1 ] + 0.5;
	#endif
	#if DEPTH_PACKING == 3200
		gl_FragColor = vec4( vec3( 1.0 - fragCoordZ ), opacity );
	#elif DEPTH_PACKING == 3201
		gl_FragColor = packDepthToRGBA( fragCoordZ );
	#elif DEPTH_PACKING == 3202
		gl_FragColor = vec4( packDepthToRGB( fragCoordZ ), 1.0 );
	#elif DEPTH_PACKING == 3203
		gl_FragColor = vec4( packDepthToRG( fragCoordZ ), 0.0, 1.0 );
	#endif
}`,UA=`#define DISTANCE
varying vec3 vWorldPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <batching_vertex>
	#include <skinbase_vertex>
	#include <morphinstance_vertex>
	#ifdef USE_DISPLACEMENTMAP
		#include <beginnormal_vertex>
		#include <morphnormal_vertex>
		#include <skinnormal_vertex>
	#endif
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <worldpos_vertex>
	#include <clipping_planes_vertex>
	vWorldPosition = worldPosition.xyz;
}`,LA=`#define DISTANCE
uniform vec3 referencePosition;
uniform float nearDistance;
uniform float farDistance;
varying vec3 vWorldPosition;
#include <common>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( 1.0 );
	#include <clipping_planes_fragment>
	#include <map_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	float dist = length( vWorldPosition - referencePosition );
	dist = ( dist - nearDistance ) / ( farDistance - nearDistance );
	dist = saturate( dist );
	gl_FragColor = vec4( dist, 0.0, 0.0, 1.0 );
}`,OA=`varying vec3 vWorldDirection;
#include <common>
void main() {
	vWorldDirection = transformDirection( position, modelMatrix );
	#include <begin_vertex>
	#include <project_vertex>
}`,PA=`uniform sampler2D tEquirect;
varying vec3 vWorldDirection;
#include <common>
void main() {
	vec3 direction = normalize( vWorldDirection );
	vec2 sampleUV = equirectUv( direction );
	gl_FragColor = texture2D( tEquirect, sampleUV );
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,IA=`uniform float scale;
attribute float lineDistance;
varying float vLineDistance;
#include <common>
#include <uv_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <morphtarget_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	vLineDistance = scale * lineDistance;
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <fog_vertex>
}`,BA=`uniform vec3 diffuse;
uniform float opacity;
uniform float dashSize;
uniform float totalSize;
varying float vLineDistance;
#include <common>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <fog_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	if ( mod( vLineDistance, totalSize ) > dashSize ) {
		discard;
	}
	vec3 outgoingLight = vec3( 0.0 );
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	outgoingLight = diffuseColor.rgb;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
}`,zA=`#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <envmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#if defined ( USE_ENVMAP ) || defined ( USE_SKINNING )
		#include <beginnormal_vertex>
		#include <morphnormal_vertex>
		#include <skinbase_vertex>
		#include <skinnormal_vertex>
		#include <defaultnormal_vertex>
	#endif
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <worldpos_vertex>
	#include <envmap_vertex>
	#include <fog_vertex>
}`,FA=`uniform vec3 diffuse;
uniform float opacity;
#ifndef FLAT_SHADED
	varying vec3 vNormal;
#endif
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <envmap_common_pars_fragment>
#include <envmap_pars_fragment>
#include <fog_pars_fragment>
#include <specularmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <specularmap_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	#ifdef USE_LIGHTMAP
		vec4 lightMapTexel = texture2D( lightMap, vLightMapUv );
		reflectedLight.indirectDiffuse += lightMapTexel.rgb * lightMapIntensity * RECIPROCAL_PI;
	#else
		reflectedLight.indirectDiffuse += vec3( 1.0 );
	#endif
	#include <aomap_fragment>
	reflectedLight.indirectDiffuse *= diffuseColor.rgb;
	vec3 outgoingLight = reflectedLight.indirectDiffuse;
	#include <envmap_fragment>
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,HA=`#define LAMBERT
varying vec3 vViewPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <envmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <shadowmap_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vViewPosition = - mvPosition.xyz;
	#include <worldpos_vertex>
	#include <envmap_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
}`,GA=`#define LAMBERT
uniform vec3 diffuse;
uniform vec3 emissive;
uniform float opacity;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <emissivemap_pars_fragment>
#include <cube_uv_reflection_fragment>
#include <envmap_common_pars_fragment>
#include <envmap_pars_fragment>
#include <envmap_physical_pars_fragment>
#include <fog_pars_fragment>
#include <bsdfs>
#include <lights_pars_begin>
#include <normal_pars_fragment>
#include <lights_lambert_pars_fragment>
#include <shadowmap_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <specularmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	vec3 totalEmissiveRadiance = emissive;
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <specularmap_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	#include <emissivemap_fragment>
	#include <lights_lambert_fragment>
	#include <lights_fragment_begin>
	#include <lights_fragment_maps>
	#include <lights_fragment_end>
	#include <aomap_fragment>
	vec3 outgoingLight = reflectedLight.directDiffuse + reflectedLight.indirectDiffuse + totalEmissiveRadiance;
	#include <envmap_fragment>
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,VA=`#define MATCAP
varying vec3 vViewPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <color_pars_vertex>
#include <displacementmap_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <fog_vertex>
	vViewPosition = - mvPosition.xyz;
}`,kA=`#define MATCAP
uniform vec3 diffuse;
uniform float opacity;
uniform sampler2D matcap;
varying vec3 vViewPosition;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <fog_pars_fragment>
#include <normal_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	vec3 viewDir = normalize( vViewPosition );
	vec3 x = normalize( vec3( viewDir.z, 0.0, - viewDir.x ) );
	vec3 y = cross( viewDir, x );
	vec2 uv = vec2( dot( x, normal ), dot( y, normal ) ) * 0.495 + 0.5;
	#ifdef USE_MATCAP
		vec4 matcapColor = texture2D( matcap, uv );
	#else
		vec4 matcapColor = vec4( vec3( mix( 0.2, 0.8, uv.y ) ), 1.0 );
	#endif
	vec3 outgoingLight = diffuseColor.rgb * matcapColor.rgb;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,XA=`#define NORMAL
#if defined( FLAT_SHADED ) || defined( USE_BUMPMAP ) || defined( USE_NORMALMAP_TANGENTSPACE )
	varying vec3 vViewPosition;
#endif
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphinstance_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
#if defined( FLAT_SHADED ) || defined( USE_BUMPMAP ) || defined( USE_NORMALMAP_TANGENTSPACE )
	vViewPosition = - mvPosition.xyz;
#endif
}`,WA=`#define NORMAL
uniform float opacity;
#if defined( FLAT_SHADED ) || defined( USE_BUMPMAP ) || defined( USE_NORMALMAP_TANGENTSPACE )
	varying vec3 vViewPosition;
#endif
#include <uv_pars_fragment>
#include <normal_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( 0.0, 0.0, 0.0, opacity );
	#include <clipping_planes_fragment>
	#include <logdepthbuf_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	gl_FragColor = vec4( normalize( normal ) * 0.5 + 0.5, diffuseColor.a );
	#ifdef OPAQUE
		gl_FragColor.a = 1.0;
	#endif
}`,qA=`#define PHONG
varying vec3 vViewPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <envmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <shadowmap_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphinstance_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vViewPosition = - mvPosition.xyz;
	#include <worldpos_vertex>
	#include <envmap_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
}`,YA=`#define PHONG
uniform vec3 diffuse;
uniform vec3 emissive;
uniform vec3 specular;
uniform float shininess;
uniform float opacity;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <emissivemap_pars_fragment>
#include <cube_uv_reflection_fragment>
#include <envmap_common_pars_fragment>
#include <envmap_pars_fragment>
#include <envmap_physical_pars_fragment>
#include <fog_pars_fragment>
#include <bsdfs>
#include <lights_pars_begin>
#include <normal_pars_fragment>
#include <lights_phong_pars_fragment>
#include <shadowmap_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <specularmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	vec3 totalEmissiveRadiance = emissive;
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <specularmap_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	#include <emissivemap_fragment>
	#include <lights_phong_fragment>
	#include <lights_fragment_begin>
	#include <lights_fragment_maps>
	#include <lights_fragment_end>
	#include <aomap_fragment>
	vec3 outgoingLight = reflectedLight.directDiffuse + reflectedLight.indirectDiffuse + reflectedLight.directSpecular + reflectedLight.indirectSpecular + totalEmissiveRadiance;
	#include <envmap_fragment>
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,jA=`#define STANDARD
varying vec3 vViewPosition;
#ifdef USE_TRANSMISSION
	varying vec3 vWorldPosition;
#endif
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <shadowmap_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vViewPosition = - mvPosition.xyz;
	#include <worldpos_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
#ifdef USE_TRANSMISSION
	vWorldPosition = worldPosition.xyz;
#endif
}`,ZA=`#define STANDARD
#ifdef PHYSICAL
	#define IOR
	#define USE_SPECULAR
#endif
uniform vec3 diffuse;
uniform vec3 emissive;
uniform float roughness;
uniform float metalness;
uniform float opacity;
#ifdef IOR
	uniform float ior;
#endif
#ifdef USE_SPECULAR
	uniform float specularIntensity;
	uniform vec3 specularColor;
	#ifdef USE_SPECULAR_COLORMAP
		uniform sampler2D specularColorMap;
	#endif
	#ifdef USE_SPECULAR_INTENSITYMAP
		uniform sampler2D specularIntensityMap;
	#endif
#endif
#ifdef USE_CLEARCOAT
	uniform float clearcoat;
	uniform float clearcoatRoughness;
#endif
#ifdef USE_DISPERSION
	uniform float dispersion;
#endif
#ifdef USE_RETROREFLECTION
	uniform float retroreflectivity;
#endif
#ifdef USE_IRIDESCENCE
	uniform float iridescence;
	uniform float iridescenceIOR;
	uniform float iridescenceThicknessMinimum;
	uniform float iridescenceThicknessMaximum;
#endif
#ifdef USE_SHEEN
	uniform vec3 sheenColor;
	uniform float sheenRoughness;
	#ifdef USE_SHEEN_COLORMAP
		uniform sampler2D sheenColorMap;
	#endif
	#ifdef USE_SHEEN_ROUGHNESSMAP
		uniform sampler2D sheenRoughnessMap;
	#endif
#endif
#ifdef USE_ANISOTROPY
	uniform vec2 anisotropyVector;
	#ifdef USE_ANISOTROPYMAP
		uniform sampler2D anisotropyMap;
	#endif
#endif
varying vec3 vViewPosition;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <emissivemap_pars_fragment>
#include <iridescence_fragment>
#include <cube_uv_reflection_fragment>
#include <envmap_common_pars_fragment>
#include <envmap_physical_pars_fragment>
#include <fog_pars_fragment>
#include <lights_pars_begin>
#include <normal_pars_fragment>
#include <lights_physical_pars_fragment>
#include <transmission_pars_fragment>
#include <shadowmap_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <clearcoat_pars_fragment>
#include <iridescence_pars_fragment>
#include <roughnessmap_pars_fragment>
#include <metalnessmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	vec3 totalEmissiveRadiance = emissive;
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <roughnessmap_fragment>
	#include <metalnessmap_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	#include <clearcoat_normal_fragment_begin>
	#include <clearcoat_normal_fragment_maps>
	#include <emissivemap_fragment>
	#include <lights_physical_fragment>
	#include <lights_fragment_begin>
	#include <lights_fragment_maps>
	#include <lights_fragment_end>
	#include <aomap_fragment>
	vec3 totalDiffuse = reflectedLight.directDiffuse + reflectedLight.indirectDiffuse;
	vec3 totalSpecular = reflectedLight.directSpecular + reflectedLight.indirectSpecular;
	#include <transmission_fragment>
	vec3 outgoingLight = totalDiffuse + totalSpecular + totalEmissiveRadiance;
	#ifdef USE_SHEEN
 
		outgoingLight = outgoingLight + sheenSpecularDirect + sheenSpecularIndirect;
 
 	#endif
	#ifdef USE_CLEARCOAT
		float dotNVcc = saturate( dot( geometryClearcoatNormal, geometryViewDir ) );
		vec3 Fcc = F_Schlick( material.clearcoatF0, material.clearcoatF90, dotNVcc );
		outgoingLight = outgoingLight * ( 1.0 - material.clearcoat * Fcc ) + ( clearcoatSpecularDirect + clearcoatSpecularIndirect ) * material.clearcoat;
	#endif
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,KA=`#define TOON
varying vec3 vViewPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <shadowmap_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vViewPosition = - mvPosition.xyz;
	#include <worldpos_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
}`,QA=`#define TOON
uniform vec3 diffuse;
uniform vec3 emissive;
uniform float opacity;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <emissivemap_pars_fragment>
#include <gradientmap_pars_fragment>
#include <fog_pars_fragment>
#include <bsdfs>
#include <lights_pars_begin>
#include <normal_pars_fragment>
#include <lights_toon_pars_fragment>
#include <shadowmap_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	vec3 totalEmissiveRadiance = emissive;
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	#include <emissivemap_fragment>
	#include <lights_toon_fragment>
	#include <lights_fragment_begin>
	#include <lights_fragment_maps>
	#include <lights_fragment_end>
	#include <aomap_fragment>
	vec3 outgoingLight = reflectedLight.directDiffuse + reflectedLight.indirectDiffuse + totalEmissiveRadiance;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,JA=`uniform float size;
uniform float scale;
#include <common>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <morphtarget_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
#ifdef USE_POINTS_UV
	varying vec2 vUv;
	uniform mat3 uvTransform;
#endif
void main() {
	#ifdef USE_POINTS_UV
		vUv = ( uvTransform * vec3( uv, 1 ) ).xy;
	#endif
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <project_vertex>
	gl_PointSize = size;
	#ifdef USE_SIZEATTENUATION
		bool isPerspective = isPerspectiveMatrix( projectionMatrix );
		if ( isPerspective ) gl_PointSize *= ( scale / - mvPosition.z );
	#endif
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <worldpos_vertex>
	#include <fog_vertex>
}`,$A=`uniform vec3 diffuse;
uniform float opacity;
#include <common>
#include <color_pars_fragment>
#include <map_particle_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <fog_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	vec3 outgoingLight = vec3( 0.0 );
	#include <logdepthbuf_fragment>
	#include <map_particle_fragment>
	#include <color_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	outgoingLight = diffuseColor.rgb;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
}`,e1=`#include <common>
#include <batching_pars_vertex>
#include <fog_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <shadowmap_pars_vertex>
void main() {
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphinstance_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <worldpos_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
}`,t1=`uniform vec3 color;
uniform float opacity;
#include <common>
#include <fog_pars_fragment>
#include <bsdfs>
#include <lights_pars_begin>
#include <logdepthbuf_pars_fragment>
#include <shadowmap_pars_fragment>
#include <shadowmask_pars_fragment>
void main() {
	#include <logdepthbuf_fragment>
	gl_FragColor = vec4( color, opacity * ( 1.0 - getShadowMask() ) );
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
}`,n1=`uniform float rotation;
uniform vec2 center;
#include <common>
#include <uv_pars_vertex>
#include <fog_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	vec4 mvPosition = modelViewMatrix[ 3 ];
	vec2 scale = vec2( length( modelMatrix[ 0 ].xyz ), length( modelMatrix[ 1 ].xyz ) );
	#ifndef USE_SIZEATTENUATION
		bool isPerspective = isPerspectiveMatrix( projectionMatrix );
		if ( isPerspective ) scale *= - mvPosition.z;
	#endif
	vec2 alignedPosition = ( position.xy - ( center - vec2( 0.5 ) ) ) * scale;
	vec2 rotatedPosition;
	rotatedPosition.x = cos( rotation ) * alignedPosition.x - sin( rotation ) * alignedPosition.y;
	rotatedPosition.y = sin( rotation ) * alignedPosition.x + cos( rotation ) * alignedPosition.y;
	mvPosition.xy += rotatedPosition;
	gl_Position = projectionMatrix * mvPosition;
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <fog_vertex>
}`,i1=`uniform vec3 diffuse;
uniform float opacity;
#include <common>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <fog_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	vec3 outgoingLight = vec3( 0.0 );
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	outgoingLight = diffuseColor.rgb;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
}`,mt={alphahash_fragment:bb,alphahash_pars_fragment:Tb,alphamap_fragment:Ab,alphamap_pars_fragment:Rb,alphatest_fragment:wb,alphatest_pars_fragment:Cb,aomap_fragment:Nb,aomap_pars_fragment:Db,batching_pars_vertex:Ub,batching_vertex:Lb,begin_vertex:Ob,beginnormal_vertex:Pb,bsdfs:Ib,iridescence_fragment:Bb,bumpmap_pars_fragment:zb,clipping_planes_fragment:Fb,clipping_planes_pars_fragment:Hb,clipping_planes_pars_vertex:Gb,clipping_planes_vertex:Vb,color_fragment:kb,color_pars_fragment:Xb,color_pars_vertex:Wb,color_vertex:qb,common:Yb,cube_uv_reflection_fragment:jb,defaultnormal_vertex:Zb,displacementmap_pars_vertex:Kb,displacementmap_vertex:Qb,emissivemap_fragment:Jb,emissivemap_pars_fragment:$b,colorspace_fragment:eT,colorspace_pars_fragment:tT,envmap_fragment:nT,envmap_common_pars_fragment:iT,envmap_pars_fragment:aT,envmap_pars_vertex:sT,envmap_physical_pars_fragment:gT,envmap_vertex:rT,fog_vertex:oT,fog_pars_vertex:lT,fog_fragment:cT,fog_pars_fragment:uT,gradientmap_pars_fragment:fT,lightmap_pars_fragment:dT,lights_lambert_fragment:hT,lights_lambert_pars_fragment:pT,lights_pars_begin:mT,lights_toon_fragment:_T,lights_toon_pars_fragment:vT,lights_phong_fragment:xT,lights_phong_pars_fragment:ST,lights_physical_fragment:yT,lights_physical_pars_fragment:MT,lights_fragment_begin:ET,lights_fragment_maps:bT,lights_fragment_end:TT,lightprobes_pars_fragment:AT,logdepthbuf_fragment:RT,logdepthbuf_pars_fragment:wT,logdepthbuf_pars_vertex:CT,logdepthbuf_vertex:NT,map_fragment:DT,map_pars_fragment:UT,map_particle_fragment:LT,map_particle_pars_fragment:OT,metalnessmap_fragment:PT,metalnessmap_pars_fragment:IT,morphinstance_vertex:BT,morphcolor_vertex:zT,morphnormal_vertex:FT,morphtarget_pars_vertex:HT,morphtarget_vertex:GT,normal_fragment_begin:VT,normal_fragment_maps:kT,normal_pars_fragment:XT,normal_pars_vertex:WT,normal_vertex:qT,normalmap_pars_fragment:YT,clearcoat_normal_fragment_begin:jT,clearcoat_normal_fragment_maps:ZT,clearcoat_pars_fragment:KT,iridescence_pars_fragment:QT,opaque_fragment:JT,packing:$T,premultiplied_alpha_fragment:eA,project_vertex:tA,dithering_fragment:nA,dithering_pars_fragment:iA,roughnessmap_fragment:aA,roughnessmap_pars_fragment:sA,shadowmap_pars_fragment:rA,shadowmap_pars_vertex:oA,shadowmap_vertex:lA,shadowmask_pars_fragment:cA,skinbase_vertex:uA,skinning_pars_vertex:fA,skinning_vertex:dA,skinnormal_vertex:hA,specularmap_fragment:pA,specularmap_pars_fragment:mA,tonemapping_fragment:gA,tonemapping_pars_fragment:_A,transmission_fragment:vA,transmission_pars_fragment:xA,uv_pars_fragment:SA,uv_pars_vertex:yA,uv_vertex:MA,worldpos_vertex:EA,background_vert:bA,background_frag:TA,backgroundCube_vert:AA,backgroundCube_frag:RA,cube_vert:wA,cube_frag:CA,depth_vert:NA,depth_frag:DA,distance_vert:UA,distance_frag:LA,equirect_vert:OA,equirect_frag:PA,linedashed_vert:IA,linedashed_frag:BA,meshbasic_vert:zA,meshbasic_frag:FA,meshlambert_vert:HA,meshlambert_frag:GA,meshmatcap_vert:VA,meshmatcap_frag:kA,meshnormal_vert:XA,meshnormal_frag:WA,meshphong_vert:qA,meshphong_frag:YA,meshphysical_vert:jA,meshphysical_frag:ZA,meshtoon_vert:KA,meshtoon_frag:QA,points_vert:JA,points_frag:$A,shadow_vert:e1,shadow_frag:t1,sprite_vert:n1,sprite_frag:i1},He={common:{diffuse:{value:new Dt(16777215)},opacity:{value:1},map:{value:null},mapTransform:{value:new ft},alphaMap:{value:null},alphaMapTransform:{value:new ft},alphaTest:{value:0}},specularmap:{specularMap:{value:null},specularMapTransform:{value:new ft}},envmap:{envMap:{value:null},envMapRotation:{value:new ft},reflectivity:{value:1},ior:{value:1.5},refractionRatio:{value:.98},dfgLUT:{value:null}},aomap:{aoMap:{value:null},aoMapIntensity:{value:1},aoMapTransform:{value:new ft}},lightmap:{lightMap:{value:null},lightMapIntensity:{value:1},lightMapTransform:{value:new ft}},bumpmap:{bumpMap:{value:null},bumpMapTransform:{value:new ft},bumpScale:{value:1}},normalmap:{normalMap:{value:null},normalMapTransform:{value:new ft},normalScale:{value:new Ut(1,1)}},displacementmap:{displacementMap:{value:null},displacementMapTransform:{value:new ft},displacementScale:{value:1},displacementBias:{value:0}},emissivemap:{emissiveMap:{value:null},emissiveMapTransform:{value:new ft}},metalnessmap:{metalnessMap:{value:null},metalnessMapTransform:{value:new ft}},roughnessmap:{roughnessMap:{value:null},roughnessMapTransform:{value:new ft}},gradientmap:{gradientMap:{value:null}},fog:{fogDensity:{value:25e-5},fogNear:{value:1},fogFar:{value:2e3},fogColor:{value:new Dt(16777215)}},lights:{ambientLightColor:{value:[]},lightProbe:{value:[]},sunLights:{value:[],properties:{direction:{},color:{}}},sunLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{}}},sunShadowMatrix:{value:[]},sunShadowCascade:{value:[]},directionalLights:{value:[],properties:{direction:{},color:{}}},directionalLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{}}},directionalShadowMatrix:{value:[]},spotLights:{value:[],properties:{color:{},position:{},direction:{},distance:{},coneCos:{},penumbraCos:{},decay:{}}},spotLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{}}},spotLightMap:{value:[]},spotLightMatrix:{value:[]},pointLights:{value:[],properties:{color:{},position:{},decay:{},distance:{}}},pointLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{},shadowCameraNear:{},shadowCameraFar:{}}},pointShadowMatrix:{value:[]},hemisphereLights:{value:[],properties:{direction:{},skyColor:{},groundColor:{}}},rectAreaLights:{value:[],properties:{color:{},position:{},width:{},height:{}}},ltc_1:{value:null},ltc_2:{value:null},probesSH:{value:null},probesMin:{value:new he},probesMax:{value:new he},probesResolution:{value:new he}},points:{diffuse:{value:new Dt(16777215)},opacity:{value:1},size:{value:1},scale:{value:1},map:{value:null},alphaMap:{value:null},alphaMapTransform:{value:new ft},alphaTest:{value:0},uvTransform:{value:new ft}},sprite:{diffuse:{value:new Dt(16777215)},opacity:{value:1},center:{value:new Ut(.5,.5)},rotation:{value:0},map:{value:null},mapTransform:{value:new ft},alphaMap:{value:null},alphaMapTransform:{value:new ft},alphaTest:{value:0}}},ta={basic:{uniforms:Vn([He.common,He.specularmap,He.envmap,He.aomap,He.lightmap,He.fog]),vertexShader:mt.meshbasic_vert,fragmentShader:mt.meshbasic_frag},lambert:{uniforms:Vn([He.common,He.specularmap,He.envmap,He.aomap,He.lightmap,He.emissivemap,He.bumpmap,He.normalmap,He.displacementmap,He.fog,He.lights,{emissive:{value:new Dt(0)},envMapIntensity:{value:1}}]),vertexShader:mt.meshlambert_vert,fragmentShader:mt.meshlambert_frag},phong:{uniforms:Vn([He.common,He.specularmap,He.envmap,He.aomap,He.lightmap,He.emissivemap,He.bumpmap,He.normalmap,He.displacementmap,He.fog,He.lights,{emissive:{value:new Dt(0)},specular:{value:new Dt(1118481)},shininess:{value:30},envMapIntensity:{value:1}}]),vertexShader:mt.meshphong_vert,fragmentShader:mt.meshphong_frag},standard:{uniforms:Vn([He.common,He.envmap,He.aomap,He.lightmap,He.emissivemap,He.bumpmap,He.normalmap,He.displacementmap,He.roughnessmap,He.metalnessmap,He.fog,He.lights,{emissive:{value:new Dt(0)},roughness:{value:1},metalness:{value:0},envMapIntensity:{value:1}}]),vertexShader:mt.meshphysical_vert,fragmentShader:mt.meshphysical_frag},toon:{uniforms:Vn([He.common,He.aomap,He.lightmap,He.emissivemap,He.bumpmap,He.normalmap,He.displacementmap,He.gradientmap,He.fog,He.lights,{emissive:{value:new Dt(0)}}]),vertexShader:mt.meshtoon_vert,fragmentShader:mt.meshtoon_frag},matcap:{uniforms:Vn([He.common,He.bumpmap,He.normalmap,He.displacementmap,He.fog,{matcap:{value:null}}]),vertexShader:mt.meshmatcap_vert,fragmentShader:mt.meshmatcap_frag},points:{uniforms:Vn([He.points,He.fog]),vertexShader:mt.points_vert,fragmentShader:mt.points_frag},dashed:{uniforms:Vn([He.common,He.fog,{scale:{value:1},dashSize:{value:1},totalSize:{value:2}}]),vertexShader:mt.linedashed_vert,fragmentShader:mt.linedashed_frag},depth:{uniforms:Vn([He.common,He.displacementmap]),vertexShader:mt.depth_vert,fragmentShader:mt.depth_frag},normal:{uniforms:Vn([He.common,He.bumpmap,He.normalmap,He.displacementmap,{opacity:{value:1}}]),vertexShader:mt.meshnormal_vert,fragmentShader:mt.meshnormal_frag},sprite:{uniforms:Vn([He.sprite,He.fog]),vertexShader:mt.sprite_vert,fragmentShader:mt.sprite_frag},background:{uniforms:{uvTransform:{value:new ft},t2D:{value:null},backgroundIntensity:{value:1}},vertexShader:mt.background_vert,fragmentShader:mt.background_frag},backgroundCube:{uniforms:{envMap:{value:null},backgroundBlurriness:{value:0},backgroundIntensity:{value:1},backgroundRotation:{value:new ft}},vertexShader:mt.backgroundCube_vert,fragmentShader:mt.backgroundCube_frag},cube:{uniforms:{tCube:{value:null},tFlip:{value:-1},opacity:{value:1}},vertexShader:mt.cube_vert,fragmentShader:mt.cube_frag},equirect:{uniforms:{tEquirect:{value:null}},vertexShader:mt.equirect_vert,fragmentShader:mt.equirect_frag},distance:{uniforms:Vn([He.common,He.displacementmap,{referencePosition:{value:new he},nearDistance:{value:1},farDistance:{value:1e3}}]),vertexShader:mt.distance_vert,fragmentShader:mt.distance_frag},shadow:{uniforms:Vn([He.lights,He.fog,{color:{value:new Dt(0)},opacity:{value:1}}]),vertexShader:mt.shadow_vert,fragmentShader:mt.shadow_frag}};ta.physical={uniforms:Vn([ta.standard.uniforms,{clearcoat:{value:0},clearcoatMap:{value:null},clearcoatMapTransform:{value:new ft},clearcoatNormalMap:{value:null},clearcoatNormalMapTransform:{value:new ft},clearcoatNormalScale:{value:new Ut(1,1)},clearcoatRoughness:{value:0},clearcoatRoughnessMap:{value:null},clearcoatRoughnessMapTransform:{value:new ft},dispersion:{value:0},retroreflectivity:{value:0},iridescence:{value:0},iridescenceMap:{value:null},iridescenceMapTransform:{value:new ft},iridescenceIOR:{value:1.3},iridescenceThicknessMinimum:{value:100},iridescenceThicknessMaximum:{value:400},iridescenceThicknessMap:{value:null},iridescenceThicknessMapTransform:{value:new ft},sheen:{value:0},sheenColor:{value:new Dt(0)},sheenColorMap:{value:null},sheenColorMapTransform:{value:new ft},sheenRoughness:{value:1},sheenRoughnessMap:{value:null},sheenRoughnessMapTransform:{value:new ft},transmission:{value:0},transmissionMap:{value:null},transmissionMapTransform:{value:new ft},transmissionSamplerSize:{value:new Ut},transmissionSamplerMap:{value:null},thickness:{value:0},thicknessMap:{value:null},thicknessMapTransform:{value:new ft},attenuationDistance:{value:0},attenuationColor:{value:new Dt(0)},specularColor:{value:new Dt(1,1,1)},specularColorMap:{value:null},specularColorMapTransform:{value:new ft},specularIntensity:{value:1},specularIntensityMap:{value:null},specularIntensityMapTransform:{value:new ft},anisotropyVector:{value:new Ut},anisotropyMap:{value:null},anisotropyMapTransform:{value:new ft}}]),vertexShader:mt.meshphysical_vert,fragmentShader:mt.meshphysical_frag};const gu={r:0,b:0,g:0},a1=new un,Qx=new ft;Qx.set(-1,0,0,0,1,0,0,0,1);function s1(o,t,i,r,c,f){const d=new Dt(0);let h=c===!0?0:1,m,g,x=null,_=0,v=null;function E(z){let W=z.isScene===!0?z.background:null;if(W&&W.isTexture){const C=z.backgroundBlurriness>0;W=t.get(W,C)}return W}function R(z){let W=!1;const C=E(z);C===null?y(d,h):C&&C.isColor&&(y(C,1),W=!0);const D=o.xr.getEnvironmentBlendMode();D==="additive"?i.buffers.color.setClear(0,0,0,1,f):D==="alpha-blend"&&i.buffers.color.setClear(0,0,0,0,f),(o.autoClear||W)&&(i.buffers.depth.setTest(!0),i.buffers.depth.setMask(!0),i.buffers.color.setMask(!0),o.clear(o.autoClearColor,o.autoClearDepth,o.autoClearStencil))}function N(z,W){const C=E(W);C&&(C.isCubeTexture||C.mapping===Du)?(g===void 0&&(g=new la(new oo(1,1,1),new ca({name:"BackgroundCubeMaterial",uniforms:so(ta.backgroundCube.uniforms),vertexShader:ta.backgroundCube.vertexShader,fragmentShader:ta.backgroundCube.fragmentShader,side:Kn,depthTest:!1,depthWrite:!1,fog:!1,allowOverride:!1})),g.geometry.deleteAttribute("normal"),g.geometry.deleteAttribute("uv"),g.onBeforeRender=function(D,U,I){this.matrixWorld.copyPosition(I.matrixWorld)},Object.defineProperty(g.material,"envMap",{get:function(){return this.uniforms.envMap.value}}),r.update(g)),g.material.uniforms.envMap.value=C,g.material.uniforms.backgroundBlurriness.value=W.backgroundBlurriness,g.material.uniforms.backgroundIntensity.value=W.backgroundIntensity,g.material.uniforms.backgroundRotation.value.setFromMatrix4(a1.makeRotationFromEuler(W.backgroundRotation)).transpose(),C.isCubeTexture&&C.isRenderTargetTexture===!1&&g.material.uniforms.backgroundRotation.value.premultiply(Qx),g.material.toneMapped=Ct.getTransfer(C.colorSpace)!==Yt,(x!==C||_!==C.version||v!==o.toneMapping)&&(g.material.needsUpdate=!0,x=C,_=C.version,v=o.toneMapping),g.layers.enableAll(),z.unshift(g,g.geometry,g.material,0,0,null)):C&&C.isTexture&&(m===void 0&&(m=new la(new Uu(2,2),new ca({name:"BackgroundMaterial",uniforms:so(ta.background.uniforms),vertexShader:ta.background.vertexShader,fragmentShader:ta.background.fragmentShader,side:Ks,depthTest:!1,depthWrite:!1,fog:!1,allowOverride:!1})),m.geometry.deleteAttribute("normal"),Object.defineProperty(m.material,"map",{get:function(){return this.uniforms.t2D.value}}),r.update(m)),m.material.uniforms.t2D.value=C,m.material.uniforms.backgroundIntensity.value=W.backgroundIntensity,m.material.toneMapped=Ct.getTransfer(C.colorSpace)!==Yt,C.matrixAutoUpdate===!0&&C.updateMatrix(),m.material.uniforms.uvTransform.value.copy(C.matrix),(x!==C||_!==C.version||v!==o.toneMapping)&&(m.material.needsUpdate=!0,x=C,_=C.version,v=o.toneMapping),m.layers.enableAll(),z.unshift(m,m.geometry,m.material,0,0,null))}function y(z,W){z.getRGB(gu,qx(o)),i.buffers.color.setClear(gu.r,gu.g,gu.b,W,f)}function S(){g!==void 0&&(g.geometry.dispose(),g.material.dispose(),g=void 0),m!==void 0&&(m.geometry.dispose(),m.material.dispose(),m=void 0)}return{getClearColor:function(){return d},setClearColor:function(z,W=1){d.set(z),h=W,y(d,h)},getClearAlpha:function(){return h},setClearAlpha:function(z){h=z,y(d,h)},render:R,addToRenderList:N,dispose:S}}function r1(o,t){const i=o.getParameter(o.MAX_VERTEX_ATTRIBS),r={},c=v(null);let f=c,d=!1;function h(Y,$,O,q,ne){let j=!1;const Q=_(Y,q,O,$);f!==Q&&(f=Q,g(f.object)),j=E(Y,q,O,ne),j&&R(Y,q,O,ne),ne!==null&&t.update(ne,o.ELEMENT_ARRAY_BUFFER),(j||d)&&(d=!1,C(Y,$,O,q),ne!==null&&o.bindBuffer(o.ELEMENT_ARRAY_BUFFER,t.get(ne).buffer))}function m(){return o.createVertexArray()}function g(Y){return o.bindVertexArray(Y)}function x(Y){return o.deleteVertexArray(Y)}function _(Y,$,O,q){const ne=q.wireframe===!0;let j=r[$.id];j===void 0&&(j={},r[$.id]=j);const Q=Y.isInstancedMesh===!0?Y.id:0;let pe=j[Q];pe===void 0&&(pe={},j[Q]=pe);let ce=pe[O.id];ce===void 0&&(ce={},pe[O.id]=ce);let me=ce[ne];return me===void 0&&(me=v(m()),ce[ne]=me),me}function v(Y){const $=[],O=[],q=[];for(let ne=0;ne<i;ne++)$[ne]=0,O[ne]=0,q[ne]=0;return{geometry:null,program:null,wireframe:!1,newAttributes:$,enabledAttributes:O,attributeDivisors:q,object:Y,attributes:{},index:null}}function E(Y,$,O,q){const ne=f.attributes,j=$.attributes;let Q=0;const pe=O.getAttributes();for(const ce in pe)if(pe[ce].location>=0){const be=ne[ce];let Ze=j[ce];if(Ze===void 0&&(ce==="instanceMatrix"&&Y.instanceMatrix&&(Ze=Y.instanceMatrix),ce==="instanceColor"&&Y.instanceColor&&(Ze=Y.instanceColor)),be===void 0||be.attribute!==Ze||Ze&&be.data!==Ze.data)return!0;Q++}return f.attributesNum!==Q||f.index!==q}function R(Y,$,O,q){const ne={},j=$.attributes;let Q=0;const pe=O.getAttributes();for(const ce in pe)if(pe[ce].location>=0){let be=j[ce];be===void 0&&(ce==="instanceMatrix"&&Y.instanceMatrix&&(be=Y.instanceMatrix),ce==="instanceColor"&&Y.instanceColor&&(be=Y.instanceColor));const Ze={};Ze.attribute=be,be&&be.data&&(Ze.data=be.data),ne[ce]=Ze,Q++}f.attributes=ne,f.attributesNum=Q,f.index=q}function N(){const Y=f.newAttributes;for(let $=0,O=Y.length;$<O;$++)Y[$]=0}function y(Y){S(Y,0)}function S(Y,$){const O=f.newAttributes,q=f.enabledAttributes,ne=f.attributeDivisors;O[Y]=1,q[Y]===0&&(o.enableVertexAttribArray(Y),q[Y]=1),ne[Y]!==$&&(o.vertexAttribDivisor(Y,$),ne[Y]=$)}function z(){const Y=f.newAttributes,$=f.enabledAttributes;for(let O=0,q=$.length;O<q;O++)$[O]!==Y[O]&&(o.disableVertexAttribArray(O),$[O]=0)}function W(Y,$,O,q,ne,j,Q){Q===!0?o.vertexAttribIPointer(Y,$,O,ne,j):o.vertexAttribPointer(Y,$,O,q,ne,j)}function C(Y,$,O,q){N();const ne=q.attributes,j=O.getAttributes(),Q=$.defaultAttributeValues;for(const pe in j){const ce=j[pe];if(ce.location>=0){let me=ne[pe];if(me===void 0&&(pe==="instanceMatrix"&&Y.instanceMatrix&&(me=Y.instanceMatrix),pe==="instanceColor"&&Y.instanceColor&&(me=Y.instanceColor)),me!==void 0){const be=me.normalized,Ze=me.itemSize,je=t.get(me);if(je===void 0)continue;const H=je.buffer,F=je.type,ge=je.bytesPerElement,X=F===o.INT||F===o.UNSIGNED_INT||me.gpuType===Gp;if(me.isInterleavedBufferAttribute){const ue=me.data,Ae=ue.stride,Ie=me.offset;if(ue.isInstancedInterleavedBuffer){for(let xe=0;xe<ce.locationSize;xe++)S(ce.location+xe,ue.meshPerAttribute);Y.isInstancedMesh!==!0&&q._maxInstanceCount===void 0&&(q._maxInstanceCount=ue.meshPerAttribute*ue.count)}else for(let xe=0;xe<ce.locationSize;xe++)y(ce.location+xe);o.bindBuffer(o.ARRAY_BUFFER,H);for(let xe=0;xe<ce.locationSize;xe++)W(ce.location+xe,Ze/ce.locationSize,F,be,Ae*ge,(Ie+Ze/ce.locationSize*xe)*ge,X)}else{if(me.isInstancedBufferAttribute){for(let ue=0;ue<ce.locationSize;ue++)S(ce.location+ue,me.meshPerAttribute);Y.isInstancedMesh!==!0&&q._maxInstanceCount===void 0&&(q._maxInstanceCount=me.meshPerAttribute*me.count)}else for(let ue=0;ue<ce.locationSize;ue++)y(ce.location+ue);o.bindBuffer(o.ARRAY_BUFFER,H);for(let ue=0;ue<ce.locationSize;ue++)W(ce.location+ue,Ze/ce.locationSize,F,be,Ze*ge,Ze/ce.locationSize*ue*ge,X)}}else if(Q!==void 0){const be=Q[pe];if(be!==void 0)switch(be.length){case 2:o.vertexAttrib2fv(ce.location,be);break;case 3:o.vertexAttrib3fv(ce.location,be);break;case 4:o.vertexAttrib4fv(ce.location,be);break;default:o.vertexAttrib1fv(ce.location,be)}}}}z()}function D(){L();for(const Y in r){const $=r[Y];for(const O in $){const q=$[O];for(const ne in q){const j=q[ne];for(const Q in j)x(j[Q].object),delete j[Q];delete q[ne]}}delete r[Y]}}function U(Y){if(r[Y.id]===void 0)return;const $=r[Y.id];for(const O in $){const q=$[O];for(const ne in q){const j=q[ne];for(const Q in j)x(j[Q].object),delete j[Q];delete q[ne]}}delete r[Y.id]}function I(Y){for(const $ in r){const O=r[$];for(const q in O){const ne=O[q];if(ne[Y.id]===void 0)continue;const j=ne[Y.id];for(const Q in j)x(j[Q].object),delete j[Q];delete ne[Y.id]}}}function b(Y){for(const $ in r){const O=r[$],q=Y.isInstancedMesh===!0?Y.id:0,ne=O[q];if(ne!==void 0){for(const j in ne){const Q=ne[j];for(const pe in Q)x(Q[pe].object),delete Q[pe];delete ne[j]}delete O[q],Object.keys(O).length===0&&delete r[$]}}}function L(){k(),d=!0,f!==c&&(f=c,g(f.object))}function k(){c.geometry=null,c.program=null,c.wireframe=!1}return{setup:h,reset:L,resetDefaultState:k,dispose:D,releaseStatesOfGeometry:U,releaseStatesOfObject:b,releaseStatesOfProgram:I,initAttributes:N,enableAttribute:y,disableUnusedAttributes:z}}function o1(o,t,i){let r;function c(m){r=m}function f(m,g){o.drawArrays(r,m,g),i.update(g,r,1)}function d(m,g,x){x!==0&&(o.drawArraysInstanced(r,m,g,x),i.update(g,r,x))}function h(m,g,x){if(x===0)return;t.get("WEBGL_multi_draw").multiDrawArraysWEBGL(r,m,0,g,0,x);let v=0;for(let E=0;E<x;E++)v+=g[E];i.update(v,r,1)}this.setMode=c,this.render=f,this.renderInstances=d,this.renderMultiDraw=h}function l1(o,t,i,r){let c;function f(){if(c!==void 0)return c;if(t.has("EXT_texture_filter_anisotropic")===!0){const I=t.get("EXT_texture_filter_anisotropic");c=o.getParameter(I.MAX_TEXTURE_MAX_ANISOTROPY_EXT)}else c=0;return c}function d(I){return!(I!==Ii&&r.convert(I)!==o.getParameter(o.IMPLEMENTATION_COLOR_READ_FORMAT))}function h(I){const b=I===oa&&(t.has("EXT_color_buffer_half_float")||t.has("EXT_color_buffer_float"));return!(I!==fi&&I!==ia&&!b&&r.convert(I)!==o.getParameter(o.IMPLEMENTATION_COLOR_READ_TYPE))}function m(I){if(I==="highp"){if(o.getShaderPrecisionFormat(o.VERTEX_SHADER,o.HIGH_FLOAT).precision>0&&o.getShaderPrecisionFormat(o.FRAGMENT_SHADER,o.HIGH_FLOAT).precision>0)return"highp";I="mediump"}return I==="mediump"&&o.getShaderPrecisionFormat(o.VERTEX_SHADER,o.MEDIUM_FLOAT).precision>0&&o.getShaderPrecisionFormat(o.FRAGMENT_SHADER,o.MEDIUM_FLOAT).precision>0?"mediump":"lowp"}let g=i.precision!==void 0?i.precision:"highp";const x=m(g);x!==g&&(ot("WebGLRenderer:",g,"not supported, using",x,"instead."),g=x);const _=i.logarithmicDepthBuffer===!0,v=i.reversedDepthBuffer===!0&&t.has("EXT_clip_control");i.reversedDepthBuffer===!0&&v===!1&&ot("WebGLRenderer: Unable to use reversed depth buffer due to missing EXT_clip_control extension. Fallback to default depth buffer.");const E=o.getParameter(o.MAX_TEXTURE_IMAGE_UNITS),R=o.getParameter(o.MAX_VERTEX_TEXTURE_IMAGE_UNITS),N=o.getParameter(o.MAX_TEXTURE_SIZE),y=o.getParameter(o.MAX_CUBE_MAP_TEXTURE_SIZE),S=o.getParameter(o.MAX_VERTEX_ATTRIBS),z=o.getParameter(o.MAX_VERTEX_UNIFORM_VECTORS),W=o.getParameter(o.MAX_VARYING_VECTORS),C=o.getParameter(o.MAX_FRAGMENT_UNIFORM_VECTORS),D=o.getParameter(o.MAX_SAMPLES),U=o.getParameter(o.SAMPLES);return{isWebGL2:!0,getMaxAnisotropy:f,getMaxPrecision:m,textureFormatReadable:d,textureTypeReadable:h,precision:g,logarithmicDepthBuffer:_,reversedDepthBuffer:v,maxTextures:E,maxVertexTextures:R,maxTextureSize:N,maxCubemapSize:y,maxAttributes:S,maxVertexUniforms:z,maxVaryings:W,maxFragmentUniforms:C,maxSamples:D,samples:U}}function c1(o){const t=this;let i=null,r=0,c=!1,f=!1;const d=new _s,h=new ft,m={value:null,needsUpdate:!1};this.uniform=m,this.numPlanes=0,this.numIntersection=0,this.init=function(_,v){const E=_.length!==0||v||r!==0||c;return c=v,r=_.length,E},this.beginShadows=function(){f=!0,x(null)},this.endShadows=function(){f=!1},this.setGlobalState=function(_,v){i=x(_,v,0)},this.setState=function(_,v,E){const R=_.clippingPlanes,N=_.clipIntersection,y=_.clipShadows,S=o.get(_);if(!c||R===null||R.length===0||f&&!y)f?x(null):g();else{const z=f?0:r,W=z*4;let C=S.clippingState||null;m.value=C,C=x(R,v,W,E);for(let D=0;D!==W;++D)C[D]=i[D];S.clippingState=C,this.numIntersection=N?this.numPlanes:0,this.numPlanes+=z}};function g(){m.value!==i&&(m.value=i,m.needsUpdate=r>0),t.numPlanes=r,t.numIntersection=0}function x(_,v,E,R){const N=_!==null?_.length:0;let y=null;if(N!==0){if(y=m.value,R!==!0||y===null){const S=E+N*4,z=v.matrixWorldInverse;h.getNormalMatrix(z),(y===null||y.length<S)&&(y=new Float32Array(S));for(let W=0,C=E;W!==N;++W,C+=4)d.copy(_[W]).applyMatrix4(z,h),d.normal.toArray(y,C),y[C+3]=d.constant}m.value=y,m.needsUpdate=!0}return t.numPlanes=N,t.numIntersection=0,y}}const to=4,u1=6,f1=20,d1=256,hl=new Zx,Wv=new Dt;let Vh=null,kh=0,Xh=0,Wh=!1;const h1=new he,Ys=new he;class qv{constructor(t){this._renderer=t,this._pingPongRenderTarget=null,this._lodMax=0,this._cubeSize=0,this._sizeLods=[],this._lodMeshes=[],this._backgroundBox=null,this._cubemapMaterial=null,this._equirectMaterial=null,this._blurMaterial=null,this._ggxMaterial=null}fromScene(t,i=0,r=.1,c=100,f={}){const{size:d=256,position:h=h1}=f;Vh=this._renderer.getRenderTarget(),kh=this._renderer.getActiveCubeFace(),Xh=this._renderer.getActiveMipmapLevel(),Wh=this._renderer.xr.enabled,this._renderer.xr.enabled=!1,this._setSize(d);const m=this._allocateTargets();return m.depthBuffer=!0,this._sceneToCubeUV(t,r,c,m,h),i>0&&this._blur(m,0,0,i),this._applyPMREM(m),this._cleanup(m),m}fromEquirectangular(t,i=null){return this._fromTexture(t,i)}fromCubemap(t,i=null){return this._fromTexture(t,i)}compileCubemapShader(){this._cubemapMaterial===null&&(this._cubemapMaterial=Zv(),this._compileMaterial(this._cubemapMaterial))}compileEquirectangularShader(){this._equirectMaterial===null&&(this._equirectMaterial=jv(),this._compileMaterial(this._equirectMaterial))}dispose(){this._dispose(),this._cubemapMaterial!==null&&this._cubemapMaterial.dispose(),this._equirectMaterial!==null&&this._equirectMaterial.dispose(),this._backgroundBox!==null&&(this._backgroundBox.geometry.dispose(),this._backgroundBox.material.dispose())}_setSize(t){this._lodMax=Math.floor(Math.log2(t)),this._cubeSize=Math.pow(2,this._lodMax)}_dispose(){this._blurMaterial!==null&&this._blurMaterial.dispose(),this._ggxMaterial!==null&&this._ggxMaterial.dispose(),this._pingPongRenderTarget!==null&&this._pingPongRenderTarget.dispose();for(let t=0;t<this._lodMeshes.length;t++)this._lodMeshes[t].geometry.dispose()}_cleanup(t){this._renderer.setRenderTarget(Vh,kh,Xh),this._renderer.xr.enabled=Wh,t.scissorTest=!1,$r(t,0,0,t.width,t.height)}_fromTexture(t,i){t.mapping===Qs||t.mapping===ao?this._setSize(t.image.length===0?16:t.image[0].width||t.image[0].image.width):this._setSize(t.image.width/4),Vh=this._renderer.getRenderTarget(),kh=this._renderer.getActiveCubeFace(),Xh=this._renderer.getActiveMipmapLevel(),Wh=this._renderer.xr.enabled,this._renderer.xr.enabled=!1;const r=i||this._allocateTargets();return this._textureToCubeUV(t,r),this._applyPMREM(r),this._cleanup(r),r}_allocateTargets(){const t=3*Math.max(this._cubeSize,112),i=4*this._cubeSize,r={magFilter:In,minFilter:In,generateMipmaps:!1,type:oa,format:Ii,colorSpace:Au,depthBuffer:!1},c=Yv(t,i,r);if(this._pingPongRenderTarget===null||this._pingPongRenderTarget.width!==t||this._pingPongRenderTarget.height!==i){this._pingPongRenderTarget!==null&&this._dispose(),this._pingPongRenderTarget=Yv(t,i,r);const{_lodMax:f}=this;({lodMeshes:this._lodMeshes,sizeLods:this._sizeLods}=p1(f)),this._blurMaterial=g1(f,t,i),this._ggxMaterial=m1(f,t,i)}return c}_compileMaterial(t){const i=new la(new ua,t);this._renderer.compile(i,hl)}_sceneToCubeUV(t,i,r,c,f){const m=new ui(90,1,i,r),g=[1,-1,1,1,1,1],x=[1,1,1,-1,-1,-1],_=this._renderer,v=_.autoClear,E=_.toneMapping;_.getClearColor(Wv),_.toneMapping=sa,_.autoClear=!1,_.state.buffers.depth.getReversed()&&(_.setRenderTarget(c),_.clearDepth(),_.setRenderTarget(null)),this._backgroundBox===null&&(this._backgroundBox=new la(new oo,new kx({name:"PMREM.Background",side:Kn,depthWrite:!1,depthTest:!1})));const N=this._backgroundBox,y=N.material;let S=!1;const z=t.background;z?z.isColor&&(y.color.copy(z),t.background=null,S=!0):(y.color.copy(Wv),S=!0);for(let W=0;W<6;W++){const C=W%3;C===0?(m.up.set(0,g[W],0),m.position.set(f.x,f.y,f.z),m.lookAt(f.x+x[W],f.y,f.z)):C===1?(m.up.set(0,0,g[W]),m.position.set(f.x,f.y,f.z),m.lookAt(f.x,f.y+x[W],f.z)):(m.up.set(0,g[W],0),m.position.set(f.x,f.y,f.z),m.lookAt(f.x,f.y,f.z+x[W]));const D=this._cubeSize;$r(c,C*D,W>2?D:0,D,D),_.setRenderTarget(c),S&&_.render(N,m),_.render(t,m)}_.toneMapping=E,_.autoClear=v,t.background=z}_textureToCubeUV(t,i){const r=this._renderer,c=t.mapping===Qs||t.mapping===ao;c?(this._cubemapMaterial===null&&(this._cubemapMaterial=Zv()),this._cubemapMaterial.uniforms.flipEnvMap.value=t.isRenderTargetTexture===!1?-1:1):this._equirectMaterial===null&&(this._equirectMaterial=jv());const f=c?this._cubemapMaterial:this._equirectMaterial,d=this._lodMeshes[0];d.material=f;const h=f.uniforms;h.envMap.value=t;const m=this._cubeSize;$r(i,0,0,3*m,2*m),r.setRenderTarget(i),r.render(d,hl)}_applyPMREM(t){const i=this._renderer,r=i.autoClear;i.autoClear=!1;const c=this._lodMeshes.length;for(let f=1;f<c;f++)this._applyGGXFilter(t,f-1,f);i.autoClear=r}_applyGGXFilter(t,i,r){const c=this._renderer,f=this._pingPongRenderTarget,d=this._ggxMaterial,h=this._lodMeshes[r];h.material=d;const m=d.uniforms,g=r/(this._lodMeshes.length-1),x=i/(this._lodMeshes.length-1),_=Math.sqrt(g*g-x*x),v=g*1.25,E=_*v,{_lodMax:R}=this,N=this._sizeLods[r],y=3*N*(r>R-to?r-R+to:0),S=4*(this._cubeSize-N);m.envMap.value=t.texture,m.roughness.value=E,m.mipInt.value=R-i,$r(f,y,S,3*N,2*N),c.setRenderTarget(f),c.render(h,hl),m.envMap.value=f.texture,m.roughness.value=0,m.mipInt.value=R-r,$r(t,y,S,3*N,2*N),c.setRenderTarget(t),c.render(h,hl)}_blur(t,i,r,c){const f=this._pingPongRenderTarget,d=Math.min(c,Math.PI)/Math.SQRT2;this._blurPass(t,f,i,r,d),this._blurPass(f,t,r,r,d)}_blurPass(t,i,r,c,f){const d=this._renderer,h=this._blurMaterial,m=this._lodMeshes[c];m.material=h;const g=h.uniforms;g.envMap.value=t.texture,g.sigma.value=f,g.mipInt.value=this._lodMax-r;const x=this._sizeLods[c],_=3*x*(c>this._lodMax-to?c-this._lodMax+to:0),v=4*(this._cubeSize-x);$r(i,_,v,3*x,2*x),d.setRenderTarget(i),d.render(m,hl)}}function p1(o){const t=[],i=[];let r=o;const c=o-to+1+u1;for(let f=0;f<c;f++){const d=Math.pow(2,r);t.push(d);const h=1/(d-2),m=-h,g=1+h,x=[m,m,g,m,g,g,m,m,g,g,m,g],_=6,v=6,E=3,R=new Float32Array(E*v*_),N=new Float32Array(E*v*_);for(let S=0;S<_;S++){const z=S%3*2/3-1,W=S>2?0:-1,C=[z,W,0,z+2/3,W,0,z+2/3,W+1,0,z,W,0,z+2/3,W+1,0,z,W+1,0];R.set(C,E*v*S);for(let D=0;D<v;D++){const U=x[D*2]*2-1,I=x[D*2+1]*2-1;S===0?Ys.set(1,I,U):S===1?Ys.set(-U,1,-I):S===2?Ys.set(-U,I,1):S===3?Ys.set(-1,I,-U):S===4?Ys.set(-U,-1,I):Ys.set(U,I,-1),Ys.toArray(N,(S*v+D)*E)}}const y=new ua;y.setAttribute("position",new Pa(R,E)),y.setAttribute("outputDirection",new Pa(N,E)),i.push(new la(y,null)),r>to&&r--}return{lodMeshes:i,sizeLods:t}}function Yv(o,t,i){const r=new Bi(o,t,i);return r.texture.mapping=Du,r.texture.name="PMREM.cubeUv",r.scissorTest=!0,r}function $r(o,t,i,r,c){o.viewport.set(t,i,r,c),o.scissor.set(t,i,r,c)}function m1(o,t,i){return new ca({name:"PMREMGGXConvolution",defines:{GGX_SAMPLES:d1,CUBEUV_TEXEL_WIDTH:1/t,CUBEUV_TEXEL_HEIGHT:1/i,CUBEUV_MAX_MIP:`${o}.0`},uniforms:{envMap:{value:null},roughness:{value:0},mipInt:{value:0}},vertexShader:Lu(),fragmentShader:`

			precision highp float;
			precision highp int;

			varying vec3 vOutputDirection;

			uniform sampler2D envMap;
			uniform float roughness;
			uniform float mipInt;

			#define ENVMAP_TYPE_CUBE_UV
			#include <cube_uv_reflection_fragment>

			#define PI 3.14159265359

			// Van der Corput radical inverse
			float radicalInverse_VdC(uint bits) {
				bits = (bits << 16u) | (bits >> 16u);
				bits = ((bits & 0x55555555u) << 1u) | ((bits & 0xAAAAAAAAu) >> 1u);
				bits = ((bits & 0x33333333u) << 2u) | ((bits & 0xCCCCCCCCu) >> 2u);
				bits = ((bits & 0x0F0F0F0Fu) << 4u) | ((bits & 0xF0F0F0F0u) >> 4u);
				bits = ((bits & 0x00FF00FFu) << 8u) | ((bits & 0xFF00FF00u) >> 8u);
				return float(bits) * 2.3283064365386963e-10; // / 0x100000000
			}

			// Hammersley sequence
			vec2 hammersley(uint i, uint N) {
				return vec2(float(i) / float(N), radicalInverse_VdC(i));
			}

			// GGX VNDF importance sampling (Eric Heitz 2018)
			// "Sampling the GGX Distribution of Visible Normals"
			// https://jcgt.org/published/0007/04/01/
			vec3 importanceSampleGGX_VNDF(vec2 Xi, vec3 V, float roughness) {
				float alpha = roughness * roughness;

				// Section 4.1: Orthonormal basis
				vec3 T1 = vec3(1.0, 0.0, 0.0);
				vec3 T2 = cross(V, T1);

				// Section 4.2: Parameterization of projected area
				float r = sqrt(Xi.x);
				float phi = 2.0 * PI * Xi.y;
				float t1 = r * cos(phi);
				float t2 = r * sin(phi);
				float s = 0.5 * (1.0 + V.z);
				t2 = (1.0 - s) * sqrt(1.0 - t1 * t1) + s * t2;

				// Section 4.3: Reprojection onto hemisphere
				vec3 Nh = t1 * T1 + t2 * T2 + sqrt(max(0.0, 1.0 - t1 * t1 - t2 * t2)) * V;

				// Section 3.4: Transform back to ellipsoid configuration
				return normalize(vec3(alpha * Nh.x, alpha * Nh.y, max(0.0, Nh.z)));
			}

			void main() {
				vec3 N = normalize(vOutputDirection);
				vec3 V = N; // Assume view direction equals normal for pre-filtering

				vec3 prefilteredColor = vec3(0.0);
				float totalWeight = 0.0;

				// For very low roughness, just sample the environment directly
				if (roughness < 0.001) {
					gl_FragColor = vec4(bilinearCubeUV(envMap, N, mipInt), 1.0);
					return;
				}

				// Tangent space basis for VNDF sampling
				vec3 up = abs(N.z) < 0.999 ? vec3(0.0, 0.0, 1.0) : vec3(1.0, 0.0, 0.0);
				vec3 tangent = normalize(cross(up, N));
				vec3 bitangent = cross(N, tangent);

				for(uint i = 0u; i < uint(GGX_SAMPLES); i++) {
					vec2 Xi = hammersley(i, uint(GGX_SAMPLES));

					// For PMREM, V = N, so in tangent space V is always (0, 0, 1)
					vec3 H_tangent = importanceSampleGGX_VNDF(Xi, vec3(0.0, 0.0, 1.0), roughness);

					// Transform H back to world space
					vec3 H = normalize(tangent * H_tangent.x + bitangent * H_tangent.y + N * H_tangent.z);
					vec3 L = normalize(2.0 * dot(V, H) * H - V);

					float NdotL = max(dot(N, L), 0.0);

					if(NdotL > 0.0) {
						// Sample environment at fixed mip level
						// VNDF importance sampling handles the distribution filtering
						vec3 sampleColor = bilinearCubeUV(envMap, L, mipInt);

						// Weight by NdotL for the split-sum approximation
						// VNDF PDF naturally accounts for the visible microfacet distribution
						prefilteredColor += sampleColor * NdotL;
						totalWeight += NdotL;
					}
				}

				if (totalWeight > 0.0) {
					prefilteredColor = prefilteredColor / totalWeight;
				}

				gl_FragColor = vec4(prefilteredColor, 1.0);
			}
		`,blending:La,depthTest:!1,depthWrite:!1})}function g1(o,t,i){return new ca({name:"SphericalGaussianBlur",defines:{SAMPLES:f1,CUBEUV_TEXEL_WIDTH:1/t,CUBEUV_TEXEL_HEIGHT:1/i,CUBEUV_MAX_MIP:`${o}.0`},uniforms:{envMap:{value:null},sigma:{value:0},mipInt:{value:0}},vertexShader:Lu(),fragmentShader:`

			precision highp float;
			precision highp int;

			varying vec3 vOutputDirection;

			uniform sampler2D envMap;
			uniform float sigma;
			uniform float mipInt;

			#define ENVMAP_TYPE_CUBE_UV
			#include <cube_uv_reflection_fragment>

			#define PI 3.14159265359
			#define GOLDEN_ANGLE 2.39996322973

			void main() {

				if ( sigma == 0.0 ) {

					gl_FragColor = vec4( bilinearCubeUV( envMap, vOutputDirection, mipInt ), 1.0 );
					return;

				}

				vec3 outputDirection = normalize( vOutputDirection );

				vec3 up = abs( outputDirection.z ) < 0.999 ? vec3( 0.0, 0.0, 1.0 ) : vec3( 1.0, 0.0, 0.0 );
				vec3 tangent = normalize( cross( up, outputDirection ) );
				vec3 bitangent = cross( outputDirection, tangent );

				// Truncate the kernel at three standard deviations or at the antipode.
				float thetaMax = min( 3.0 * sigma, PI );
				float truncation = 1.0 - exp( - 0.5 * thetaMax * thetaMax / ( sigma * sigma ) );

				vec3 accumColor = vec3( 0.0 );
				float accumWeight = 0.0;

				for ( int i = 0; i < SAMPLES; i ++ ) {

					// Stratified inverse-CDF sampling of the Gaussian, placed on a golden-angle spiral.
					float stratum = ( float( i ) + 0.5 ) / float( SAMPLES );
					float theta = sigma * sqrt( - 2.0 * log( 1.0 - stratum * truncation ) );
					float phi = float( i ) * GOLDEN_ANGLE;

					vec3 offset = cos( phi ) * tangent + sin( phi ) * bitangent;
					vec3 sampleDirection = cos( theta ) * outputDirection + sin( theta ) * offset;

					// Correct the planar sample density to solid angle.
					float weight = sin( theta ) / theta;

					accumColor += weight * bilinearCubeUV( envMap, sampleDirection, mipInt );
					accumWeight += weight;

				}

				gl_FragColor = vec4( accumColor / accumWeight, 1.0 );

			}
		`,blending:La,depthTest:!1,depthWrite:!1})}function jv(){return new ca({name:"EquirectangularToCubeUV",uniforms:{envMap:{value:null}},vertexShader:Lu(),fragmentShader:`

			precision mediump float;
			precision mediump int;

			varying vec3 vOutputDirection;

			uniform sampler2D envMap;

			#include <common>

			void main() {

				vec3 outputDirection = normalize( vOutputDirection );
				vec2 uv = equirectUv( outputDirection );

				gl_FragColor = vec4( texture2D ( envMap, uv ).rgb, 1.0 );

			}
		`,blending:La,depthTest:!1,depthWrite:!1})}function Zv(){return new ca({name:"CubemapToCubeUV",uniforms:{envMap:{value:null},flipEnvMap:{value:-1}},vertexShader:Lu(),fragmentShader:`

			precision mediump float;
			precision mediump int;

			uniform float flipEnvMap;

			varying vec3 vOutputDirection;

			uniform samplerCube envMap;

			void main() {

				gl_FragColor = textureCube( envMap, vec3( flipEnvMap * vOutputDirection.x, vOutputDirection.yz ) );

			}
		`,blending:La,depthTest:!1,depthWrite:!1})}function Lu(){return`

		precision mediump float;
		precision mediump int;

		attribute vec3 outputDirection;

		varying vec3 vOutputDirection;

		void main() {

			vOutputDirection = outputDirection;
			gl_Position = vec4( position, 1.0 );

		}
	`}class Jx extends Bi{constructor(t=1,i={}){super(t,t,i),this.isWebGLCubeRenderTarget=!0;const r={width:t,height:t,depth:1},c=[r,r,r,r,r,r];this.texture=new Xx(c),this._setTextureOptions(i),this.texture.isRenderTargetTexture=!0}fromEquirectangularTexture(t,i){this.texture.type=i.type,this.texture.colorSpace=i.colorSpace,this.texture.generateMipmaps=i.generateMipmaps,this.texture.minFilter=i.minFilter,this.texture.magFilter=i.magFilter;const r={uniforms:{tEquirect:{value:null}},vertexShader:`

				varying vec3 vWorldDirection;

				vec3 transformDirection( in vec3 dir, in mat4 matrix ) {

					return normalize( ( matrix * vec4( dir, 0.0 ) ).xyz );

				}

				void main() {

					vWorldDirection = transformDirection( position, modelMatrix );

					#include <begin_vertex>
					#include <project_vertex>

				}
			`,fragmentShader:`

				uniform sampler2D tEquirect;

				varying vec3 vWorldDirection;

				#include <common>

				void main() {

					vec3 direction = normalize( vWorldDirection );

					vec2 sampleUV = equirectUv( direction );

					gl_FragColor = texture2D( tEquirect, sampleUV );

				}
			`},c=new oo(5,5,5),f=new ca({name:"CubemapFromEquirect",uniforms:so(r.uniforms),vertexShader:r.vertexShader,fragmentShader:r.fragmentShader,side:Kn,blending:La});f.uniforms.tEquirect.value=i;const d=new la(c,f),h=i.minFilter;return i.minFilter===js&&(i.minFilter=In),new Sb(1,10,this).update(t,d),i.minFilter=h,d.geometry.dispose(),d.material.dispose(),this}clear(t,i=!0,r=!0,c=!0){const f=t.getRenderTarget();for(let d=0;d<6;d++)t.setRenderTarget(this,d),t.clear(i,r,c);t.setRenderTarget(f)}}function _1(o){let t=new WeakMap,i=new WeakMap,r=null;function c(v,E=!1){return v==null?null:E?d(v):f(v)}function f(v){if(v&&v.isTexture){const E=v.mapping;if(E===ph||E===mh)if(t.has(v)){const R=t.get(v).texture;return h(R,v.mapping)}else{const R=v.image;if(R&&R.height>0){const N=new Jx(R.height);return N.fromEquirectangularTexture(o,v),t.set(v,N),v.addEventListener("dispose",g),h(N.texture,v.mapping)}else return null}}return v}function d(v){if(v&&v.isTexture){const E=v.mapping,R=E===ph||E===mh,N=E===Qs||E===ao;if(R||N){let y=i.get(v);const S=y!==void 0?y.texture.pmremVersion:0;if(v.isRenderTargetTexture&&v.pmremVersion!==S)return r===null&&(r=new qv(o)),y=R?r.fromEquirectangular(v,y):r.fromCubemap(v,y),y.texture.pmremVersion=v.pmremVersion,i.set(v,y),y.texture;if(y!==void 0)return y.texture;{const z=v.image;return R&&z&&z.height>0||N&&z&&m(z)?(r===null&&(r=new qv(o)),y=R?r.fromEquirectangular(v):r.fromCubemap(v),y.texture.pmremVersion=v.pmremVersion,i.set(v,y),v.addEventListener("dispose",x),y.texture):null}}}return v}function h(v,E){return E===ph?v.mapping=Qs:E===mh&&(v.mapping=ao),v}function m(v){let E=0;const R=6;for(let N=0;N<R;N++)v[N]!==void 0&&E++;return E===R}function g(v){const E=v.target;E.removeEventListener("dispose",g);const R=t.get(E);R!==void 0&&(t.delete(E),R.dispose())}function x(v){const E=v.target;E.removeEventListener("dispose",x);const R=i.get(E);R!==void 0&&(i.delete(E),R.dispose())}function _(){t=new WeakMap,i=new WeakMap,r!==null&&(r.dispose(),r=null)}return{get:c,dispose:_}}function v1(o){const t={};function i(r){if(t[r]!==void 0)return t[r];const c=o.getExtension(r);return t[r]=c,c}return{has:function(r){return i(r)!==null},init:function(){i("EXT_color_buffer_float"),i("WEBGL_clip_cull_distance"),i("OES_texture_float_linear"),i("EXT_color_buffer_half_float"),i("WEBGL_multisampled_render_to_texture"),i("WEBGL_render_shared_exponent")},get:function(r){const c=i(r);return c===null&&no("WebGLRenderer: "+r+" extension not supported."),c}}}function x1(o,t,i,r){const c={},f=new WeakMap;function d(_){const v=_.target;v.index!==null&&t.remove(v.index);for(const R in v.attributes)t.remove(v.attributes[R]);v.removeEventListener("dispose",d),delete c[v.id];const E=f.get(v);E&&(t.remove(E),f.delete(v)),r.releaseStatesOfGeometry(v),v.isInstancedBufferGeometry===!0&&delete v._maxInstanceCount,i.memory.geometries--}function h(_,v){return c[v.id]===!0||(v.addEventListener("dispose",d),c[v.id]=!0,i.memory.geometries++),v}function m(_){const v=_.attributes;for(const E in v)t.update(v[E],o.ARRAY_BUFFER)}function g(_){const v=[],E=_.index,R=_.attributes.position;let N=0;if(R===void 0)return;if(E!==null){const z=E.array;N=E.version;for(let W=0,C=z.length;W<C;W+=3){const D=z[W+0],U=z[W+1],I=z[W+2];v.push(D,U,U,I,I,D)}}else{const z=R.array;N=R.version;for(let W=0,C=z.length/3-1;W<C;W+=3){const D=W+0,U=W+1,I=W+2;v.push(D,U,U,I,I,D)}}const y=new(R.count>=65535?Vx:Gx)(v,1);y.version=N;const S=f.get(_);S&&t.remove(S),f.set(_,y)}function x(_){const v=f.get(_);if(v){const E=_.index;E!==null&&v.version<E.version&&g(_)}else g(_);return f.get(_)}return{get:h,update:m,getWireframeAttribute:x}}function S1(o,t,i){let r;function c(_){r=_}let f,d;function h(_){f=_.type,d=_.bytesPerElement}function m(_,v){o.drawElements(r,v,f,_*d),i.update(v,r,1)}function g(_,v,E){E!==0&&(o.drawElementsInstanced(r,v,f,_*d,E),i.update(v,r,E))}function x(_,v,E){if(E===0)return;t.get("WEBGL_multi_draw").multiDrawElementsWEBGL(r,v,0,f,_,0,E);let N=0;for(let y=0;y<E;y++)N+=v[y];i.update(N,r,1)}this.setMode=c,this.setIndex=h,this.render=m,this.renderInstances=g,this.renderMultiDraw=x}function y1(o){const t={geometries:0,textures:0},i={frame:0,calls:0,triangles:0,points:0,lines:0};function r(f,d,h){switch(i.calls++,d){case o.TRIANGLES:i.triangles+=h*(f/3);break;case o.LINES:i.lines+=h*(f/2);break;case o.LINE_STRIP:i.lines+=h*(f-1);break;case o.LINE_LOOP:i.lines+=h*f;break;case o.POINTS:i.points+=h*f;break;default:It("WebGLInfo: Unknown draw mode:",d);break}}function c(){i.calls=0,i.triangles=0,i.points=0,i.lines=0}return{memory:t,render:i,programs:null,autoReset:!0,reset:c,update:r}}function M1(o,t,i){const r=new WeakMap,c=new sn;function f(d,h,m){const g=d.morphTargetInfluences,x=h.morphAttributes.position||h.morphAttributes.normal||h.morphAttributes.color,_=x!==void 0?x.length:0;let v=r.get(h);if(v===void 0||v.count!==_){let k=function(){b.dispose(),r.delete(h),h.removeEventListener("dispose",k)};var E=k;v!==void 0&&v.texture.dispose();const R=h.morphAttributes.position!==void 0,N=h.morphAttributes.normal!==void 0,y=h.morphAttributes.color!==void 0,S=h.morphAttributes.position||[],z=h.morphAttributes.normal||[],W=h.morphAttributes.color||[];let C=0;R===!0&&(C=1),N===!0&&(C=2),y===!0&&(C=3);let D=h.attributes.position.count*C,U=1;D>t.maxTextureSize&&(U=Math.ceil(D/t.maxTextureSize),D=t.maxTextureSize);const I=new Float32Array(D*U*4*_),b=new zx(I,D,U,_);b.type=ia,b.needsUpdate=!0;const L=C*4;for(let Y=0;Y<_;Y++){const $=S[Y],O=z[Y],q=W[Y],ne=D*U*4*Y;for(let j=0;j<$.count;j++){const Q=j*L;R===!0&&(c.fromBufferAttribute($,j),I[ne+Q+0]=c.x,I[ne+Q+1]=c.y,I[ne+Q+2]=c.z,I[ne+Q+3]=0),N===!0&&(c.fromBufferAttribute(O,j),I[ne+Q+4]=c.x,I[ne+Q+5]=c.y,I[ne+Q+6]=c.z,I[ne+Q+7]=0),y===!0&&(c.fromBufferAttribute(q,j),I[ne+Q+8]=c.x,I[ne+Q+9]=c.y,I[ne+Q+10]=c.z,I[ne+Q+11]=q.itemSize===4?c.w:1)}}v={count:_,texture:b,size:new Ut(D,U)},r.set(h,v),h.addEventListener("dispose",k)}if(d.isInstancedMesh===!0&&d.morphTexture!==null)m.getUniforms().setValue(o,"morphTexture",d.morphTexture,i);else{let R=0;for(let y=0;y<g.length;y++)R+=g[y];const N=h.morphTargetsRelative?1:1-R;m.getUniforms().setValue(o,"morphTargetBaseInfluence",N),m.getUniforms().setValue(o,"morphTargetInfluences",g)}m.getUniforms().setValue(o,"morphTargetsTexture",v.texture,i),m.getUniforms().setValue(o,"morphTargetsTextureSize",v.size)}return{update:f}}function E1(o,t,i,r,c){let f=new WeakMap;function d(g){const x=c.render.frame,_=g.geometry,v=t.get(g,_);if(f.get(v)!==x&&(t.update(v),f.set(v,x)),g.isInstancedMesh&&(g.hasEventListener("dispose",m)===!1&&g.addEventListener("dispose",m),f.get(g)!==x&&(i.update(g.instanceMatrix,o.ARRAY_BUFFER),g.instanceColor!==null&&i.update(g.instanceColor,o.ARRAY_BUFFER),f.set(g,x))),g.isSkinnedMesh){const E=g.skeleton;f.get(E)!==x&&(E.update(),f.set(E,x))}return v}function h(){f=new WeakMap}function m(g){const x=g.target;x.removeEventListener("dispose",m),r.releaseStatesOfObject(x),i.remove(x.instanceMatrix),x.instanceColor!==null&&i.remove(x.instanceColor)}return{update:d,dispose:h}}const b1={[Mx]:"LINEAR_TONE_MAPPING",[Ex]:"REINHARD_TONE_MAPPING",[bx]:"CINEON_TONE_MAPPING",[Tx]:"ACES_FILMIC_TONE_MAPPING",[Rx]:"AGX_TONE_MAPPING",[wx]:"NEUTRAL_TONE_MAPPING",[Ax]:"CUSTOM_TONE_MAPPING"};function T1(o,t,i,r,c,f){const d=new Bi(t,i,{type:o,depthBuffer:c,stencilBuffer:f,samples:r?4:0,storeMultisampledDepthBuffer:!1,storeMultisampledStencilBuffer:!1,resolveDepthBuffer:!1,resolveStencilBuffer:!1});let h=null,m=null;const g=new ua;g.setAttribute("position",new Ti([-1,3,0,-1,-1,0,3,-1,0],3)),g.setAttribute("uv",new Ti([0,2,0,0,2,0],2));const x=new hb({uniforms:{tDiffuse:{value:null}},vertexShader:`
			precision highp float;

			uniform mat4 modelViewMatrix;
			uniform mat4 projectionMatrix;

			attribute vec3 position;
			attribute vec2 uv;

			varying vec2 vUv;

			void main() {
				vUv = uv;
				gl_Position = projectionMatrix * modelViewMatrix * vec4( position, 1.0 );
			}`,fragmentShader:`
			precision highp float;

			uniform sampler2D tDiffuse;

			varying vec2 vUv;

			#include <tonemapping_pars_fragment>
			#include <colorspace_pars_fragment>

			void main() {
				gl_FragColor = texture2D( tDiffuse, vUv );

				#ifdef LINEAR_TONE_MAPPING
					gl_FragColor.rgb = LinearToneMapping( gl_FragColor.rgb );
				#elif defined( REINHARD_TONE_MAPPING )
					gl_FragColor.rgb = ReinhardToneMapping( gl_FragColor.rgb );
				#elif defined( CINEON_TONE_MAPPING )
					gl_FragColor.rgb = CineonToneMapping( gl_FragColor.rgb );
				#elif defined( ACES_FILMIC_TONE_MAPPING )
					gl_FragColor.rgb = ACESFilmicToneMapping( gl_FragColor.rgb );
				#elif defined( AGX_TONE_MAPPING )
					gl_FragColor.rgb = AgXToneMapping( gl_FragColor.rgb );
				#elif defined( NEUTRAL_TONE_MAPPING )
					gl_FragColor.rgb = NeutralToneMapping( gl_FragColor.rgb );
				#elif defined( CUSTOM_TONE_MAPPING )
					gl_FragColor.rgb = CustomToneMapping( gl_FragColor.rgb );
				#endif

				#ifdef SRGB_TRANSFER
					gl_FragColor = sRGBTransferOETF( gl_FragColor );
				#endif
			}`,depthTest:!1,depthWrite:!1}),_=new la(g,x),v=new Zx(-1,1,1,-1,0,1);let E=null,R=null,N=!1,y,S=null,z=[],W=!1;this.setSize=function(C,D){d.setSize(C,D),h!==null&&h.setSize(C,D),m!==null&&m.setSize(C,D);for(let U=0;U<z.length;U++){const I=z[U];I.setSize&&I.setSize(C,D)}},this.setEffects=function(C){z=C,W=z.length>0&&z[0].isRenderPass===!0;const D=d.width,U=d.height;z.length>0&&h===null&&(h=new Bi(D,U,{type:oa,depthBuffer:!1,stencilBuffer:!1}),m=new Bi(D,U,{type:oa,depthBuffer:!1,stencilBuffer:!1}));for(let I=0;I<z.length;I++){const b=z[I];b.setSize&&b.setSize(D,U)}},this.begin=function(C,D){if(N||C.toneMapping===sa&&z.length===0)return!1;if(S=D,D!==null){const U=D.width,I=D.height;(d.width!==U||d.height!==I)&&this.setSize(U,I)}return W===!1&&C.setRenderTarget(d),y=C.toneMapping,C.toneMapping=sa,!0},this.hasRenderPass=function(){return W},this.end=function(C,D){C.toneMapping=y,N=!0;let U=d,I=h;for(let b=0;b<z.length;b++){const L=z[b];L.enabled!==!1&&(L.render(C,I,U,D),L.needsSwap!==!1&&(U=I,I=I===h?m:h))}if(E!==C.outputColorSpace||R!==C.toneMapping){E=C.outputColorSpace,R=C.toneMapping,x.defines={},Ct.getTransfer(E)===Yt&&(x.defines.SRGB_TRANSFER="");const b=b1[R];b&&(x.defines[b]=""),x.needsUpdate=!0}x.uniforms.tDiffuse.value=U.texture,C.setRenderTarget(S),C.render(_,v),S=null,N=!1},this.isCompositing=function(){return N},this.dispose=function(){d.dispose(),h!==null&&h.dispose(),m!==null&&m.dispose(),g.dispose(),x.dispose()}}const $x=new kn,Ip=new El(1,1),eS=new zx,tS=new kE,nS=new Xx,Kv=[],Qv=[],Jv=new Float32Array(16),$v=new Float32Array(9),ex=new Float32Array(4);function lo(o,t,i){const r=o[0];if(r<=0||r>0)return o;const c=t*i;let f=Kv[c];if(f===void 0&&(f=new Float32Array(c),Kv[c]=f),t!==0){r.toArray(f,0);for(let d=1,h=0;d!==t;++d)h+=i,o[d].toArray(f,h)}return f}function Sn(o,t){if(o.length!==t.length)return!1;for(let i=0,r=o.length;i<r;i++)if(o[i]!==t[i])return!1;return!0}function yn(o,t){for(let i=0,r=t.length;i<r;i++)o[i]=t[i]}function Ou(o,t){let i=Qv[t];i===void 0&&(i=new Int32Array(t),Qv[t]=i);for(let r=0;r!==t;++r)i[r]=o.allocateTextureUnit();return i}function A1(o,t){const i=this.cache;i[0]!==t&&(o.uniform1f(this.addr,t),i[0]=t)}function R1(o,t){const i=this.cache;if(t.x!==void 0)(i[0]!==t.x||i[1]!==t.y)&&(o.uniform2f(this.addr,t.x,t.y),i[0]=t.x,i[1]=t.y);else{if(Sn(i,t))return;o.uniform2fv(this.addr,t),yn(i,t)}}function w1(o,t){const i=this.cache;if(t.x!==void 0)(i[0]!==t.x||i[1]!==t.y||i[2]!==t.z)&&(o.uniform3f(this.addr,t.x,t.y,t.z),i[0]=t.x,i[1]=t.y,i[2]=t.z);else if(t.r!==void 0)(i[0]!==t.r||i[1]!==t.g||i[2]!==t.b)&&(o.uniform3f(this.addr,t.r,t.g,t.b),i[0]=t.r,i[1]=t.g,i[2]=t.b);else{if(Sn(i,t))return;o.uniform3fv(this.addr,t),yn(i,t)}}function C1(o,t){const i=this.cache;if(t.x!==void 0)(i[0]!==t.x||i[1]!==t.y||i[2]!==t.z||i[3]!==t.w)&&(o.uniform4f(this.addr,t.x,t.y,t.z,t.w),i[0]=t.x,i[1]=t.y,i[2]=t.z,i[3]=t.w);else{if(Sn(i,t))return;o.uniform4fv(this.addr,t),yn(i,t)}}function N1(o,t){const i=this.cache,r=t.elements;if(r===void 0){if(Sn(i,t))return;o.uniformMatrix2fv(this.addr,!1,t),yn(i,t)}else{if(Sn(i,r))return;ex.set(r),o.uniformMatrix2fv(this.addr,!1,ex),yn(i,r)}}function D1(o,t){const i=this.cache,r=t.elements;if(r===void 0){if(Sn(i,t))return;o.uniformMatrix3fv(this.addr,!1,t),yn(i,t)}else{if(Sn(i,r))return;$v.set(r),o.uniformMatrix3fv(this.addr,!1,$v),yn(i,r)}}function U1(o,t){const i=this.cache,r=t.elements;if(r===void 0){if(Sn(i,t))return;o.uniformMatrix4fv(this.addr,!1,t),yn(i,t)}else{if(Sn(i,r))return;Jv.set(r),o.uniformMatrix4fv(this.addr,!1,Jv),yn(i,r)}}function L1(o,t){const i=this.cache;i[0]!==t&&(o.uniform1i(this.addr,t),i[0]=t)}function O1(o,t){const i=this.cache;if(t.x!==void 0)(i[0]!==t.x||i[1]!==t.y)&&(o.uniform2i(this.addr,t.x,t.y),i[0]=t.x,i[1]=t.y);else{if(Sn(i,t))return;o.uniform2iv(this.addr,t),yn(i,t)}}function P1(o,t){const i=this.cache;if(t.x!==void 0)(i[0]!==t.x||i[1]!==t.y||i[2]!==t.z)&&(o.uniform3i(this.addr,t.x,t.y,t.z),i[0]=t.x,i[1]=t.y,i[2]=t.z);else{if(Sn(i,t))return;o.uniform3iv(this.addr,t),yn(i,t)}}function I1(o,t){const i=this.cache;if(t.x!==void 0)(i[0]!==t.x||i[1]!==t.y||i[2]!==t.z||i[3]!==t.w)&&(o.uniform4i(this.addr,t.x,t.y,t.z,t.w),i[0]=t.x,i[1]=t.y,i[2]=t.z,i[3]=t.w);else{if(Sn(i,t))return;o.uniform4iv(this.addr,t),yn(i,t)}}function B1(o,t){const i=this.cache;i[0]!==t&&(o.uniform1ui(this.addr,t),i[0]=t)}function z1(o,t){const i=this.cache;if(t.x!==void 0)(i[0]!==t.x||i[1]!==t.y)&&(o.uniform2ui(this.addr,t.x,t.y),i[0]=t.x,i[1]=t.y);else{if(Sn(i,t))return;o.uniform2uiv(this.addr,t),yn(i,t)}}function F1(o,t){const i=this.cache;if(t.x!==void 0)(i[0]!==t.x||i[1]!==t.y||i[2]!==t.z)&&(o.uniform3ui(this.addr,t.x,t.y,t.z),i[0]=t.x,i[1]=t.y,i[2]=t.z);else{if(Sn(i,t))return;o.uniform3uiv(this.addr,t),yn(i,t)}}function H1(o,t){const i=this.cache;if(t.x!==void 0)(i[0]!==t.x||i[1]!==t.y||i[2]!==t.z||i[3]!==t.w)&&(o.uniform4ui(this.addr,t.x,t.y,t.z,t.w),i[0]=t.x,i[1]=t.y,i[2]=t.z,i[3]=t.w);else{if(Sn(i,t))return;o.uniform4uiv(this.addr,t),yn(i,t)}}function G1(o,t,i){const r=this.cache,c=i.allocateTextureUnit();r[0]!==c&&(o.uniform1i(this.addr,c),r[0]=c);let f;this.type===o.SAMPLER_2D_SHADOW?(Ip.compareFunction=i.isReversedDepthBuffer()?jp:Yp,f=Ip):f=$x,i.setTexture2D(t||f,c)}function V1(o,t,i){const r=this.cache,c=i.allocateTextureUnit();r[0]!==c&&(o.uniform1i(this.addr,c),r[0]=c),i.setTexture3D(t||tS,c)}function k1(o,t,i){const r=this.cache,c=i.allocateTextureUnit();r[0]!==c&&(o.uniform1i(this.addr,c),r[0]=c),i.setTextureCube(t||nS,c)}function X1(o,t,i){const r=this.cache,c=i.allocateTextureUnit();r[0]!==c&&(o.uniform1i(this.addr,c),r[0]=c),i.setTexture2DArray(t||eS,c)}function W1(o){switch(o){case 5126:return A1;case 35664:return R1;case 35665:return w1;case 35666:return C1;case 35674:return N1;case 35675:return D1;case 35676:return U1;case 5124:case 35670:return L1;case 35667:case 35671:return O1;case 35668:case 35672:return P1;case 35669:case 35673:return I1;case 5125:return B1;case 36294:return z1;case 36295:return F1;case 36296:return H1;case 35678:case 36198:case 36298:case 36306:case 35682:return G1;case 35679:case 36299:case 36307:return V1;case 35680:case 36300:case 36308:case 36293:return k1;case 36289:case 36303:case 36311:case 36292:return X1}}function q1(o,t){o.uniform1fv(this.addr,t)}function Y1(o,t){const i=lo(t,this.size,2);o.uniform2fv(this.addr,i)}function j1(o,t){const i=lo(t,this.size,3);o.uniform3fv(this.addr,i)}function Z1(o,t){const i=lo(t,this.size,4);o.uniform4fv(this.addr,i)}function K1(o,t){const i=lo(t,this.size,4);o.uniformMatrix2fv(this.addr,!1,i)}function Q1(o,t){const i=lo(t,this.size,9);o.uniformMatrix3fv(this.addr,!1,i)}function J1(o,t){const i=lo(t,this.size,16);o.uniformMatrix4fv(this.addr,!1,i)}function $1(o,t){o.uniform1iv(this.addr,t)}function eR(o,t){o.uniform2iv(this.addr,t)}function tR(o,t){o.uniform3iv(this.addr,t)}function nR(o,t){o.uniform4iv(this.addr,t)}function iR(o,t){o.uniform1uiv(this.addr,t)}function aR(o,t){o.uniform2uiv(this.addr,t)}function sR(o,t){o.uniform3uiv(this.addr,t)}function rR(o,t){o.uniform4uiv(this.addr,t)}function oR(o,t,i){const r=this.cache,c=t.length,f=Ou(i,c);Sn(r,f)||(o.uniform1iv(this.addr,f),yn(r,f));let d;this.type===o.SAMPLER_2D_SHADOW?d=Ip:d=$x;for(let h=0;h!==c;++h)i.setTexture2D(t[h]||d,f[h])}function lR(o,t,i){const r=this.cache,c=t.length,f=Ou(i,c);Sn(r,f)||(o.uniform1iv(this.addr,f),yn(r,f));for(let d=0;d!==c;++d)i.setTexture3D(t[d]||tS,f[d])}function cR(o,t,i){const r=this.cache,c=t.length,f=Ou(i,c);Sn(r,f)||(o.uniform1iv(this.addr,f),yn(r,f));for(let d=0;d!==c;++d)i.setTextureCube(t[d]||nS,f[d])}function uR(o,t,i){const r=this.cache,c=t.length,f=Ou(i,c);Sn(r,f)||(o.uniform1iv(this.addr,f),yn(r,f));for(let d=0;d!==c;++d)i.setTexture2DArray(t[d]||eS,f[d])}function fR(o){switch(o){case 5126:return q1;case 35664:return Y1;case 35665:return j1;case 35666:return Z1;case 35674:return K1;case 35675:return Q1;case 35676:return J1;case 5124:case 35670:return $1;case 35667:case 35671:return eR;case 35668:case 35672:return tR;case 35669:case 35673:return nR;case 5125:return iR;case 36294:return aR;case 36295:return sR;case 36296:return rR;case 35678:case 36198:case 36298:case 36306:case 35682:return oR;case 35679:case 36299:case 36307:return lR;case 35680:case 36300:case 36308:case 36293:return cR;case 36289:case 36303:case 36311:case 36292:return uR}}class dR{constructor(t,i,r){this.id=t,this.addr=r,this.cache=[],this.type=i.type,this.setValue=W1(i.type)}}class hR{constructor(t,i,r){this.id=t,this.addr=r,this.cache=[],this.type=i.type,this.size=i.size,this.setValue=fR(i.type)}}class pR{constructor(t){this.id=t,this.seq=[],this.map={}}setValue(t,i,r){const c=this.seq;for(let f=0,d=c.length;f!==d;++f){const h=c[f];h.setValue(t,i[h.id],r)}}}const qh=/(\w+)(\])?(\[|\.)?/g;function tx(o,t){o.seq.push(t),o.map[t.id]=t}function mR(o,t,i){const r=o.name,c=r.length;for(qh.lastIndex=0;;){const f=qh.exec(r),d=qh.lastIndex;let h=f[1];const m=f[2]==="]",g=f[3];if(m&&(h=h|0),g===void 0||g==="["&&d+2===c){tx(i,g===void 0?new dR(h,o,t):new hR(h,o,t));break}else{let _=i.map[h];_===void 0&&(_=new pR(h),tx(i,_)),i=_}}}class Eu{constructor(t,i){this.seq=[],this.map={};const r=t.getProgramParameter(i,t.ACTIVE_UNIFORMS);for(let d=0;d<r;++d){const h=t.getActiveUniform(i,d),m=t.getUniformLocation(i,h.name);mR(h,m,this)}const c=[],f=[];for(const d of this.seq)d.type===t.SAMPLER_2D_SHADOW||d.type===t.SAMPLER_CUBE_SHADOW||d.type===t.SAMPLER_2D_ARRAY_SHADOW?c.push(d):f.push(d);c.length>0&&(this.seq=c.concat(f))}setValue(t,i,r,c){const f=this.map[i];f!==void 0&&f.setValue(t,r,c)}setOptional(t,i,r){const c=i[r];c!==void 0&&this.setValue(t,r,c)}static upload(t,i,r,c){for(let f=0,d=i.length;f!==d;++f){const h=i[f],m=r[h.id];m.needsUpdate!==!1&&h.setValue(t,m.value,c)}}static seqWithValue(t,i){const r=[];for(let c=0,f=t.length;c!==f;++c){const d=t[c];d.id in i&&r.push(d)}return r}}function nx(o,t,i){const r=o.createShader(t);return o.shaderSource(r,i),o.compileShader(r),r}const gR=37297;let _R=0;function vR(o,t){const i=o.split(`
`),r=[],c=Math.max(t-6,0),f=Math.min(t+6,i.length);for(let d=c;d<f;d++){const h=d+1;r.push(`${h===t?">":" "} ${h}: ${i[d]}`)}return r.join(`
`)}const ix=new ft;function xR(o){Ct._getMatrix(ix,Ct.workingColorSpace,o);const t=`mat3( ${ix.elements.map(i=>i.toFixed(4))} )`;switch(Ct.getTransfer(o)){case Ru:return[t,"LinearTransferOETF"];case Yt:return[t,"sRGBTransferOETF"];default:return ot("WebGLProgram: Unsupported color space: ",o),[t,"LinearTransferOETF"]}}function ax(o,t,i){const r=o.getShaderParameter(t,o.COMPILE_STATUS),f=(o.getShaderInfoLog(t)||"").trim();if(r&&f==="")return"";const d=/ERROR: 0:(\d+)/.exec(f);if(d){const h=parseInt(d[1]);return i.toUpperCase()+`

`+f+`

`+vR(o.getShaderSource(t),h)}else return f}function SR(o,t){const i=xR(t);return[`vec4 ${o}( vec4 value ) {`,`	return ${i[1]}( vec4( value.rgb * ${i[0]}, value.a ) );`,"}"].join(`
`)}const yR={[Mx]:"Linear",[Ex]:"Reinhard",[bx]:"Cineon",[Tx]:"ACESFilmic",[Rx]:"AgX",[wx]:"Neutral",[Ax]:"Custom"};function MR(o,t){const i=yR[t];return i===void 0?(ot("WebGLProgram: Unsupported toneMapping:",t),"vec3 "+o+"( vec3 color ) { return LinearToneMapping( color ); }"):"vec3 "+o+"( vec3 color ) { return "+i+"ToneMapping( color ); }"}const _u=new he;function ER(){Ct.getLuminanceCoefficients(_u);const o=_u.x.toFixed(4),t=_u.y.toFixed(4),i=_u.z.toFixed(4);return["float luminance( const in vec3 rgb ) {",`	const vec3 weights = vec3( ${o}, ${t}, ${i} );`,"	return dot( weights, rgb );","}"].join(`
`)}function bR(o){return[o.extensionClipCullDistance?"#extension GL_ANGLE_clip_cull_distance : require":"",o.extensionMultiDraw?"#extension GL_ANGLE_multi_draw : require":""].filter(gl).join(`
`)}function TR(o){const t=[];for(const i in o){const r=o[i];r!==!1&&t.push("#define "+i+" "+r)}return t.join(`
`)}function AR(o,t){const i={},r=o.getProgramParameter(t,o.ACTIVE_ATTRIBUTES);for(let c=0;c<r;c++){const f=o.getActiveAttrib(t,c),d=f.name;let h=1;f.type===o.FLOAT_MAT2&&(h=2),f.type===o.FLOAT_MAT3&&(h=3),f.type===o.FLOAT_MAT4&&(h=4),i[d]={type:f.type,location:o.getAttribLocation(t,d),locationSize:h}}return i}function gl(o){return o!==""}function sx(o,t){const i=t.numSpotLightShadows+t.numSpotLightMaps-t.numSpotLightShadowsWithMaps;return o.replace(/NUM_SUN_LIGHTS/g,t.numSunLights).replace(/NUM_DIR_LIGHTS/g,t.numDirLights).replace(/NUM_SPOT_LIGHTS/g,t.numSpotLights).replace(/NUM_SPOT_LIGHT_MAPS/g,t.numSpotLightMaps).replace(/NUM_SPOT_LIGHT_COORDS/g,i).replace(/NUM_RECT_AREA_LIGHTS/g,t.numRectAreaLights).replace(/NUM_POINT_LIGHTS/g,t.numPointLights).replace(/NUM_HEMI_LIGHTS/g,t.numHemiLights).replace(/NUM_SUN_LIGHT_SHADOWS/g,t.numSunLightShadows).replace(/NUM_DIR_LIGHT_SHADOWS/g,t.numDirLightShadows).replace(/NUM_SPOT_LIGHT_SHADOWS_WITH_MAPS/g,t.numSpotLightShadowsWithMaps).replace(/NUM_SPOT_LIGHT_SHADOWS/g,t.numSpotLightShadows).replace(/NUM_POINT_LIGHT_SHADOWS/g,t.numPointLightShadows)}function rx(o,t){return o.replace(/NUM_CLIPPING_PLANES/g,t.numClippingPlanes).replace(/UNION_CLIPPING_PLANES/g,t.numClippingPlanes-t.numClipIntersection)}const RR=/^[ \t]*#include +<([\w\d./]+)>/gm;function Bp(o){return o.replace(RR,CR)}const wR=new Map;function CR(o,t){let i=mt[t];if(i===void 0){const r=wR.get(t);if(r!==void 0)i=mt[r],ot('WebGLRenderer: Shader chunk "%s" has been deprecated. Use "%s" instead.',t,r);else throw new Error("THREE.WebGLProgram: Can not resolve #include <"+t+">")}return Bp(i)}const NR=/#pragma unroll_loop_start\s+for\s*\(\s*int\s+i\s*=\s*(\d+)\s*;\s*i\s*<\s*(\d+)\s*;\s*i\s*\+\+\s*\)\s*{([\s\S]+?)}\s+#pragma unroll_loop_end/g;function ox(o){return o.replace(NR,DR)}function DR(o,t,i,r){let c="";for(let f=parseInt(t);f<parseInt(i);f++)c+=r.replace(/\[\s*i\s*\]/g,"[ "+f+" ]").replace(/UNROLLED_LOOP_INDEX/g,f);return c}function lx(o){let t=`precision ${o.precision} float;
	precision ${o.precision} int;
	precision ${o.precision} sampler2D;
	precision ${o.precision} samplerCube;
	precision ${o.precision} sampler3D;
	precision ${o.precision} sampler2DArray;
	precision ${o.precision} sampler2DShadow;
	precision ${o.precision} samplerCubeShadow;
	precision ${o.precision} sampler2DArrayShadow;
	precision ${o.precision} isampler2D;
	precision ${o.precision} isampler3D;
	precision ${o.precision} isamplerCube;
	precision ${o.precision} isampler2DArray;
	precision ${o.precision} usampler2D;
	precision ${o.precision} usampler3D;
	precision ${o.precision} usamplerCube;
	precision ${o.precision} usampler2DArray;
	`;return o.precision==="highp"?t+=`
#define HIGH_PRECISION`:o.precision==="mediump"?t+=`
#define MEDIUM_PRECISION`:o.precision==="lowp"&&(t+=`
#define LOW_PRECISION`),t}const UR={[vu]:"SHADOWMAP_TYPE_PCF",[ml]:"SHADOWMAP_TYPE_VSM"};function LR(o){return UR[o.shadowMapType]||"SHADOWMAP_TYPE_BASIC"}const OR={[Qs]:"ENVMAP_TYPE_CUBE",[ao]:"ENVMAP_TYPE_CUBE",[Du]:"ENVMAP_TYPE_CUBE_UV"};function PR(o){return o.envMap===!1?"ENVMAP_TYPE_CUBE":OR[o.envMapMode]||"ENVMAP_TYPE_CUBE"}const IR={[ao]:"ENVMAP_MODE_REFRACTION"};function BR(o){return o.envMap===!1?"ENVMAP_MODE_REFLECTION":IR[o.envMapMode]||"ENVMAP_MODE_REFLECTION"}const zR={[yx]:"ENVMAP_BLENDING_MULTIPLY",[SE]:"ENVMAP_BLENDING_MIX",[yE]:"ENVMAP_BLENDING_ADD"};function FR(o){return o.envMap===!1?"ENVMAP_BLENDING_NONE":zR[o.combine]||"ENVMAP_BLENDING_NONE"}function HR(o){const t=o.envMapCubeUVHeight;if(t===null)return null;const i=Math.log2(t)-2,r=1/t;return{texelWidth:1/(3*Math.max(Math.pow(2,i),112)),texelHeight:r,maxMip:i}}function GR(o,t,i,r){const c=o.getContext(),f=i.defines;let d=i.vertexShader,h=i.fragmentShader;const m=LR(i),g=PR(i),x=BR(i),_=FR(i),v=HR(i),E=bR(i),R=TR(f),N=c.createProgram();let y,S,z=i.glslVersion?"#version "+i.glslVersion+`
`:"";i.isRawShaderMaterial?(y=["#define SHADER_TYPE "+i.shaderType,"#define SHADER_NAME "+i.shaderName,R].filter(gl).join(`
`),y.length>0&&(y+=`
`),S=["#define SHADER_TYPE "+i.shaderType,"#define SHADER_NAME "+i.shaderName,R].filter(gl).join(`
`),S.length>0&&(S+=`
`)):(y=[lx(i),"#define SHADER_TYPE "+i.shaderType,"#define SHADER_NAME "+i.shaderName,R,i.extensionClipCullDistance?"#define USE_CLIP_DISTANCE":"",i.batching?"#define USE_BATCHING":"",i.batchingColor?"#define USE_BATCHING_COLOR":"",i.instancing?"#define USE_INSTANCING":"",i.instancingColor?"#define USE_INSTANCING_COLOR":"",i.instancingMorph?"#define USE_INSTANCING_MORPH":"",i.useFog&&i.fog?"#define USE_FOG":"",i.useFog&&i.fogExp2?"#define FOG_EXP2":"",i.map?"#define USE_MAP":"",i.envMap?"#define USE_ENVMAP":"",i.envMap?"#define "+x:"",i.lightMap?"#define USE_LIGHTMAP":"",i.aoMap?"#define USE_AOMAP":"",i.bumpMap?"#define USE_BUMPMAP":"",i.normalMap?"#define USE_NORMALMAP":"",i.normalMapObjectSpace?"#define USE_NORMALMAP_OBJECTSPACE":"",i.normalMapTangentSpace?"#define USE_NORMALMAP_TANGENTSPACE":"",i.displacementMap?"#define USE_DISPLACEMENTMAP":"",i.emissiveMap?"#define USE_EMISSIVEMAP":"",i.anisotropy?"#define USE_ANISOTROPY":"",i.anisotropyMap?"#define USE_ANISOTROPYMAP":"",i.clearcoatMap?"#define USE_CLEARCOATMAP":"",i.clearcoatRoughnessMap?"#define USE_CLEARCOAT_ROUGHNESSMAP":"",i.clearcoatNormalMap?"#define USE_CLEARCOAT_NORMALMAP":"",i.iridescenceMap?"#define USE_IRIDESCENCEMAP":"",i.iridescenceThicknessMap?"#define USE_IRIDESCENCE_THICKNESSMAP":"",i.specularMap?"#define USE_SPECULARMAP":"",i.specularColorMap?"#define USE_SPECULAR_COLORMAP":"",i.specularIntensityMap?"#define USE_SPECULAR_INTENSITYMAP":"",i.roughnessMap?"#define USE_ROUGHNESSMAP":"",i.metalnessMap?"#define USE_METALNESSMAP":"",i.alphaMap?"#define USE_ALPHAMAP":"",i.alphaHash?"#define USE_ALPHAHASH":"",i.transmission?"#define USE_TRANSMISSION":"",i.transmissionMap?"#define USE_TRANSMISSIONMAP":"",i.thicknessMap?"#define USE_THICKNESSMAP":"",i.sheenColorMap?"#define USE_SHEEN_COLORMAP":"",i.sheenRoughnessMap?"#define USE_SHEEN_ROUGHNESSMAP":"",i.mapUv?"#define MAP_UV "+i.mapUv:"",i.alphaMapUv?"#define ALPHAMAP_UV "+i.alphaMapUv:"",i.lightMapUv?"#define LIGHTMAP_UV "+i.lightMapUv:"",i.aoMapUv?"#define AOMAP_UV "+i.aoMapUv:"",i.emissiveMapUv?"#define EMISSIVEMAP_UV "+i.emissiveMapUv:"",i.bumpMapUv?"#define BUMPMAP_UV "+i.bumpMapUv:"",i.normalMapUv?"#define NORMALMAP_UV "+i.normalMapUv:"",i.displacementMapUv?"#define DISPLACEMENTMAP_UV "+i.displacementMapUv:"",i.metalnessMapUv?"#define METALNESSMAP_UV "+i.metalnessMapUv:"",i.roughnessMapUv?"#define ROUGHNESSMAP_UV "+i.roughnessMapUv:"",i.anisotropyMapUv?"#define ANISOTROPYMAP_UV "+i.anisotropyMapUv:"",i.clearcoatMapUv?"#define CLEARCOATMAP_UV "+i.clearcoatMapUv:"",i.clearcoatNormalMapUv?"#define CLEARCOAT_NORMALMAP_UV "+i.clearcoatNormalMapUv:"",i.clearcoatRoughnessMapUv?"#define CLEARCOAT_ROUGHNESSMAP_UV "+i.clearcoatRoughnessMapUv:"",i.iridescenceMapUv?"#define IRIDESCENCEMAP_UV "+i.iridescenceMapUv:"",i.iridescenceThicknessMapUv?"#define IRIDESCENCE_THICKNESSMAP_UV "+i.iridescenceThicknessMapUv:"",i.sheenColorMapUv?"#define SHEEN_COLORMAP_UV "+i.sheenColorMapUv:"",i.sheenRoughnessMapUv?"#define SHEEN_ROUGHNESSMAP_UV "+i.sheenRoughnessMapUv:"",i.specularMapUv?"#define SPECULARMAP_UV "+i.specularMapUv:"",i.specularColorMapUv?"#define SPECULAR_COLORMAP_UV "+i.specularColorMapUv:"",i.specularIntensityMapUv?"#define SPECULAR_INTENSITYMAP_UV "+i.specularIntensityMapUv:"",i.transmissionMapUv?"#define TRANSMISSIONMAP_UV "+i.transmissionMapUv:"",i.thicknessMapUv?"#define THICKNESSMAP_UV "+i.thicknessMapUv:"",i.vertexTangents&&i.flatShading===!1?"#define USE_TANGENT":"",i.vertexNormals?"#define HAS_NORMAL":"",i.vertexColors?"#define USE_COLOR":"",i.vertexAlphas?"#define USE_COLOR_ALPHA":"",i.vertexUv1s?"#define USE_UV1":"",i.vertexUv2s?"#define USE_UV2":"",i.vertexUv3s?"#define USE_UV3":"",i.pointsUvs?"#define USE_POINTS_UV":"",i.flatShading?"#define FLAT_SHADED":"",i.skinning?"#define USE_SKINNING":"",i.morphTargets?"#define USE_MORPHTARGETS":"",i.morphNormals&&i.flatShading===!1?"#define USE_MORPHNORMALS":"",i.morphColors?"#define USE_MORPHCOLORS":"",i.morphTargetsCount>0?"#define MORPHTARGETS_TEXTURE_STRIDE "+i.morphTextureStride:"",i.morphTargetsCount>0?"#define MORPHTARGETS_COUNT "+i.morphTargetsCount:"",i.doubleSided?"#define DOUBLE_SIDED":"",i.flipSided?"#define FLIP_SIDED":"",i.shadowMapEnabled?"#define USE_SHADOWMAP":"",i.shadowMapEnabled?"#define "+m:"",i.sizeAttenuation?"#define USE_SIZEATTENUATION":"",i.numLightProbes>0?"#define USE_LIGHT_PROBES":"",i.logarithmicDepthBuffer?"#define USE_LOGARITHMIC_DEPTH_BUFFER":"",i.reversedDepthBuffer?"#define USE_REVERSED_DEPTH_BUFFER":"","uniform mat4 modelMatrix;","uniform mat4 modelViewMatrix;","uniform mat4 projectionMatrix;","uniform mat4 viewMatrix;","uniform mat3 normalMatrix;","uniform vec3 cameraPosition;","uniform bool isOrthographic;","#ifdef USE_INSTANCING","	attribute mat4 instanceMatrix;","#endif","#ifdef USE_INSTANCING_COLOR","	attribute vec3 instanceColor;","#endif","#ifdef USE_INSTANCING_MORPH","	uniform sampler2D morphTexture;","#endif","attribute vec3 position;","attribute vec3 normal;","attribute vec2 uv;","#ifdef USE_UV1","	attribute vec2 uv1;","#endif","#ifdef USE_UV2","	attribute vec2 uv2;","#endif","#ifdef USE_UV3","	attribute vec2 uv3;","#endif","#ifdef USE_TANGENT","	attribute vec4 tangent;","#endif","#if defined( USE_COLOR_ALPHA )","	attribute vec4 color;","#elif defined( USE_COLOR )","	attribute vec3 color;","#endif","#ifdef USE_SKINNING","	attribute vec4 skinIndex;","	attribute vec4 skinWeight;","#endif",`
`].filter(gl).join(`
`),S=[lx(i),"#define SHADER_TYPE "+i.shaderType,"#define SHADER_NAME "+i.shaderName,R,i.useFog&&i.fog?"#define USE_FOG":"",i.useFog&&i.fogExp2?"#define FOG_EXP2":"",i.alphaToCoverage?"#define ALPHA_TO_COVERAGE":"",i.map?"#define USE_MAP":"",i.matcap?"#define USE_MATCAP":"",i.envMap?"#define USE_ENVMAP":"",i.envMap?"#define "+g:"",i.envMap?"#define "+x:"",i.envMap?"#define "+_:"",v?"#define CUBEUV_TEXEL_WIDTH "+v.texelWidth:"",v?"#define CUBEUV_TEXEL_HEIGHT "+v.texelHeight:"",v?"#define CUBEUV_MAX_MIP "+v.maxMip+".0":"",i.lightMap?"#define USE_LIGHTMAP":"",i.aoMap?"#define USE_AOMAP":"",i.bumpMap?"#define USE_BUMPMAP":"",i.normalMap?"#define USE_NORMALMAP":"",i.normalMapObjectSpace?"#define USE_NORMALMAP_OBJECTSPACE":"",i.normalMapTangentSpace?"#define USE_NORMALMAP_TANGENTSPACE":"",i.packedNormalMap?"#define USE_PACKED_NORMALMAP":"",i.emissiveMap?"#define USE_EMISSIVEMAP":"",i.anisotropy?"#define USE_ANISOTROPY":"",i.anisotropyMap?"#define USE_ANISOTROPYMAP":"",i.clearcoat?"#define USE_CLEARCOAT":"",i.clearcoatMap?"#define USE_CLEARCOATMAP":"",i.clearcoatRoughnessMap?"#define USE_CLEARCOAT_ROUGHNESSMAP":"",i.clearcoatNormalMap?"#define USE_CLEARCOAT_NORMALMAP":"",i.dispersion?"#define USE_DISPERSION":"",i.retroreflection?"#define USE_RETROREFLECTION":"",i.iridescence?"#define USE_IRIDESCENCE":"",i.iridescenceMap?"#define USE_IRIDESCENCEMAP":"",i.iridescenceThicknessMap?"#define USE_IRIDESCENCE_THICKNESSMAP":"",i.specularMap?"#define USE_SPECULARMAP":"",i.specularColorMap?"#define USE_SPECULAR_COLORMAP":"",i.specularIntensityMap?"#define USE_SPECULAR_INTENSITYMAP":"",i.roughnessMap?"#define USE_ROUGHNESSMAP":"",i.metalnessMap?"#define USE_METALNESSMAP":"",i.alphaMap?"#define USE_ALPHAMAP":"",i.alphaTest?"#define USE_ALPHATEST":"",i.alphaHash?"#define USE_ALPHAHASH":"",i.sheen?"#define USE_SHEEN":"",i.sheenColorMap?"#define USE_SHEEN_COLORMAP":"",i.sheenRoughnessMap?"#define USE_SHEEN_ROUGHNESSMAP":"",i.transmission?"#define USE_TRANSMISSION":"",i.transmissionMap?"#define USE_TRANSMISSIONMAP":"",i.thicknessMap?"#define USE_THICKNESSMAP":"",i.vertexTangents&&i.flatShading===!1?"#define USE_TANGENT":"",i.vertexColors||i.instancingColor?"#define USE_COLOR":"",i.vertexAlphas||i.batchingColor?"#define USE_COLOR_ALPHA":"",i.vertexUv1s?"#define USE_UV1":"",i.vertexUv2s?"#define USE_UV2":"",i.vertexUv3s?"#define USE_UV3":"",i.pointsUvs?"#define USE_POINTS_UV":"",i.gradientMap?"#define USE_GRADIENTMAP":"",i.flatShading?"#define FLAT_SHADED":"",i.doubleSided?"#define DOUBLE_SIDED":"",i.flipSided?"#define FLIP_SIDED":"",i.shadowMapEnabled?"#define USE_SHADOWMAP":"",i.shadowMapEnabled?"#define "+m:"",i.premultipliedAlpha?"#define PREMULTIPLIED_ALPHA":"",i.numLightProbes>0?"#define USE_LIGHT_PROBES":"",i.numLightProbeGrids>0?"#define USE_LIGHT_PROBES_GRID":"",i.decodeVideoTexture?"#define DECODE_VIDEO_TEXTURE":"",i.decodeVideoTextureEmissive?"#define DECODE_VIDEO_TEXTURE_EMISSIVE":"",i.logarithmicDepthBuffer?"#define USE_LOGARITHMIC_DEPTH_BUFFER":"",i.reversedDepthBuffer?"#define USE_REVERSED_DEPTH_BUFFER":"","uniform mat4 viewMatrix;","uniform vec3 cameraPosition;","uniform bool isOrthographic;",i.toneMapping!==sa?"#define TONE_MAPPING":"",i.toneMapping!==sa?mt.tonemapping_pars_fragment:"",i.toneMapping!==sa?MR("toneMapping",i.toneMapping):"",i.dithering?"#define DITHERING":"",i.opaque?"#define OPAQUE":"",mt.colorspace_pars_fragment,SR("linearToOutputTexel",i.outputColorSpace),ER(),i.useDepthPacking?"#define DEPTH_PACKING "+i.depthPacking:"",`
`].filter(gl).join(`
`)),d=Bp(d),d=sx(d,i),d=rx(d,i),h=Bp(h),h=sx(h,i),h=rx(h,i),d=ox(d),h=ox(h),i.isRawShaderMaterial!==!0&&(z=`#version 300 es
`,y=[E,"#define attribute in","#define varying out","#define texture2D texture"].join(`
`)+`
`+y,S=["#define varying in",i.glslVersion===yv?"":"layout(location = 0) out highp vec4 pc_fragColor;",i.glslVersion===yv?"":"#define gl_FragColor pc_fragColor","#define gl_FragDepthEXT gl_FragDepth","#define texture2D texture","#define textureCube texture","#define texture2DProj textureProj","#define texture2DLodEXT textureLod","#define texture2DProjLodEXT textureProjLod","#define textureCubeLodEXT textureLod","#define texture2DGradEXT textureGrad","#define texture2DProjGradEXT textureProjGrad","#define textureCubeGradEXT textureGrad"].join(`
`)+`
`+S);const W=z+y+d,C=z+S+h,D=nx(c,c.VERTEX_SHADER,W),U=nx(c,c.FRAGMENT_SHADER,C);c.attachShader(N,D),c.attachShader(N,U),i.index0AttributeName!==void 0?c.bindAttribLocation(N,0,i.index0AttributeName):i.hasPositionAttribute===!0&&c.bindAttribLocation(N,0,"position"),c.linkProgram(N);function I(Y){if(o.debug.checkShaderErrors){const $=c.getProgramInfoLog(N)||"",O=c.getShaderInfoLog(D)||"",q=c.getShaderInfoLog(U)||"",ne=$.trim(),j=O.trim(),Q=q.trim();let pe=!0,ce=!0;if(c.getProgramParameter(N,c.LINK_STATUS)===!1)if(pe=!1,typeof o.debug.onShaderError=="function")o.debug.onShaderError(c,N,D,U);else{const me=ax(c,D,"vertex"),be=ax(c,U,"fragment");It("WebGLProgram: Shader Error "+c.getError()+" - VALIDATE_STATUS "+c.getProgramParameter(N,c.VALIDATE_STATUS)+`

Material Name: `+Y.name+`
Material Type: `+Y.type+`

Program Info Log: `+ne+`
`+me+`
`+be)}else ne!==""?ot("WebGLProgram: Program Info Log:",ne):(j===""||Q==="")&&(ce=!1);ce&&(Y.diagnostics={runnable:pe,programLog:ne,vertexShader:{log:j,prefix:y},fragmentShader:{log:Q,prefix:S}})}c.deleteShader(D),c.deleteShader(U),b=new Eu(c,N),L=AR(c,N)}let b;this.getUniforms=function(){return b===void 0&&I(this),b};let L;this.getAttributes=function(){return L===void 0&&I(this),L};let k=i.rendererExtensionParallelShaderCompile===!1;return this.isReady=function(){return k===!1&&(k=c.getProgramParameter(N,gR)),k},this.destroy=function(){r.releaseStatesOfProgram(this),c.deleteProgram(N),this.program=void 0},this.type=i.shaderType,this.name=i.shaderName,this.id=_R++,this.cacheKey=t,this.usedTimes=1,this.program=N,this.vertexShader=D,this.fragmentShader=U,this}let VR=0;class kR{constructor(){this.shaderCache=new Map,this.materialCache=new Map}update(t,i,r){const c=this._getShaderCacheForMaterial(t);return c.has(i)===!1&&(c.add(i),i.usedTimes++),c.has(r)===!1&&(c.add(r),r.usedTimes++),this}remove(t){const i=this.materialCache.get(t);for(const r of i)r.usedTimes--,r.usedTimes===0&&this.shaderCache.delete(r.code);return this.materialCache.delete(t),this}getVertexShaderStage(t){return this._getShaderStage(t.vertexShader)}getFragmentShaderStage(t){return this._getShaderStage(t.fragmentShader)}dispose(){this.shaderCache.clear(),this.materialCache.clear()}_getShaderCacheForMaterial(t){const i=this.materialCache;let r=i.get(t);return r===void 0&&(r=new Set,i.set(t,r)),r}_getShaderStage(t){const i=this.shaderCache;let r=i.get(t);return r===void 0&&(r=new XR(t),i.set(t,r)),r}}class XR{constructor(t){this.id=VR++,this.code=t,this.usedTimes=0}}function WR(o){return o===Js||o===bu||o===Tu}function qR(o,t,i,r,c,f){const d=new Fx,h=new kR,m=new Set,g=[],x=new Map,_=r.logarithmicDepthBuffer;let v=r.precision;const E={MeshDepthMaterial:"depth",MeshDistanceMaterial:"distance",MeshNormalMaterial:"normal",MeshBasicMaterial:"basic",MeshLambertMaterial:"lambert",MeshPhongMaterial:"phong",MeshToonMaterial:"toon",MeshStandardMaterial:"physical",MeshPhysicalMaterial:"physical",MeshMatcapMaterial:"matcap",LineBasicMaterial:"basic",LineDashedMaterial:"dashed",PointsMaterial:"points",ShadowMaterial:"shadow",SpriteMaterial:"sprite"};function R(b){return m.add(b),b===0?"uv":`uv${b}`}function N(b,L,k,Y,$,O){const q=Y.fog,ne=$.geometry,j=b.isMeshStandardMaterial||b.isMeshLambertMaterial||b.isMeshPhongMaterial?Y.environment:null,Q=b.isMeshStandardMaterial||b.isMeshLambertMaterial&&!b.envMap||b.isMeshPhongMaterial&&!b.envMap,pe=t.get(b.envMap||j,Q),ce=pe&&pe.mapping===Du?pe.image.height:null,me=E[b.type];b.precision!==null&&(v=r.getMaxPrecision(b.precision),v!==b.precision&&ot("WebGLProgram.getParameters:",b.precision,"not supported, using",v,"instead."));const be=ne.morphAttributes.position||ne.morphAttributes.normal||ne.morphAttributes.color,Ze=be!==void 0?be.length:0;let je=0;ne.morphAttributes.position!==void 0&&(je=1),ne.morphAttributes.normal!==void 0&&(je=2),ne.morphAttributes.color!==void 0&&(je=3);let H,F,ge,X;if(me){const Rt=ta[me];H=Rt.vertexShader,F=Rt.fragmentShader}else{H=b.vertexShader,F=b.fragmentShader;const Rt=h.getVertexShaderStage(b),ct=h.getFragmentShaderStage(b);h.update(b,Rt,ct),ge=Rt.id,X=ct.id}const ue=o.getRenderTarget(),Ae=o.state.buffers.depth.getReversed(),Ie=$.isInstancedMesh===!0,xe=$.isBatchedMesh===!0,we=!!b.map,Vt=!!b.matcap,pt=!!pe,gt=!!b.aoMap,St=!!b.lightMap,tt=!!b.bumpMap&&b.wireframe===!1,it=!!b.normalMap,kt=!!b.displacementMap,dn=!!b.emissiveMap,Pt=!!b.metalnessMap,en=!!b.roughnessMap,K=b.anisotropy>0,nn=b.clearcoat>0,Ot=b.dispersion>0,P=b.retroreflectivity>0,M=b.iridescence>0,ie=b.sheen>0,le=b.transmission>0,_e=K&&!!b.anisotropyMap,Re=nn&&!!b.clearcoatMap,De=nn&&!!b.clearcoatNormalMap,ve=nn&&!!b.clearcoatRoughnessMap,Ee=M&&!!b.iridescenceMap,Ne=M&&!!b.iridescenceThicknessMap,$e=ie&&!!b.sheenColorMap,Be=ie&&!!b.sheenRoughnessMap,Pe=!!b.specularMap,ke=!!b.specularColorMap,nt=!!b.specularIntensityMap,lt=le&&!!b.transmissionMap,Z=le&&!!b.thicknessMap,Ce=!!b.gradientMap,Me=!!b.alphaMap,Ue=b.alphaTest>0,Ve=!!b.alphaHash,Te=!!b.extensions;let Je=sa;b.toneMapped&&(ue===null||ue.isXRRenderTarget===!0)&&(Je=o.toneMapping);const Ge={shaderID:me,shaderType:b.type,shaderName:b.name,vertexShader:H,fragmentShader:F,defines:b.defines,customVertexShaderID:ge,customFragmentShaderID:X,isRawShaderMaterial:b.isRawShaderMaterial===!0,glslVersion:b.glslVersion,precision:v,batching:xe,batchingColor:xe&&$._colorsTexture!==null,instancing:Ie,instancingColor:Ie&&$.instanceColor!==null,instancingMorph:Ie&&$.morphTexture!==null,outputColorSpace:ue===null?o.outputColorSpace:ue.isXRRenderTarget===!0?ue.texture.colorSpace:Ct.workingColorSpace,alphaToCoverage:!!b.alphaToCoverage,map:we,matcap:Vt,envMap:pt,envMapMode:pt&&pe.mapping,envMapCubeUVHeight:ce,aoMap:gt,lightMap:St,bumpMap:tt,normalMap:it,displacementMap:kt,emissiveMap:dn,normalMapObjectSpace:it&&b.normalMapType===bE,normalMapTangentSpace:it&&b.normalMapType===Op,packedNormalMap:it&&b.normalMapType===Op&&WR(b.normalMap.format),metalnessMap:Pt,roughnessMap:en,anisotropy:K,anisotropyMap:_e,clearcoat:nn,clearcoatMap:Re,clearcoatNormalMap:De,clearcoatRoughnessMap:ve,dispersion:Ot,retroreflection:P,iridescence:M,iridescenceMap:Ee,iridescenceThicknessMap:Ne,sheen:ie,sheenColorMap:$e,sheenRoughnessMap:Be,specularMap:Pe,specularColorMap:ke,specularIntensityMap:nt,transmission:le,transmissionMap:lt,thicknessMap:Z,gradientMap:Ce,opaque:b.transparent===!1&&b.blending===vl&&b.alphaToCoverage===!1,alphaMap:Me,alphaTest:Ue,alphaHash:Ve,combine:b.combine,mapUv:we&&R(b.map.channel),aoMapUv:gt&&R(b.aoMap.channel),lightMapUv:St&&R(b.lightMap.channel),bumpMapUv:tt&&R(b.bumpMap.channel),normalMapUv:it&&R(b.normalMap.channel),displacementMapUv:kt&&R(b.displacementMap.channel),emissiveMapUv:dn&&R(b.emissiveMap.channel),metalnessMapUv:Pt&&R(b.metalnessMap.channel),roughnessMapUv:en&&R(b.roughnessMap.channel),anisotropyMapUv:_e&&R(b.anisotropyMap.channel),clearcoatMapUv:Re&&R(b.clearcoatMap.channel),clearcoatNormalMapUv:De&&R(b.clearcoatNormalMap.channel),clearcoatRoughnessMapUv:ve&&R(b.clearcoatRoughnessMap.channel),iridescenceMapUv:Ee&&R(b.iridescenceMap.channel),iridescenceThicknessMapUv:Ne&&R(b.iridescenceThicknessMap.channel),sheenColorMapUv:$e&&R(b.sheenColorMap.channel),sheenRoughnessMapUv:Be&&R(b.sheenRoughnessMap.channel),specularMapUv:Pe&&R(b.specularMap.channel),specularColorMapUv:ke&&R(b.specularColorMap.channel),specularIntensityMapUv:nt&&R(b.specularIntensityMap.channel),transmissionMapUv:lt&&R(b.transmissionMap.channel),thicknessMapUv:Z&&R(b.thicknessMap.channel),alphaMapUv:Me&&R(b.alphaMap.channel),vertexTangents:!!ne.attributes.tangent&&(it||K),vertexNormals:!!ne.attributes.normal,vertexColors:b.vertexColors,vertexAlphas:b.vertexColors===!0&&!!ne.attributes.color&&ne.attributes.color.itemSize===4,pointsUvs:$.isPoints===!0&&!!ne.attributes.uv&&(we||Me),fog:!!q,useFog:b.fog===!0,fogExp2:!!q&&q.isFogExp2,flatShading:b.wireframe===!1&&(b.flatShading===!0||ne.attributes.normal===void 0&&it===!1&&(b.isMeshLambertMaterial||b.isMeshPhongMaterial||b.isMeshStandardMaterial||b.isMeshPhysicalMaterial)),sizeAttenuation:b.sizeAttenuation===!0,logarithmicDepthBuffer:_,reversedDepthBuffer:Ae,skinning:$.isSkinnedMesh===!0,hasPositionAttribute:ne.attributes.position!==void 0,morphTargets:ne.morphAttributes.position!==void 0,morphNormals:ne.morphAttributes.normal!==void 0,morphColors:ne.morphAttributes.color!==void 0,morphTargetsCount:Ze,morphTextureStride:je,numSunLights:L.sun.length,numDirLights:L.directional.length,numPointLights:L.point.length,numSpotLights:L.spot.length,numSpotLightMaps:L.spotLightMap.length,numRectAreaLights:L.rectArea.length,numHemiLights:L.hemi.length,numSunLightShadows:L.sunShadowMap.length,numDirLightShadows:L.directionalShadowMap.length,numPointLightShadows:L.pointShadowMap.length,numSpotLightShadows:L.spotShadowMap.length,numSpotLightShadowsWithMaps:L.numSpotLightShadowsWithMaps,numLightProbes:L.numLightProbes,numLightProbeGrids:O.length,numClippingPlanes:f.numPlanes,numClipIntersection:f.numIntersection,dithering:b.dithering,shadowMapEnabled:o.shadowMap.enabled&&k.length>0,shadowMapType:o.shadowMap.type,toneMapping:Je,decodeVideoTexture:we&&b.map.isVideoTexture===!0&&Ct.getTransfer(b.map.colorSpace)===Yt,decodeVideoTextureEmissive:dn&&b.emissiveMap.isVideoTexture===!0&&Ct.getTransfer(b.emissiveMap.colorSpace)===Yt,premultipliedAlpha:b.premultipliedAlpha,doubleSided:b.side===na,flipSided:b.side===Kn,useDepthPacking:b.depthPacking>=0,depthPacking:b.depthPacking||0,index0AttributeName:b.index0AttributeName,extensionClipCullDistance:Te&&b.extensions.clipCullDistance===!0&&i.has("WEBGL_clip_cull_distance"),extensionMultiDraw:(Te&&b.extensions.multiDraw===!0||xe)&&i.has("WEBGL_multi_draw"),rendererExtensionParallelShaderCompile:i.has("KHR_parallel_shader_compile"),customProgramCacheKey:b.customProgramCacheKey()};return Ge.vertexUv1s=m.has(1),Ge.vertexUv2s=m.has(2),Ge.vertexUv3s=m.has(3),m.clear(),Ge}function y(b){const L=[];if(b.shaderID?L.push(b.shaderID):(L.push(b.customVertexShaderID),L.push(b.customFragmentShaderID)),b.defines!==void 0)for(const k in b.defines)L.push(k),L.push(b.defines[k]);return b.isRawShaderMaterial===!1&&(S(L,b),z(L,b),L.push(o.outputColorSpace)),L.push(b.customProgramCacheKey),L.join()}function S(b,L){b.push(L.precision),b.push(L.outputColorSpace),b.push(L.envMapMode),b.push(L.envMapCubeUVHeight),b.push(L.mapUv),b.push(L.alphaMapUv),b.push(L.lightMapUv),b.push(L.aoMapUv),b.push(L.bumpMapUv),b.push(L.normalMapUv),b.push(L.displacementMapUv),b.push(L.emissiveMapUv),b.push(L.metalnessMapUv),b.push(L.roughnessMapUv),b.push(L.anisotropyMapUv),b.push(L.clearcoatMapUv),b.push(L.clearcoatNormalMapUv),b.push(L.clearcoatRoughnessMapUv),b.push(L.iridescenceMapUv),b.push(L.iridescenceThicknessMapUv),b.push(L.sheenColorMapUv),b.push(L.sheenRoughnessMapUv),b.push(L.specularMapUv),b.push(L.specularColorMapUv),b.push(L.specularIntensityMapUv),b.push(L.transmissionMapUv),b.push(L.thicknessMapUv),b.push(L.combine),b.push(L.fogExp2),b.push(L.sizeAttenuation),b.push(L.morphTargetsCount),b.push(L.morphAttributeCount),b.push(L.numSunLights),b.push(L.numDirLights),b.push(L.numPointLights),b.push(L.numSpotLights),b.push(L.numSpotLightMaps),b.push(L.numHemiLights),b.push(L.numRectAreaLights),b.push(L.numSunLightShadows),b.push(L.numDirLightShadows),b.push(L.numPointLightShadows),b.push(L.numSpotLightShadows),b.push(L.numSpotLightShadowsWithMaps),b.push(L.numLightProbes),b.push(L.shadowMapType),b.push(L.toneMapping),b.push(L.numClippingPlanes),b.push(L.numClipIntersection),b.push(L.depthPacking)}function z(b,L){d.disableAll(),L.instancing&&d.enable(0),L.instancingColor&&d.enable(1),L.instancingMorph&&d.enable(2),L.matcap&&d.enable(3),L.envMap&&d.enable(4),L.normalMapObjectSpace&&d.enable(5),L.normalMapTangentSpace&&d.enable(6),L.clearcoat&&d.enable(7),L.iridescence&&d.enable(8),L.alphaTest&&d.enable(9),L.vertexColors&&d.enable(10),L.vertexAlphas&&d.enable(11),L.vertexUv1s&&d.enable(12),L.vertexUv2s&&d.enable(13),L.vertexUv3s&&d.enable(14),L.vertexTangents&&d.enable(15),L.anisotropy&&d.enable(16),L.alphaHash&&d.enable(17),L.batching&&d.enable(18),L.dispersion&&d.enable(19),L.retroreflection&&d.enable(24),L.batchingColor&&d.enable(20),L.gradientMap&&d.enable(21),L.packedNormalMap&&d.enable(22),L.vertexNormals&&d.enable(23),b.push(d.mask),d.disableAll(),L.fog&&d.enable(0),L.useFog&&d.enable(1),L.flatShading&&d.enable(2),L.logarithmicDepthBuffer&&d.enable(3),L.reversedDepthBuffer&&d.enable(4),L.skinning&&d.enable(5),L.morphTargets&&d.enable(6),L.morphNormals&&d.enable(7),L.morphColors&&d.enable(8),L.premultipliedAlpha&&d.enable(9),L.shadowMapEnabled&&d.enable(10),L.doubleSided&&d.enable(11),L.flipSided&&d.enable(12),L.useDepthPacking&&d.enable(13),L.dithering&&d.enable(14),L.transmission&&d.enable(15),L.sheen&&d.enable(16),L.opaque&&d.enable(17),L.pointsUvs&&d.enable(18),L.decodeVideoTexture&&d.enable(19),L.decodeVideoTextureEmissive&&d.enable(20),L.alphaToCoverage&&d.enable(21),L.numLightProbeGrids>0&&d.enable(22),L.hasPositionAttribute&&d.enable(23),b.push(d.mask)}function W(b){const L=E[b.type];let k;if(L){const Y=ta[L];k=ub.clone(Y.uniforms)}else k=b.uniforms;return k}function C(b,L){let k=x.get(L);return k!==void 0?++k.usedTimes:(k=new GR(o,L,b,c),g.push(k),x.set(L,k)),k}function D(b){if(--b.usedTimes===0){const L=g.indexOf(b);g[L]=g[g.length-1],g.pop(),x.delete(b.cacheKey),b.destroy()}}function U(b){h.remove(b)}function I(){h.dispose()}return{getParameters:N,getProgramCacheKey:y,getUniforms:W,acquireProgram:C,releaseProgram:D,releaseShaderCache:U,programs:g,dispose:I}}function YR(){let o=new WeakMap;function t(d){return o.has(d)}function i(d){let h=o.get(d);return h===void 0&&(h={},o.set(d,h)),h}function r(d){o.delete(d)}function c(d,h,m){o.get(d)[h]=m}function f(){o=new WeakMap}return{has:t,get:i,remove:r,update:c,dispose:f}}function jR(o,t){return o.groupOrder!==t.groupOrder?o.groupOrder-t.groupOrder:o.renderOrder!==t.renderOrder?o.renderOrder-t.renderOrder:o.material.id!==t.material.id?o.material.id-t.material.id:o.materialVariant!==t.materialVariant?o.materialVariant-t.materialVariant:o.z!==t.z?o.z-t.z:o.id-t.id}function cx(o,t){return o.groupOrder!==t.groupOrder?o.groupOrder-t.groupOrder:o.renderOrder!==t.renderOrder?o.renderOrder-t.renderOrder:o.z!==t.z?t.z-o.z:o.id-t.id}function ux(){const o=[];let t=0;const i=[],r=[],c=[];function f(){t=0,i.length=0,r.length=0,c.length=0}function d(v){let E=0;return v.isInstancedMesh&&(E+=2),v.isSkinnedMesh&&(E+=1),E}function h(v,E,R,N,y,S){let z=o[t];return z===void 0?(z={id:v.id,object:v,geometry:E,material:R,materialVariant:d(v),groupOrder:N,renderOrder:v.renderOrder,z:y,group:S},o[t]=z):(z.id=v.id,z.object=v,z.geometry=E,z.material=R,z.materialVariant=d(v),z.groupOrder=N,z.renderOrder=v.renderOrder,z.z=y,z.group=S),t++,z}function m(v,E,R,N,y,S,z){z.reversedDepth===!0&&(y=-y);const W=h(v,E,R,N,y,S);R.transmission>0?r.push(W):R.transparent===!0?c.push(W):i.push(W)}function g(v,E,R,N,y,S){const z=h(v,E,R,N,y,S);R.transmission>0?r.unshift(z):R.transparent===!0?c.unshift(z):i.unshift(z)}function x(v,E){i.length>1&&i.sort(v||jR),r.length>1&&r.sort(E||cx),c.length>1&&c.sort(E||cx)}function _(){for(let v=t,E=o.length;v<E;v++){const R=o[v];if(R.id===null)break;R.id=null,R.object=null,R.geometry=null,R.material=null,R.group=null}}return{opaque:i,transmissive:r,transparent:c,init:f,push:m,unshift:g,finish:_,sort:x}}function ZR(){let o=new WeakMap;function t(r,c){const f=o.get(r);let d;return f===void 0?(d=new ux,o.set(r,[d])):c>=f.length?(d=new ux,f.push(d)):d=f[c],d}function i(){o=new WeakMap}return{get:t,dispose:i}}function KR(){const o={};return{get:function(t){if(o[t.id]!==void 0)return o[t.id];let i;switch(t.type){case"SunLight":case"DirectionalLight":i={direction:new he,color:new Dt};break;case"SpotLight":i={position:new he,direction:new he,color:new Dt,distance:0,coneCos:0,penumbraCos:0,decay:0};break;case"PointLight":i={position:new he,color:new Dt,distance:0,decay:0};break;case"HemisphereLight":i={direction:new he,skyColor:new Dt,groundColor:new Dt};break;case"RectAreaLight":i={color:new Dt,position:new he,halfWidth:new he,halfHeight:new he};break}return o[t.id]=i,i}}}function QR(){const o={};return{get:function(t){if(o[t.id]!==void 0)return o[t.id];let i;switch(t.type){case"SunLight":case"DirectionalLight":i={shadowIntensity:1,shadowBias:0,shadowNormalBias:0,shadowRadius:1,shadowMapSize:new Ut};break;case"SpotLight":i={shadowIntensity:1,shadowBias:0,shadowNormalBias:0,shadowRadius:1,shadowMapSize:new Ut};break;case"PointLight":i={shadowIntensity:1,shadowBias:0,shadowNormalBias:0,shadowRadius:1,shadowMapSize:new Ut,shadowCameraNear:1,shadowCameraFar:1e3};break}return o[t.id]=i,i}}}let JR=0;function $R(o,t){return(t.castShadow?2:0)-(o.castShadow?2:0)+(t.map?1:0)-(o.map?1:0)}function ew(o){const t=new KR,i=QR(),r={version:0,hash:{sunLength:-1,directionalLength:-1,pointLength:-1,spotLength:-1,rectAreaLength:-1,hemiLength:-1,numSunShadows:-1,numDirectionalShadows:-1,numPointShadows:-1,numSpotShadows:-1,numSpotMaps:-1,numLightProbes:-1},ambient:[0,0,0],probe:[],sun:[],sunShadow:[],sunShadowMap:[],sunShadowMatrix:[],sunShadowCascade:[],directional:[],directionalShadow:[],directionalShadowMap:[],directionalShadowMatrix:[],spot:[],spotLightMap:[],spotShadow:[],spotShadowMap:[],spotLightMatrix:[],rectArea:[],rectAreaLTC1:null,rectAreaLTC2:null,point:[],pointShadow:[],pointShadowMap:[],pointShadowMatrix:[],hemi:[],numSpotLightShadowsWithMaps:0,numLightProbes:0};for(let g=0;g<9;g++)r.probe.push(new he);const c=new he,f=new un,d=new un;function h(g){let x=0,_=0,v=0;for(let $=0;$<9;$++)r.probe[$].set(0,0,0);let E=0,R=0,N=0,y=0,S=0,z=0,W=0,C=0,D=0,U=0,I=0,b=0,L=0,k=0;g.sort($R);for(let $=0,O=g.length;$<O;$++){const q=g[$],ne=q.color,j=q.intensity,Q=q.distance;let pe=null;if(q.shadow&&q.shadow.map&&(q.shadow.map.texture.format===Js?pe=q.shadow.map.texture:pe=q.shadow.map.depthTexture||q.shadow.map.texture),q.isAmbientLight)x+=ne.r*j,_+=ne.g*j,v+=ne.b*j;else if(q.isLightProbe){for(let ce=0;ce<9;ce++)r.probe[ce].addScaledVector(q.sh.coefficients[ce],j);k++}else if(q.isSunLight){const ce=t.get(q);if(ce.color.copy(q.color).multiplyScalar(q.intensity),q.castShadow){const me=q.shadow,be=i.get(q);be.shadowIntensity=me.intensity,be.shadowBias=me.bias,be.shadowNormalBias=me.normalBias,be.shadowRadius=me.radius,be.shadowMapSize.copy(me.mapSize).multiply(me.getFrameExtents()),r.sunShadow[R]=be,r.sunShadowMap[R]=pe;const Ze=me.getViewportCount();for(let je=0;je<Ze;je++)r.sunShadowMatrix[N+je]=me.getMatrix(je),r.sunShadowCascade[N+je]=me._cascadeData[je];N+=Ze,R++}r.sun[E]=ce,E++}else if(q.isDirectionalLight){const ce=t.get(q);if(ce.color.copy(q.color).multiplyScalar(q.intensity),q.castShadow){const me=q.shadow,be=i.get(q);be.shadowIntensity=me.intensity,be.shadowBias=me.bias,be.shadowNormalBias=me.normalBias,be.shadowRadius=me.radius,be.shadowMapSize=me.mapSize,r.directionalShadow[y]=be,r.directionalShadowMap[y]=pe,r.directionalShadowMatrix[y]=q.shadow.matrix,D++}r.directional[y]=ce,y++}else if(q.isSpotLight){const ce=t.get(q);ce.position.setFromMatrixPosition(q.matrixWorld),ce.color.copy(ne).multiplyScalar(j),ce.distance=Q,ce.coneCos=Math.cos(q.angle),ce.penumbraCos=Math.cos(q.angle*(1-q.penumbra)),ce.decay=q.decay,r.spot[z]=ce;const me=q.shadow;if(q.map&&(r.spotLightMap[b]=q.map,b++,me.updateMatrices(q),q.castShadow&&L++),r.spotLightMatrix[z]=me.matrix,q.castShadow){const be=i.get(q);be.shadowIntensity=me.intensity,be.shadowBias=me.bias,be.shadowNormalBias=me.normalBias,be.shadowRadius=me.radius,be.shadowMapSize=me.mapSize,r.spotShadow[z]=be,r.spotShadowMap[z]=pe,I++}z++}else if(q.isRectAreaLight){const ce=t.get(q);ce.color.copy(ne).multiplyScalar(j),ce.halfWidth.set(q.width*.5,0,0),ce.halfHeight.set(0,q.height*.5,0),r.rectArea[W]=ce,W++}else if(q.isPointLight){const ce=t.get(q);if(ce.color.copy(q.color).multiplyScalar(q.intensity),ce.distance=q.distance,ce.decay=q.decay,q.castShadow){const me=q.shadow,be=i.get(q);be.shadowIntensity=me.intensity,be.shadowBias=me.bias,be.shadowNormalBias=me.normalBias,be.shadowRadius=me.radius,be.shadowMapSize=me.mapSize,be.shadowCameraNear=me.camera.near,be.shadowCameraFar=me.camera.far,r.pointShadow[S]=be,r.pointShadowMap[S]=pe,r.pointShadowMatrix[S]=q.shadow.matrix,U++}r.point[S]=ce,S++}else if(q.isHemisphereLight){const ce=t.get(q);ce.skyColor.copy(q.color).multiplyScalar(j),ce.groundColor.copy(q.groundColor).multiplyScalar(j),r.hemi[C]=ce,C++}}W>0&&(o.has("OES_texture_float_linear")===!0?(r.rectAreaLTC1=He.LTC_FLOAT_1,r.rectAreaLTC2=He.LTC_FLOAT_2):(r.rectAreaLTC1=He.LTC_HALF_1,r.rectAreaLTC2=He.LTC_HALF_2)),r.ambient[0]=x,r.ambient[1]=_,r.ambient[2]=v;const Y=r.hash;(Y.sunLength!==E||Y.directionalLength!==y||Y.pointLength!==S||Y.spotLength!==z||Y.rectAreaLength!==W||Y.hemiLength!==C||Y.numSunShadows!==R||Y.numDirectionalShadows!==D||Y.numPointShadows!==U||Y.numSpotShadows!==I||Y.numSpotMaps!==b||Y.numLightProbes!==k)&&(r.sun.length=E,r.directional.length=y,r.spot.length=z,r.rectArea.length=W,r.point.length=S,r.hemi.length=C,r.sunShadow.length=R,r.sunShadowMap.length=R,r.sunShadowMatrix.length=N,r.sunShadowCascade.length=N,r.directionalShadow.length=D,r.directionalShadowMap.length=D,r.directionalShadowMatrix.length=D,r.pointShadow.length=U,r.pointShadowMap.length=U,r.pointShadowMatrix.length=U,r.spotShadow.length=I,r.spotShadowMap.length=I,r.spotLightMatrix.length=I+b-L,r.spotLightMap.length=b,r.numSpotLightShadowsWithMaps=L,r.numLightProbes=k,Y.sunLength=E,Y.directionalLength=y,Y.pointLength=S,Y.spotLength=z,Y.rectAreaLength=W,Y.hemiLength=C,Y.numSunShadows=R,Y.numDirectionalShadows=D,Y.numPointShadows=U,Y.numSpotShadows=I,Y.numSpotMaps=b,Y.numLightProbes=k,r.version=JR++)}function m(g,x){let _=0,v=0,E=0,R=0,N=0,y=0;const S=x.matrixWorldInverse;for(let z=0,W=g.length;z<W;z++){const C=g[z];if(C.isSunLight){const D=r.sun[_];D.direction.setFromMatrixPosition(C.matrixWorld),D.direction.transformDirection(S),_++}else if(C.isDirectionalLight){const D=r.directional[v];D.direction.setFromMatrixPosition(C.matrixWorld),c.setFromMatrixPosition(C.target.matrixWorld),D.direction.sub(c),D.direction.transformDirection(S),v++}else if(C.isSpotLight){const D=r.spot[R];D.position.setFromMatrixPosition(C.matrixWorld),D.position.applyMatrix4(S),D.direction.setFromMatrixPosition(C.matrixWorld),c.setFromMatrixPosition(C.target.matrixWorld),D.direction.sub(c),D.direction.transformDirection(S),R++}else if(C.isRectAreaLight){const D=r.rectArea[N];D.position.setFromMatrixPosition(C.matrixWorld),D.position.applyMatrix4(S),d.identity(),f.copy(C.matrixWorld),f.premultiply(S),d.extractRotation(f),D.halfWidth.set(C.width*.5,0,0),D.halfHeight.set(0,C.height*.5,0),D.halfWidth.applyMatrix4(d),D.halfHeight.applyMatrix4(d),N++}else if(C.isPointLight){const D=r.point[E];D.position.setFromMatrixPosition(C.matrixWorld),D.position.applyMatrix4(S),E++}else if(C.isHemisphereLight){const D=r.hemi[y];D.direction.setFromMatrixPosition(C.matrixWorld),D.direction.transformDirection(S),y++}}}return{setup:h,setupView:m,state:r}}function fx(o){const t=new ew(o),i=[],r=[],c=[];function f(v){_.camera=v,i.length=0,r.length=0,c.length=0}function d(v){i.push(v)}function h(v){r.push(v)}function m(v){c.push(v)}function g(){t.setup(i)}function x(v){t.setupView(i,v)}const _={lightsArray:i,shadowsArray:r,lightProbeGridArray:c,camera:null,lights:t,transmissionRenderTarget:{},textureUnits:0};return{init:f,state:_,setupLights:g,setupLightsView:x,pushLight:d,pushShadow:h,pushLightProbeGrid:m}}function tw(o){let t=new WeakMap;function i(c,f=0){const d=t.get(c);let h;return d===void 0?(h=new fx(o),t.set(c,[h])):f>=d.length?(h=new fx(o),d.push(h)):h=d[f],h}function r(){t=new WeakMap}return{get:i,dispose:r}}const nw=`void main() {
	gl_Position = vec4( position, 1.0 );
}`,iw=`uniform sampler2D shadow_pass;
uniform vec2 resolution;
uniform float radius;
void main() {
	const float samples = float( VSM_SAMPLES );
	float mean = 0.0;
	float squared_mean = 0.0;
	float uvStride = samples <= 1.0 ? 0.0 : 2.0 / ( samples - 1.0 );
	float uvStart = samples <= 1.0 ? 0.0 : - 1.0;
	for ( float i = 0.0; i < samples; i ++ ) {
		float uvOffset = uvStart + i * uvStride;
		#ifdef HORIZONTAL_PASS
			vec2 distribution = texture2D( shadow_pass, ( gl_FragCoord.xy + vec2( uvOffset, 0.0 ) * radius ) / resolution ).rg;
			mean += distribution.x;
			squared_mean += distribution.y * distribution.y + distribution.x * distribution.x;
		#else
			float depth = texture2D( shadow_pass, ( gl_FragCoord.xy + vec2( 0.0, uvOffset ) * radius ) / resolution ).r;
			mean += depth;
			squared_mean += depth * depth;
		#endif
	}
	mean = mean / samples;
	squared_mean = squared_mean / samples;
	float std_dev = sqrt( max( 0.0, squared_mean - mean * mean ) );
	gl_FragColor = vec4( mean, std_dev, 0.0, 1.0 );
}`,aw=[new he(1,0,0),new he(-1,0,0),new he(0,1,0),new he(0,-1,0),new he(0,0,1),new he(0,0,-1)],sw=[new he(0,-1,0),new he(0,-1,0),new he(0,0,1),new he(0,0,-1),new he(0,-1,0),new he(0,-1,0)],dx=new un,pl=new he,Yh=new he;function rw(o,t,i){let r=new Qp;const c=new Ut,f=new Ut,d=new sn,h=new pb,m=new mb,g={},x=i.maxTextureSize,_={[Ks]:Kn,[Kn]:Ks,[na]:na},v=new ca({defines:{VSM_SAMPLES:8},uniforms:{shadow_pass:{value:null},resolution:{value:new Ut},radius:{value:4}},vertexShader:nw,fragmentShader:iw}),E=v.clone();E.defines.HORIZONTAL_PASS=1;const R=new ua;R.setAttribute("position",new Pa(new Float32Array([-1,-1,.5,3,-1,.5,-1,3,.5]),3));const N=new la(R,v),y=this;this.enabled=!1,this.autoUpdate=!0,this.needsUpdate=!1,this.type=vu;let S=this.type;this.render=function(U,I,b){if(y.enabled===!1||y.autoUpdate===!1&&y.needsUpdate===!1||U.length===0)return;this.type===tE&&(ot("WebGLShadowMap: PCFSoftShadowMap has been removed. Using PCFShadowMap instead."),this.type=vu);const L=o.getRenderTarget(),k=o.getActiveCubeFace(),Y=o.getActiveMipmapLevel(),$=o.state;$.setBlending(La),$.buffers.depth.getReversed()===!0?$.buffers.color.setClear(0,0,0,0):$.buffers.color.setClear(1,1,1,1),$.buffers.depth.setTest(!0),$.setScissorTest(!1);const O=S!==this.type;O&&I.traverse(function(q){q.material&&(Array.isArray(q.material)?q.material.forEach(ne=>ne.needsUpdate=!0):q.material.needsUpdate=!0)});for(let q=0,ne=U.length;q<ne;q++){const j=U[q],Q=j.shadow;if(Q===void 0){ot("WebGLShadowMap:",j,"has no shadow.");continue}if(Q.autoUpdate===!1&&Q.needsUpdate===!1)continue;c.copy(Q.mapSize);const pe=Q.getFrameExtents();c.multiply(pe),f.copy(Q.mapSize),(c.x>x||c.y>x)&&(c.x>x&&(f.x=Math.floor(x/pe.x),c.x=f.x*pe.x,Q.mapSize.x=f.x),c.y>x&&(f.y=Math.floor(x/pe.y),c.y=f.y*pe.y,Q.mapSize.y=f.y));const ce=o.state.buffers.depth.getReversed();if(Q.camera._reversedDepth=ce,Q.map===null||O===!0){if(Q.map!==null&&(Q.map.depthTexture!==null&&(Q.map.depthTexture.dispose(),Q.map.depthTexture=null),Q.map.dispose()),this.type===ml){if(j.isPointLight){ot("WebGLShadowMap: VSM shadow maps are not supported for PointLights. Use PCF or BasicShadowMap instead.");continue}Q.map=new Bi(c.x,c.y,{format:Js,type:oa,minFilter:In,magFilter:In,generateMipmaps:!1}),Q.map.texture.name=j.name+".shadowMap",Q.map.depthTexture=new El(c.x,c.y,ia),Q.map.depthTexture.name=j.name+".shadowMapDepth",Q.map.depthTexture.format=Ia,Q.map.depthTexture.compareFunction=null,Q.map.depthTexture.minFilter=Un,Q.map.depthTexture.magFilter=Un}else j.isPointLight?(Q.map=new Jx(c.x),Q.map.depthTexture=new lb(c.x,ra)):(Q.map=new Bi(c.x,c.y),Q.map.depthTexture=new El(c.x,c.y,ra)),Q.map.depthTexture.name=j.name+".shadowMap",Q.map.depthTexture.format=Ia,this.type===vu?(Q.map.depthTexture.compareFunction=ce?jp:Yp,Q.map.depthTexture.minFilter=In,Q.map.depthTexture.magFilter=In):(Q.map.depthTexture.compareFunction=null,Q.map.depthTexture.minFilter=Un,Q.map.depthTexture.magFilter=Un);Q.camera.updateProjectionMatrix()}Q.map.isWebGLCubeRenderTarget!==!0&&(Q.map.width!==c.x||Q.map.height!==c.y)&&Q.map.setSize(c.x,c.y);const me=Q.map.isWebGLCubeRenderTarget?6:Q.getViewportCount();j.isPointLight!==!0&&Q.updateMatrices(j,b);for(let be=0;be<me;be++){const Ze=Q.getCamera(be);if(j.isPointLight){const je=Q.camera,H=Q.matrix,F=j.distance||je.far;F!==je.far&&(je.far=F,je.updateProjectionMatrix()),pl.setFromMatrixPosition(j.matrixWorld),je.position.copy(pl),Yh.copy(je.position),Yh.add(aw[be]),je.up.copy(sw[be]),je.lookAt(Yh),je.updateMatrixWorld(),H.makeTranslation(-pl.x,-pl.y,-pl.z),dx.multiplyMatrices(je.projectionMatrix,je.matrixWorldInverse),Q._frustum.setFromProjectionMatrix(dx,je.coordinateSystem,je.reversedDepth)}if(Q.map.isWebGLCubeRenderTarget)o.setRenderTarget(Q.map,be),o.clear();else{be===0&&(o.setRenderTarget(Q.map),o.clear());const je=Q.getViewport(be);d.set(f.x*je.x,f.y*je.y,f.x*je.z,f.y*je.w),$.viewport(d)}r=Q.getFrustum(be),C(I,b,Ze,j,this.type)}Q.isPointLightShadow!==!0&&this.type===ml&&z(Q,b),Q.needsUpdate=!1}S=this.type,y.needsUpdate=!1,o.setRenderTarget(L,k,Y)};function z(U,I){const b=t.update(N);v.defines.VSM_SAMPLES!==U.blurSamples&&(v.defines.VSM_SAMPLES=U.blurSamples,E.defines.VSM_SAMPLES=U.blurSamples,v.needsUpdate=!0,E.needsUpdate=!0),U.mapPass===null?U.mapPass=new Bi(c.x,c.y,{format:Js,type:oa}):(U.mapPass.width!==U.map.width||U.mapPass.height!==U.map.height)&&U.mapPass.setSize(U.map.width,U.map.height),v.uniforms.shadow_pass.value=U.map.depthTexture,v.uniforms.resolution.value.set(U.map.width,U.map.height),v.uniforms.radius.value=U.radius,o.setRenderTarget(U.mapPass),o.clear(),o.renderBufferDirect(I,null,b,v,N,null),E.uniforms.shadow_pass.value=U.mapPass.texture,E.uniforms.resolution.value.set(U.map.width,U.map.height),E.uniforms.radius.value=U.radius,o.setRenderTarget(U.map),o.clear(),o.renderBufferDirect(I,null,b,E,N,null)}function W(U,I,b,L){let k=null;const Y=b.isPointLight===!0?U.customDistanceMaterial:U.customDepthMaterial;if(Y!==void 0)k=Y;else if(k=b.isPointLight===!0?m:h,o.localClippingEnabled&&I.clipShadows===!0&&Array.isArray(I.clippingPlanes)&&I.clippingPlanes.length!==0||I.displacementMap&&I.displacementScale!==0||I.alphaMap&&I.alphaTest>0||I.map&&I.alphaTest>0||I.alphaToCoverage===!0){const $=k.uuid,O=I.uuid;let q=g[$];q===void 0&&(q={},g[$]=q);let ne=q[O];ne===void 0&&(ne=k.clone(),q[O]=ne,I.addEventListener("dispose",D)),k=ne}if(k.visible=I.visible,k.wireframe=I.wireframe,L===ml?k.side=I.shadowSide!==null?I.shadowSide:I.side:k.side=I.shadowSide!==null?I.shadowSide:_[I.side],k.alphaMap=I.alphaMap,k.alphaTest=I.alphaToCoverage===!0?.5:I.alphaTest,k.map=I.map,k.clipShadows=I.clipShadows,k.clippingPlanes=I.clippingPlanes,k.clipIntersection=I.clipIntersection,k.displacementMap=I.displacementMap,k.displacementScale=I.displacementScale,k.displacementBias=I.displacementBias,k.wireframeLinewidth=I.wireframeLinewidth,k.linewidth=I.linewidth,b.isPointLight===!0&&k.isMeshDistanceMaterial===!0){const $=o.properties.get(k);$.light=b}return k}function C(U,I,b,L,k){if(U.visible===!1)return;if(U.layers.test(I.layers)&&(U.isMesh||U.isLine||U.isPoints)&&(U.castShadow||U.receiveShadow&&k===ml)&&(!U.frustumCulled||U.intersectsFrustum(r))){U.modelViewMatrix.multiplyMatrices(b.matrixWorldInverse,U.matrixWorld);const O=t.update(U),q=U.material;if(Array.isArray(q)){const ne=O.groups;for(let j=0,Q=ne.length;j<Q;j++){const pe=ne[j],ce=q[pe.materialIndex];if(ce&&ce.visible){const me=W(U,ce,L,k);U.onBeforeShadow(o,U,I,b,O,me,pe),o.renderBufferDirect(b,null,O,me,U,pe),U.onAfterShadow(o,U,I,b,O,me,pe)}}}else if(q.visible){const ne=W(U,q,L,k);U.onBeforeShadow(o,U,I,b,O,ne,null),o.renderBufferDirect(b,null,O,ne,U,null),U.onAfterShadow(o,U,I,b,O,ne,null)}}const $=U.children;for(let O=0,q=$.length;O<q;O++)C($[O],I,b,L,k)}function D(U){U.target.removeEventListener("dispose",D);for(const b in g){const L=g[b],k=U.target.uuid;k in L&&(L[k].dispose(),delete L[k])}}}function ow(o,t){function i(){let Z=!1;const Ce=new sn;let Me=null;const Ue=new sn(0,0,0,0);return{setMask:function(Ve){Me!==Ve&&!Z&&(o.colorMask(Ve,Ve,Ve,Ve),Me=Ve)},setLocked:function(Ve){Z=Ve},setClear:function(Ve,Te,Je,Ge,Rt){Rt===!0&&(Ve*=Ge,Te*=Ge,Je*=Ge),Ce.set(Ve,Te,Je,Ge),Ue.equals(Ce)===!1&&(o.clearColor(Ve,Te,Je,Ge),Ue.copy(Ce))},reset:function(){Z=!1,Me=null,Ue.set(-1,0,0,0)}}}function r(){let Z=!1,Ce=!1,Me=null,Ue=null,Ve=null;return{setReversed:function(Te){if(Ce!==Te){const Je=t.get("EXT_clip_control");Te?Je.clipControlEXT(Je.LOWER_LEFT_EXT,Je.ZERO_TO_ONE_EXT):Je.clipControlEXT(Je.LOWER_LEFT_EXT,Je.NEGATIVE_ONE_TO_ONE_EXT),Ce=Te;const Ge=Ve;Ve=null,this.setClear(Ge)}},getReversed:function(){return Ce},setTest:function(Te){Te?ue(o.DEPTH_TEST):Ae(o.DEPTH_TEST)},setMask:function(Te){Me!==Te&&!Z&&(o.depthMask(Te),Me=Te)},setFunc:function(Te){if(Ce&&(Te=IE[Te]),Ue!==Te){switch(Te){case Zh:o.depthFunc(o.NEVER);break;case Kh:o.depthFunc(o.ALWAYS);break;case Qh:o.depthFunc(o.LESS);break;case xl:o.depthFunc(o.LEQUAL);break;case Jh:o.depthFunc(o.EQUAL);break;case $h:o.depthFunc(o.GEQUAL);break;case ep:o.depthFunc(o.GREATER);break;case tp:o.depthFunc(o.NOTEQUAL);break;default:o.depthFunc(o.LEQUAL)}Ue=Te}},setLocked:function(Te){Z=Te},setClear:function(Te){Ve!==Te&&(Ve=Te,Ce&&(Te=1-Te),o.clearDepth(Te))},reset:function(){Z=!1,Me=null,Ue=null,Ve=null,Ce=!1}}}function c(){let Z=!1,Ce=null,Me=null,Ue=null,Ve=null,Te=null,Je=null,Ge=null,Rt=null;return{setTest:function(ct){Z||(ct?ue(o.STENCIL_TEST):Ae(o.STENCIL_TEST))},setMask:function(ct){Ce!==ct&&!Z&&(o.stencilMask(ct),Ce=ct)},setFunc:function(ct,Jn,di){(Me!==ct||Ue!==Jn||Ve!==di)&&(o.stencilFunc(ct,Jn,di),Me=ct,Ue=Jn,Ve=di)},setOp:function(ct,Jn,di){(Te!==ct||Je!==Jn||Ge!==di)&&(o.stencilOp(ct,Jn,di),Te=ct,Je=Jn,Ge=di)},setLocked:function(ct){Z=ct},setClear:function(ct){Rt!==ct&&(o.clearStencil(ct),Rt=ct)},reset:function(){Z=!1,Ce=null,Me=null,Ue=null,Ve=null,Te=null,Je=null,Ge=null,Rt=null}}}const f=new i,d=new r,h=new c,m=new WeakMap,g=new WeakMap;let x={},_={},v={},E=new WeakMap,R=[],N=null,y=!1,S=null,z=null,W=null,C=null,D=null,U=null,I=null,b=new Dt(0,0,0),L=0,k=!1,Y=null,$=null,O=null,q=null,ne=null;const j=o.getParameter(o.MAX_COMBINED_TEXTURE_IMAGE_UNITS);let Q=!1,pe=0;const ce=o.getParameter(o.VERSION);ce.indexOf("WebGL")!==-1?(pe=parseFloat(/^WebGL (\d)/.exec(ce)[1]),Q=pe>=1):ce.indexOf("OpenGL ES")!==-1&&(pe=parseFloat(/^OpenGL ES (\d)/.exec(ce)[1]),Q=pe>=2);let me=null,be={};const Ze=o.getParameter(o.SCISSOR_BOX),je=o.getParameter(o.VIEWPORT),H=new sn().fromArray(Ze),F=new sn().fromArray(je);function ge(Z,Ce,Me,Ue){const Ve=new Uint8Array(4),Te=o.createTexture();o.bindTexture(Z,Te),o.texParameteri(Z,o.TEXTURE_MIN_FILTER,o.NEAREST),o.texParameteri(Z,o.TEXTURE_MAG_FILTER,o.NEAREST);for(let Je=0;Je<Me;Je++)Z===o.TEXTURE_3D||Z===o.TEXTURE_2D_ARRAY?o.texImage3D(Ce,0,o.RGBA,1,1,Ue,0,o.RGBA,o.UNSIGNED_BYTE,Ve):o.texImage2D(Ce+Je,0,o.RGBA,1,1,0,o.RGBA,o.UNSIGNED_BYTE,Ve);return Te}const X={};X[o.TEXTURE_2D]=ge(o.TEXTURE_2D,o.TEXTURE_2D,1),X[o.TEXTURE_CUBE_MAP]=ge(o.TEXTURE_CUBE_MAP,o.TEXTURE_CUBE_MAP_POSITIVE_X,6),X[o.TEXTURE_2D_ARRAY]=ge(o.TEXTURE_2D_ARRAY,o.TEXTURE_2D_ARRAY,1,1),X[o.TEXTURE_3D]=ge(o.TEXTURE_3D,o.TEXTURE_3D,1,1),f.setClear(0,0,0,1),d.setClear(1),h.setClear(0),ue(o.DEPTH_TEST),d.setFunc(xl),tt(!1),it(_v),ue(o.CULL_FACE),gt(La);function ue(Z){x[Z]!==!0&&(o.enable(Z),x[Z]=!0)}function Ae(Z){x[Z]!==!1&&(o.disable(Z),x[Z]=!1)}function Ie(Z,Ce){return v[Z]!==Ce?(o.bindFramebuffer(Z,Ce),v[Z]=Ce,Z===o.DRAW_FRAMEBUFFER&&(v[o.FRAMEBUFFER]=Ce),Z===o.FRAMEBUFFER&&(v[o.DRAW_FRAMEBUFFER]=Ce),!0):!1}function xe(Z,Ce){let Me=R,Ue=!1;if(Z){Me=E.get(Ce),Me===void 0&&(Me=[],E.set(Ce,Me));const Ve=Z.textures;if(Me.length!==Ve.length||Me[0]!==o.COLOR_ATTACHMENT0){for(let Te=0,Je=Ve.length;Te<Je;Te++)Me[Te]=o.COLOR_ATTACHMENT0+Te;Me.length=Ve.length,Ue=!0}}else Me[0]!==o.BACK&&(Me[0]=o.BACK,Ue=!0);Ue&&o.drawBuffers(Me)}function we(Z){return N!==Z?(o.useProgram(Z),N=Z,!0):!1}const Vt={[eo]:o.FUNC_ADD,[iE]:o.FUNC_SUBTRACT,[aE]:o.FUNC_REVERSE_SUBTRACT};Vt[sE]=o.MIN,Vt[rE]=o.MAX;const pt={[oE]:o.ZERO,[lE]:o.ONE,[cE]:o.SRC_COLOR,[xx]:o.SRC_ALPHA,[mE]:o.SRC_ALPHA_SATURATE,[hE]:o.DST_COLOR,[fE]:o.DST_ALPHA,[uE]:o.ONE_MINUS_SRC_COLOR,[Sx]:o.ONE_MINUS_SRC_ALPHA,[pE]:o.ONE_MINUS_DST_COLOR,[dE]:o.ONE_MINUS_DST_ALPHA,[gE]:o.CONSTANT_COLOR,[_E]:o.ONE_MINUS_CONSTANT_COLOR,[vE]:o.CONSTANT_ALPHA,[xE]:o.ONE_MINUS_CONSTANT_ALPHA};function gt(Z,Ce,Me,Ue,Ve,Te,Je,Ge,Rt,ct){if(Z===La){y===!0&&(Ae(o.BLEND),y=!1);return}if(y===!1&&(ue(o.BLEND),y=!0),Z!==nE){if(Z!==S||ct!==k){if((z!==eo||D!==eo)&&(o.blendEquation(o.FUNC_ADD),z=eo,D=eo),ct)switch(Z){case vl:o.blendFuncSeparate(o.ONE,o.ONE_MINUS_SRC_ALPHA,o.ONE,o.ONE_MINUS_SRC_ALPHA);break;case vv:o.blendFunc(o.ONE,o.ONE);break;case xv:o.blendFuncSeparate(o.ZERO,o.ONE_MINUS_SRC_COLOR,o.ZERO,o.ONE);break;case Sv:o.blendFuncSeparate(o.DST_COLOR,o.ONE_MINUS_SRC_ALPHA,o.ZERO,o.ONE);break;default:It("WebGLState: Invalid blending: ",Z);break}else switch(Z){case vl:o.blendFuncSeparate(o.SRC_ALPHA,o.ONE_MINUS_SRC_ALPHA,o.ONE,o.ONE_MINUS_SRC_ALPHA);break;case vv:o.blendFuncSeparate(o.SRC_ALPHA,o.ONE,o.ONE,o.ONE);break;case xv:It("WebGLState: SubtractiveBlending requires material.premultipliedAlpha = true");break;case Sv:It("WebGLState: MultiplyBlending requires material.premultipliedAlpha = true");break;default:It("WebGLState: Invalid blending: ",Z);break}W=null,C=null,U=null,I=null,b.set(0,0,0),L=0,S=Z,k=ct}return}Ve=Ve||Ce,Te=Te||Me,Je=Je||Ue,(Ce!==z||Ve!==D)&&(o.blendEquationSeparate(Vt[Ce],Vt[Ve]),z=Ce,D=Ve),(Me!==W||Ue!==C||Te!==U||Je!==I)&&(o.blendFuncSeparate(pt[Me],pt[Ue],pt[Te],pt[Je]),W=Me,C=Ue,U=Te,I=Je),(Ge.equals(b)===!1||Rt!==L)&&(o.blendColor(Ge.r,Ge.g,Ge.b,Rt),b.copy(Ge),L=Rt),S=Z,k=!1}function St(Z,Ce){Z.side===na?Ae(o.CULL_FACE):ue(o.CULL_FACE);let Me=Z.side===Kn;Ce&&(Me=!Me),tt(Me),Z.blending===vl&&Z.transparent===!1?gt(La):gt(Z.blending,Z.blendEquation,Z.blendSrc,Z.blendDst,Z.blendEquationAlpha,Z.blendSrcAlpha,Z.blendDstAlpha,Z.blendColor,Z.blendAlpha,Z.premultipliedAlpha),d.setFunc(Z.depthFunc),d.setTest(Z.depthTest),d.setMask(Z.depthWrite),f.setMask(Z.colorWrite);const Ue=Z.stencilWrite;h.setTest(Ue),Ue&&(h.setMask(Z.stencilWriteMask),h.setFunc(Z.stencilFunc,Z.stencilRef,Z.stencilFuncMask),h.setOp(Z.stencilFail,Z.stencilZFail,Z.stencilZPass)),dn(Z.polygonOffset,Z.polygonOffsetFactor,Z.polygonOffsetUnits),Z.alphaToCoverage===!0?ue(o.SAMPLE_ALPHA_TO_COVERAGE):Ae(o.SAMPLE_ALPHA_TO_COVERAGE)}function tt(Z){Y!==Z&&(Z?o.frontFace(o.CW):o.frontFace(o.CCW),Y=Z)}function it(Z){Z!==$M?(ue(o.CULL_FACE),Z!==$&&(Z===_v?o.cullFace(o.BACK):Z===eE?o.cullFace(o.FRONT):o.cullFace(o.FRONT_AND_BACK))):Ae(o.CULL_FACE),$=Z}function kt(Z){Z!==O&&(Q&&o.lineWidth(Z),O=Z)}function dn(Z,Ce,Me){Z?(ue(o.POLYGON_OFFSET_FILL),(q!==Ce||ne!==Me)&&(q=Ce,ne=Me,d.getReversed()&&(Ce=-Ce),o.polygonOffset(Ce,Me))):Ae(o.POLYGON_OFFSET_FILL)}function Pt(Z){Z?ue(o.SCISSOR_TEST):Ae(o.SCISSOR_TEST)}function en(Z){Z===void 0&&(Z=o.TEXTURE0+j-1),me!==Z&&(o.activeTexture(Z),me=Z)}function K(Z,Ce,Me){Me===void 0&&(me===null?Me=o.TEXTURE0+j-1:Me=me);let Ue=be[Me];Ue===void 0&&(Ue={type:void 0,texture:void 0},be[Me]=Ue),(Ue.type!==Z||Ue.texture!==Ce)&&(me!==Me&&(o.activeTexture(Me),me=Me),o.bindTexture(Z,Ce||X[Z]),Ue.type=Z,Ue.texture=Ce)}function nn(){const Z=be[me];Z!==void 0&&Z.type!==void 0&&(o.bindTexture(Z.type,null),Z.type=void 0,Z.texture=void 0)}function Ot(){try{o.compressedTexImage2D(...arguments)}catch(Z){It("WebGLState:",Z)}}function P(){try{o.compressedTexImage3D(...arguments)}catch(Z){It("WebGLState:",Z)}}function M(){try{o.texSubImage2D(...arguments)}catch(Z){It("WebGLState:",Z)}}function ie(){try{o.texSubImage3D(...arguments)}catch(Z){It("WebGLState:",Z)}}function le(){try{o.compressedTexSubImage2D(...arguments)}catch(Z){It("WebGLState:",Z)}}function _e(){try{o.compressedTexSubImage3D(...arguments)}catch(Z){It("WebGLState:",Z)}}function Re(){try{o.texStorage2D(...arguments)}catch(Z){It("WebGLState:",Z)}}function De(){try{o.texStorage3D(...arguments)}catch(Z){It("WebGLState:",Z)}}function ve(){try{o.texImage2D(...arguments)}catch(Z){It("WebGLState:",Z)}}function Ee(){try{o.texImage3D(...arguments)}catch(Z){It("WebGLState:",Z)}}function Ne(Z){return _[Z]!==void 0?_[Z]:o.getParameter(Z)}function $e(Z,Ce){_[Z]!==Ce&&(o.pixelStorei(Z,Ce),_[Z]=Ce)}function Be(Z){H.equals(Z)===!1&&(o.scissor(Z.x,Z.y,Z.z,Z.w),H.copy(Z))}function Pe(Z){F.equals(Z)===!1&&(o.viewport(Z.x,Z.y,Z.z,Z.w),F.copy(Z))}function ke(Z,Ce){let Me=g.get(Ce);Me===void 0&&(Me=new WeakMap,g.set(Ce,Me));let Ue=Me.get(Z);Ue===void 0&&(Ue=o.getUniformBlockIndex(Ce,Z.name),Me.set(Z,Ue))}function nt(Z,Ce){const Ue=g.get(Ce).get(Z);m.get(Ce)!==Ue&&(o.uniformBlockBinding(Ce,Ue,Z.__bindingPointIndex),m.set(Ce,Ue))}function lt(){o.disable(o.BLEND),o.disable(o.CULL_FACE),o.disable(o.DEPTH_TEST),o.disable(o.POLYGON_OFFSET_FILL),o.disable(o.SCISSOR_TEST),o.disable(o.STENCIL_TEST),o.disable(o.SAMPLE_ALPHA_TO_COVERAGE),o.blendEquation(o.FUNC_ADD),o.blendFunc(o.ONE,o.ZERO),o.blendFuncSeparate(o.ONE,o.ZERO,o.ONE,o.ZERO),o.blendColor(0,0,0,0),o.colorMask(!0,!0,!0,!0),o.clearColor(0,0,0,0),o.depthMask(!0),o.depthFunc(o.LESS),d.setReversed(!1),o.clearDepth(1),o.stencilMask(4294967295),o.stencilFunc(o.ALWAYS,0,4294967295),o.stencilOp(o.KEEP,o.KEEP,o.KEEP),o.clearStencil(0),o.cullFace(o.BACK),o.frontFace(o.CCW),o.polygonOffset(0,0),o.activeTexture(o.TEXTURE0),o.bindFramebuffer(o.FRAMEBUFFER,null),o.bindFramebuffer(o.DRAW_FRAMEBUFFER,null),o.bindFramebuffer(o.READ_FRAMEBUFFER,null),o.useProgram(null),o.lineWidth(1),o.scissor(0,0,o.canvas.width,o.canvas.height),o.viewport(0,0,o.canvas.width,o.canvas.height),o.pixelStorei(o.PACK_ALIGNMENT,4),o.pixelStorei(o.UNPACK_ALIGNMENT,4),o.pixelStorei(o.UNPACK_FLIP_Y_WEBGL,!1),o.pixelStorei(o.UNPACK_PREMULTIPLY_ALPHA_WEBGL,!1),o.pixelStorei(o.UNPACK_COLORSPACE_CONVERSION_WEBGL,o.BROWSER_DEFAULT_WEBGL),o.pixelStorei(o.PACK_ROW_LENGTH,0),o.pixelStorei(o.PACK_SKIP_PIXELS,0),o.pixelStorei(o.PACK_SKIP_ROWS,0),o.pixelStorei(o.UNPACK_ROW_LENGTH,0),o.pixelStorei(o.UNPACK_IMAGE_HEIGHT,0),o.pixelStorei(o.UNPACK_SKIP_PIXELS,0),o.pixelStorei(o.UNPACK_SKIP_ROWS,0),o.pixelStorei(o.UNPACK_SKIP_IMAGES,0),x={},_={},me=null,be={},v={},E=new WeakMap,R=[],N=null,y=!1,S=null,z=null,W=null,C=null,D=null,U=null,I=null,b=new Dt(0,0,0),L=0,k=!1,Y=null,$=null,O=null,q=null,ne=null,H.set(0,0,o.canvas.width,o.canvas.height),F.set(0,0,o.canvas.width,o.canvas.height),f.reset(),d.reset(),h.reset()}return{buffers:{color:f,depth:d,stencil:h},enable:ue,disable:Ae,bindFramebuffer:Ie,drawBuffers:xe,useProgram:we,setBlending:gt,setMaterial:St,setFlipSided:tt,setCullFace:it,setLineWidth:kt,setPolygonOffset:dn,setScissorTest:Pt,activeTexture:en,bindTexture:K,unbindTexture:nn,compressedTexImage2D:Ot,compressedTexImage3D:P,texImage2D:ve,texImage3D:Ee,pixelStorei:$e,getParameter:Ne,updateUBOMapping:ke,uniformBlockBinding:nt,texStorage2D:Re,texStorage3D:De,texSubImage2D:M,texSubImage3D:ie,compressedTexSubImage2D:le,compressedTexSubImage3D:_e,scissor:Be,viewport:Pe,reset:lt}}function lw(o,t,i,r,c,f,d){const h=t.has("WEBGL_multisampled_render_to_texture")?t.get("WEBGL_multisampled_render_to_texture"):null,m=typeof navigator>"u"?!1:/OculusBrowser/g.test(navigator.userAgent),g=new Ut,x=new WeakMap,_=new Set;let v;const E=new WeakMap;let R=!1;try{R=typeof OffscreenCanvas<"u"&&new OffscreenCanvas(1,1).getContext("2d")!==null}catch{}function N(P,M){return R?new OffscreenCanvas(P,M):wu("canvas")}function y(P,M,ie){let le=1;const _e=Ot(P);if((_e.width>ie||_e.height>ie)&&(le=ie/Math.max(_e.width,_e.height)),le<1)if(typeof HTMLImageElement<"u"&&P instanceof HTMLImageElement||typeof HTMLCanvasElement<"u"&&P instanceof HTMLCanvasElement||typeof ImageBitmap<"u"&&P instanceof ImageBitmap||typeof VideoFrame<"u"&&P instanceof VideoFrame){const Re=Math.floor(le*_e.width),De=Math.floor(le*_e.height);v===void 0&&(v=N(Re,De));const ve=M?N(Re,De):v;return ve.width=Re,ve.height=De,ve.getContext("2d").drawImage(P,0,0,Re,De),ot("WebGLRenderer: Texture has been resized from ("+_e.width+"x"+_e.height+") to ("+Re+"x"+De+")."),ve}else return"data"in P&&ot("WebGLRenderer: Image in DataTexture is too big ("+_e.width+"x"+_e.height+")."),P;return P}function S(P){return P.generateMipmaps}function z(P){o.generateMipmap(P)}function W(P){return P.isWebGLCubeRenderTarget?o.TEXTURE_CUBE_MAP:P.isWebGL3DRenderTarget?o.TEXTURE_3D:P.isWebGLArrayRenderTarget||P.isCompressedArrayTexture?o.TEXTURE_2D_ARRAY:o.TEXTURE_2D}function C(P,M,ie,le,_e,Re=!1){if(P!==null){if(o[P]!==void 0)return o[P];ot("WebGLRenderer: Attempt to use non-existing WebGL internal format '"+P+"'")}let De;le&&(De=t.get("EXT_texture_norm16"),De||ot("WebGLRenderer: Unable to use normalized textures without EXT_texture_norm16 extension"));let ve=M;if(M===o.RED&&(ie===o.FLOAT&&(ve=o.R32F),ie===o.HALF_FLOAT&&(ve=o.R16F),ie===o.UNSIGNED_BYTE&&(ve=o.R8),ie===o.UNSIGNED_SHORT&&De&&(ve=De.R16_EXT),ie===o.SHORT&&De&&(ve=De.R16_SNORM_EXT)),M===o.RED_INTEGER&&(ie===o.UNSIGNED_BYTE&&(ve=o.R8UI),ie===o.UNSIGNED_SHORT&&(ve=o.R16UI),ie===o.UNSIGNED_INT&&(ve=o.R32UI),ie===o.BYTE&&(ve=o.R8I),ie===o.SHORT&&(ve=o.R16I),ie===o.INT&&(ve=o.R32I)),M===o.RG&&(ie===o.FLOAT&&(ve=o.RG32F),ie===o.HALF_FLOAT&&(ve=o.RG16F),ie===o.UNSIGNED_BYTE&&(ve=o.RG8),ie===o.UNSIGNED_SHORT&&De&&(ve=De.RG16_EXT),ie===o.SHORT&&De&&(ve=De.RG16_SNORM_EXT)),M===o.RG_INTEGER&&(ie===o.UNSIGNED_BYTE&&(ve=o.RG8UI),ie===o.UNSIGNED_SHORT&&(ve=o.RG16UI),ie===o.UNSIGNED_INT&&(ve=o.RG32UI),ie===o.BYTE&&(ve=o.RG8I),ie===o.SHORT&&(ve=o.RG16I),ie===o.INT&&(ve=o.RG32I)),M===o.RGB_INTEGER&&(ie===o.UNSIGNED_BYTE&&(ve=o.RGB8UI),ie===o.UNSIGNED_SHORT&&(ve=o.RGB16UI),ie===o.UNSIGNED_INT&&(ve=o.RGB32UI),ie===o.BYTE&&(ve=o.RGB8I),ie===o.SHORT&&(ve=o.RGB16I),ie===o.INT&&(ve=o.RGB32I)),M===o.RGBA_INTEGER&&(ie===o.UNSIGNED_BYTE&&(ve=o.RGBA8UI),ie===o.UNSIGNED_SHORT&&(ve=o.RGBA16UI),ie===o.UNSIGNED_INT&&(ve=o.RGBA32UI),ie===o.BYTE&&(ve=o.RGBA8I),ie===o.SHORT&&(ve=o.RGBA16I),ie===o.INT&&(ve=o.RGBA32I)),M===o.RGB&&(ie===o.UNSIGNED_SHORT&&De&&(ve=De.RGB16_EXT),ie===o.SHORT&&De&&(ve=De.RGB16_SNORM_EXT),ie===o.UNSIGNED_INT_5_9_9_9_REV&&(ve=o.RGB9_E5),ie===o.UNSIGNED_INT_10F_11F_11F_REV&&(ve=o.R11F_G11F_B10F)),M===o.RGBA){const Ee=Re?Ru:Ct.getTransfer(_e);ie===o.FLOAT&&(ve=o.RGBA32F),ie===o.HALF_FLOAT&&(ve=o.RGBA16F),ie===o.UNSIGNED_BYTE&&(ve=Ee===Yt?o.SRGB8_ALPHA8:o.RGBA8),ie===o.UNSIGNED_SHORT&&De&&(ve=De.RGBA16_EXT),ie===o.SHORT&&De&&(ve=De.RGBA16_SNORM_EXT),ie===o.UNSIGNED_SHORT_4_4_4_4&&(ve=o.RGBA4),ie===o.UNSIGNED_SHORT_5_5_5_1&&(ve=o.RGB5_A1)}return(ve===o.R16F||ve===o.R32F||ve===o.RG16F||ve===o.RG32F||ve===o.RGBA16F||ve===o.RGBA32F)&&t.get("EXT_color_buffer_float"),ve}function D(P,M){let ie;return P?M===null||M===ra||M===yl?ie=o.DEPTH24_STENCIL8:M===ia?ie=o.DEPTH32F_STENCIL8:M===Sl&&(ie=o.DEPTH24_STENCIL8,ot("DepthTexture: 16 bit depth attachment is not supported with stencil. Using 24-bit attachment.")):M===null||M===ra||M===yl?ie=o.DEPTH_COMPONENT24:M===ia?ie=o.DEPTH_COMPONENT32F:M===Sl&&(ie=o.DEPTH_COMPONENT16),ie}function U(P,M){return S(P)===!0||P.isFramebufferTexture&&P.minFilter!==Un&&P.minFilter!==In?Math.log2(Math.max(M.width,M.height))+1:P.mipmaps!==void 0&&P.mipmaps.length>0?P.mipmaps.length:P.isCompressedTexture&&Array.isArray(P.image)?M.mipmaps.length:1}function I(P){const M=P.target;M.removeEventListener("dispose",I),L(M),M.isVideoTexture&&x.delete(M),M.isHTMLTexture&&_.delete(M)}function b(P){const M=P.target;M.removeEventListener("dispose",b),Y(M)}function L(P){const M=r.get(P);if(M.__webglInit===void 0)return;const ie=P.source,le=E.get(ie);if(le){const _e=le[M.__cacheKey];_e.usedTimes--,_e.usedTimes===0&&k(P),Object.keys(le).length===0&&E.delete(ie)}r.remove(P)}function k(P){const M=r.get(P);o.deleteTexture(M.__webglTexture);const ie=P.source,le=E.get(ie);delete le[M.__cacheKey],d.memory.textures--}function Y(P){const M=r.get(P);if(P.depthTexture&&(P.depthTexture.dispose(),r.remove(P.depthTexture)),P.isWebGLCubeRenderTarget)for(let le=0;le<6;le++){if(Array.isArray(M.__webglFramebuffer[le]))for(let _e=0;_e<M.__webglFramebuffer[le].length;_e++)o.deleteFramebuffer(M.__webglFramebuffer[le][_e]);else o.deleteFramebuffer(M.__webglFramebuffer[le]);M.__webglDepthbuffer&&o.deleteRenderbuffer(M.__webglDepthbuffer[le])}else{if(Array.isArray(M.__webglFramebuffer))for(let le=0;le<M.__webglFramebuffer.length;le++)o.deleteFramebuffer(M.__webglFramebuffer[le]);else o.deleteFramebuffer(M.__webglFramebuffer);if(M.__webglDepthbuffer&&o.deleteRenderbuffer(M.__webglDepthbuffer),M.__webglMultisampledFramebuffer&&o.deleteFramebuffer(M.__webglMultisampledFramebuffer),M.__webglColorRenderbuffer)for(let le=0;le<M.__webglColorRenderbuffer.length;le++)M.__webglColorRenderbuffer[le]&&o.deleteRenderbuffer(M.__webglColorRenderbuffer[le]);M.__webglDepthRenderbuffer&&o.deleteRenderbuffer(M.__webglDepthRenderbuffer)}const ie=P.textures;for(let le=0,_e=ie.length;le<_e;le++){const Re=r.get(ie[le]);Re.__webglTexture&&(o.deleteTexture(Re.__webglTexture),d.memory.textures--),r.remove(ie[le])}r.remove(P)}let $=0;function O(){$=0}function q(){return $}function ne(P){$=P}function j(){const P=$;return P>=c.maxTextures&&ot("WebGLTextures: Trying to use "+(P+1)+" texture units while this GPU supports only "+c.maxTextures),$+=1,P}function Q(P){const M=[];return M.push(P.wrapS),M.push(P.wrapT),M.push(P.wrapR||0),M.push(P.magFilter),M.push(P.minFilter),M.push(P.anisotropy),M.push(P.internalFormat),M.push(P.format),M.push(P.type),M.push(P.generateMipmaps),M.push(P.premultiplyAlpha),M.push(P.flipY),M.push(P.unpackAlignment),M.push(P.colorSpace),M.join()}function pe(P,M){const ie=r.get(P);if(P.isVideoTexture&&K(P),P.isRenderTargetTexture===!1&&P.isExternalTexture!==!0&&P.version>0&&ie.__version!==P.version){const le=P.image;if(le===null)ot("WebGLRenderer: Texture marked for update but no image data found.");else if(le.complete===!1)ot("WebGLRenderer: Texture marked for update but image is incomplete");else{Ae(ie,P,M);return}}else P.isExternalTexture&&(ie.__webglTexture=P.sourceTexture?P.sourceTexture:null);i.bindTexture(o.TEXTURE_2D,ie.__webglTexture,o.TEXTURE0+M)}function ce(P,M){const ie=r.get(P);if(P.isRenderTargetTexture===!1&&P.version>0&&ie.__version!==P.version){Ae(ie,P,M);return}else P.isExternalTexture&&(ie.__webglTexture=P.sourceTexture?P.sourceTexture:null);i.bindTexture(o.TEXTURE_2D_ARRAY,ie.__webglTexture,o.TEXTURE0+M)}function me(P,M){const ie=r.get(P);if(P.isRenderTargetTexture===!1&&P.version>0&&ie.__version!==P.version){Ae(ie,P,M);return}i.bindTexture(o.TEXTURE_3D,ie.__webglTexture,o.TEXTURE0+M)}function be(P,M){const ie=r.get(P);if(P.isCubeDepthTexture!==!0&&P.version>0&&ie.__version!==P.version){Ie(ie,P,M);return}i.bindTexture(o.TEXTURE_CUBE_MAP,ie.__webglTexture,o.TEXTURE0+M)}const Ze={[np]:o.REPEAT,[Ua]:o.CLAMP_TO_EDGE,[ip]:o.MIRRORED_REPEAT},je={[Un]:o.NEAREST,[ME]:o.NEAREST_MIPMAP_NEAREST,[Zc]:o.NEAREST_MIPMAP_LINEAR,[In]:o.LINEAR,[gh]:o.LINEAR_MIPMAP_NEAREST,[js]:o.LINEAR_MIPMAP_LINEAR},H={[AE]:o.NEVER,[DE]:o.ALWAYS,[RE]:o.LESS,[Yp]:o.LEQUAL,[wE]:o.EQUAL,[jp]:o.GEQUAL,[CE]:o.GREATER,[NE]:o.NOTEQUAL};function F(P,M){if(M.type===ia&&t.has("OES_texture_float_linear")===!1&&(M.magFilter===In||M.magFilter===gh||M.magFilter===Zc||M.magFilter===js||M.minFilter===In||M.minFilter===gh||M.minFilter===Zc||M.minFilter===js)&&ot("WebGLRenderer: Unable to use linear filtering with floating point textures. OES_texture_float_linear not supported on this device."),o.texParameteri(P,o.TEXTURE_WRAP_S,Ze[M.wrapS]),o.texParameteri(P,o.TEXTURE_WRAP_T,Ze[M.wrapT]),(P===o.TEXTURE_3D||P===o.TEXTURE_2D_ARRAY)&&o.texParameteri(P,o.TEXTURE_WRAP_R,Ze[M.wrapR]),o.texParameteri(P,o.TEXTURE_MAG_FILTER,je[M.magFilter]),o.texParameteri(P,o.TEXTURE_MIN_FILTER,je[M.minFilter]),M.compareFunction&&(o.texParameteri(P,o.TEXTURE_COMPARE_MODE,o.COMPARE_REF_TO_TEXTURE),o.texParameteri(P,o.TEXTURE_COMPARE_FUNC,H[M.compareFunction])),t.has("EXT_texture_filter_anisotropic")===!0){if(M.magFilter===Un||M.minFilter!==Zc&&M.minFilter!==js||M.type===ia&&t.has("OES_texture_float_linear")===!1)return;if(M.anisotropy>1||r.get(M).__currentAnisotropy){const ie=t.get("EXT_texture_filter_anisotropic");o.texParameterf(P,ie.TEXTURE_MAX_ANISOTROPY_EXT,Math.min(M.anisotropy,c.getMaxAnisotropy())),r.get(M).__currentAnisotropy=M.anisotropy}}}function ge(P,M){let ie=!1;P.__webglInit===void 0&&(P.__webglInit=!0,M.addEventListener("dispose",I));const le=M.source;let _e=E.get(le);_e===void 0&&(_e={},E.set(le,_e));const Re=Q(M);if(Re!==P.__cacheKey){_e[Re]===void 0&&(_e[Re]={texture:o.createTexture(),usedTimes:0},d.memory.textures++,ie=!0),_e[Re].usedTimes++;const De=_e[P.__cacheKey];De!==void 0&&(_e[P.__cacheKey].usedTimes--,De.usedTimes===0&&k(M)),P.__cacheKey=Re,P.__webglTexture=_e[Re].texture}return ie}function X(P,M,ie){return Math.floor(Math.floor(P/ie)/M)}function ue(P,M,ie,le){const Re=P.updateRanges;if(Re.length===0)i.texSubImage2D(o.TEXTURE_2D,0,0,0,M.width,M.height,ie,le,M.data);else{Re.sort(($e,Be)=>$e.start-Be.start);let De=0;for(let $e=1;$e<Re.length;$e++){const Be=Re[De],Pe=Re[$e],ke=Be.start+Be.count,nt=X(Pe.start,M.width,4),lt=X(Be.start,M.width,4);Pe.start<=ke+1&&nt===lt&&X(Pe.start+Pe.count-1,M.width,4)===nt?Be.count=Math.max(Be.count,Pe.start+Pe.count-Be.start):(++De,Re[De]=Pe)}Re.length=De+1;const ve=i.getParameter(o.UNPACK_ROW_LENGTH),Ee=i.getParameter(o.UNPACK_SKIP_PIXELS),Ne=i.getParameter(o.UNPACK_SKIP_ROWS);i.pixelStorei(o.UNPACK_ROW_LENGTH,M.width);for(let $e=0,Be=Re.length;$e<Be;$e++){const Pe=Re[$e],ke=Math.floor(Pe.start/4),nt=Math.ceil(Pe.count/4),lt=ke%M.width,Z=Math.floor(ke/M.width),Ce=nt,Me=1;i.pixelStorei(o.UNPACK_SKIP_PIXELS,lt),i.pixelStorei(o.UNPACK_SKIP_ROWS,Z),i.texSubImage2D(o.TEXTURE_2D,0,lt,Z,Ce,Me,ie,le,M.data)}P.clearUpdateRanges(),i.pixelStorei(o.UNPACK_ROW_LENGTH,ve),i.pixelStorei(o.UNPACK_SKIP_PIXELS,Ee),i.pixelStorei(o.UNPACK_SKIP_ROWS,Ne)}}function Ae(P,M,ie){let le=o.TEXTURE_2D;(M.isDataArrayTexture||M.isCompressedArrayTexture)&&(le=o.TEXTURE_2D_ARRAY),M.isData3DTexture&&(le=o.TEXTURE_3D);const _e=ge(P,M),Re=M.source;i.bindTexture(le,P.__webglTexture,o.TEXTURE0+ie);const De=r.get(Re);if(Re.version!==De.__version||_e===!0){if(i.activeTexture(o.TEXTURE0+ie),(typeof ImageBitmap<"u"&&M.image instanceof ImageBitmap)===!1){const Me=Ct.getPrimaries(Ct.workingColorSpace),Ue=M.colorSpace===vs?null:Ct.getPrimaries(M.colorSpace),Ve=M.colorSpace===vs||Me===Ue?o.NONE:o.BROWSER_DEFAULT_WEBGL;i.pixelStorei(o.UNPACK_FLIP_Y_WEBGL,M.flipY),i.pixelStorei(o.UNPACK_PREMULTIPLY_ALPHA_WEBGL,M.premultiplyAlpha),i.pixelStorei(o.UNPACK_COLORSPACE_CONVERSION_WEBGL,Ve)}i.pixelStorei(o.UNPACK_ALIGNMENT,M.unpackAlignment);let Ee=y(M.image,!1,c.maxTextureSize);Ee=nn(M,Ee);const Ne=f.convert(M.format,M.colorSpace),$e=f.convert(M.type);let Be=C(M.internalFormat,Ne,$e,M.normalized,M.colorSpace,M.isVideoTexture);F(le,M);let Pe;const ke=M.mipmaps,nt=M.isVideoTexture!==!0,lt=De.__version===void 0||_e===!0,Z=Re.dataReady,Ce=U(M,Ee);if(M.isDepthTexture)Be=D(M.format===Zs,M.type),lt&&(nt?i.texStorage2D(o.TEXTURE_2D,1,Be,Ee.width,Ee.height):i.texImage2D(o.TEXTURE_2D,0,Be,Ee.width,Ee.height,0,Ne,$e,null));else if(M.isDataTexture)if(ke.length>0){nt&&lt&&i.texStorage2D(o.TEXTURE_2D,Ce,Be,ke[0].width,ke[0].height);for(let Me=0,Ue=ke.length;Me<Ue;Me++)Pe=ke[Me],nt?Z&&i.texSubImage2D(o.TEXTURE_2D,Me,0,0,Pe.width,Pe.height,Ne,$e,Pe.data):i.texImage2D(o.TEXTURE_2D,Me,Be,Pe.width,Pe.height,0,Ne,$e,Pe.data);M.generateMipmaps=!1}else nt?(lt&&i.texStorage2D(o.TEXTURE_2D,Ce,Be,Ee.width,Ee.height),Z&&ue(M,Ee,Ne,$e)):i.texImage2D(o.TEXTURE_2D,0,Be,Ee.width,Ee.height,0,Ne,$e,Ee.data);else if(M.isCompressedTexture)if(M.isCompressedArrayTexture){nt&&lt&&i.texStorage3D(o.TEXTURE_2D_ARRAY,Ce,Be,ke[0].width,ke[0].height,Ee.depth);for(let Me=0,Ue=ke.length;Me<Ue;Me++)if(Pe=ke[Me],M.format!==Ii)if(Ne!==null)if(nt){if(Z)if(M.layerUpdates.size>0){const Ve=Xv(Pe.width,Pe.height,M.format,M.type);for(const Te of M.layerUpdates){const Je=Pe.data.subarray(Te*Ve/Pe.data.BYTES_PER_ELEMENT,(Te+1)*Ve/Pe.data.BYTES_PER_ELEMENT);i.compressedTexSubImage3D(o.TEXTURE_2D_ARRAY,Me,0,0,Te,Pe.width,Pe.height,1,Ne,Je)}}else i.compressedTexSubImage3D(o.TEXTURE_2D_ARRAY,Me,0,0,0,Pe.width,Pe.height,Ee.depth,Ne,Pe.data)}else i.compressedTexImage3D(o.TEXTURE_2D_ARRAY,Me,Be,Pe.width,Pe.height,Ee.depth,0,Pe.data,0,0);else ot("WebGLRenderer: Attempt to load unsupported compressed texture format in .uploadTexture()");else nt?Z&&i.texSubImage3D(o.TEXTURE_2D_ARRAY,Me,0,0,0,Pe.width,Pe.height,Ee.depth,Ne,$e,Pe.data):i.texImage3D(o.TEXTURE_2D_ARRAY,Me,Be,Pe.width,Pe.height,Ee.depth,0,Ne,$e,Pe.data);M.layerUpdates.size>0&&M.clearLayerUpdates()}else{nt&&lt&&i.texStorage2D(o.TEXTURE_2D,Ce,Be,ke[0].width,ke[0].height);for(let Me=0,Ue=ke.length;Me<Ue;Me++)Pe=ke[Me],M.format!==Ii?Ne!==null?nt?Z&&i.compressedTexSubImage2D(o.TEXTURE_2D,Me,0,0,Pe.width,Pe.height,Ne,Pe.data):i.compressedTexImage2D(o.TEXTURE_2D,Me,Be,Pe.width,Pe.height,0,Pe.data):ot("WebGLRenderer: Attempt to load unsupported compressed texture format in .uploadTexture()"):nt?Z&&i.texSubImage2D(o.TEXTURE_2D,Me,0,0,Pe.width,Pe.height,Ne,$e,Pe.data):i.texImage2D(o.TEXTURE_2D,Me,Be,Pe.width,Pe.height,0,Ne,$e,Pe.data)}else if(M.isDataArrayTexture)if(nt){if(lt&&i.texStorage3D(o.TEXTURE_2D_ARRAY,Ce,Be,Ee.width,Ee.height,Ee.depth),Z)if(M.layerUpdates.size>0){const Me=Xv(Ee.width,Ee.height,M.format,M.type);for(const Ue of M.layerUpdates){const Ve=Ee.data.subarray(Ue*Me/Ee.data.BYTES_PER_ELEMENT,(Ue+1)*Me/Ee.data.BYTES_PER_ELEMENT);i.texSubImage3D(o.TEXTURE_2D_ARRAY,0,0,0,Ue,Ee.width,Ee.height,1,Ne,$e,Ve)}M.clearLayerUpdates()}else i.texSubImage3D(o.TEXTURE_2D_ARRAY,0,0,0,0,Ee.width,Ee.height,Ee.depth,Ne,$e,Ee.data)}else i.texImage3D(o.TEXTURE_2D_ARRAY,0,Be,Ee.width,Ee.height,Ee.depth,0,Ne,$e,Ee.data);else if(M.isData3DTexture)nt?(lt&&i.texStorage3D(o.TEXTURE_3D,Ce,Be,Ee.width,Ee.height,Ee.depth),Z&&i.texSubImage3D(o.TEXTURE_3D,0,0,0,0,Ee.width,Ee.height,Ee.depth,Ne,$e,Ee.data)):i.texImage3D(o.TEXTURE_3D,0,Be,Ee.width,Ee.height,Ee.depth,0,Ne,$e,Ee.data);else if(M.isFramebufferTexture){if(lt)if(nt)i.texStorage2D(o.TEXTURE_2D,Ce,Be,Ee.width,Ee.height);else{let Me=Ee.width,Ue=Ee.height;for(let Ve=0;Ve<Ce;Ve++)i.texImage2D(o.TEXTURE_2D,Ve,Be,Me,Ue,0,Ne,$e,null),Me>>=1,Ue>>=1}}else if(M.isHTMLTexture){if("texElementImage2D"in o){const Me=o.canvas;if(Me.hasAttribute("layoutsubtree")||Me.setAttribute("layoutsubtree","true"),Ee.parentNode!==Me){Me.appendChild(Ee),_.add(M),Me.onpaint=Ue=>{const Ve=Ue.changedElements;for(const Te of _)Ve.includes(Te.image)&&(Te.needsUpdate=!0)},Me.requestPaint();return}if(o.texElementImage2D.length===3)o.texElementImage2D(o.TEXTURE_2D,o.RGBA8,Ee);else{const Ve=o.RGBA,Te=o.RGBA,Je=o.UNSIGNED_BYTE;o.texElementImage2D(o.TEXTURE_2D,0,Ve,Te,Je,Ee)}o.texParameteri(o.TEXTURE_2D,o.TEXTURE_MIN_FILTER,o.LINEAR),o.texParameteri(o.TEXTURE_2D,o.TEXTURE_WRAP_S,o.CLAMP_TO_EDGE),o.texParameteri(o.TEXTURE_2D,o.TEXTURE_WRAP_T,o.CLAMP_TO_EDGE)}}else if(ke.length>0){if(nt&&lt){const Me=Ot(ke[0]);i.texStorage2D(o.TEXTURE_2D,Ce,Be,Me.width,Me.height)}for(let Me=0,Ue=ke.length;Me<Ue;Me++)Pe=ke[Me],nt?Z&&i.texSubImage2D(o.TEXTURE_2D,Me,0,0,Ne,$e,Pe):i.texImage2D(o.TEXTURE_2D,Me,Be,Ne,$e,Pe);M.generateMipmaps=!1}else if(nt){if(lt){const Me=Ot(Ee);i.texStorage2D(o.TEXTURE_2D,Ce,Be,Me.width,Me.height)}Z&&i.texSubImage2D(o.TEXTURE_2D,0,0,0,Ne,$e,Ee)}else i.texImage2D(o.TEXTURE_2D,0,Be,Ne,$e,Ee);S(M)&&z(le),De.__version=Re.version,M.onUpdate&&M.onUpdate(M)}P.__version=M.version}function Ie(P,M,ie){if(M.image.length!==6)return;const le=ge(P,M),_e=M.source;i.bindTexture(o.TEXTURE_CUBE_MAP,P.__webglTexture,o.TEXTURE0+ie);const Re=r.get(_e);if(_e.version!==Re.__version||le===!0){i.activeTexture(o.TEXTURE0+ie);const De=Ct.getPrimaries(Ct.workingColorSpace),ve=M.colorSpace===vs?null:Ct.getPrimaries(M.colorSpace),Ee=M.colorSpace===vs||De===ve?o.NONE:o.BROWSER_DEFAULT_WEBGL;i.pixelStorei(o.UNPACK_FLIP_Y_WEBGL,M.flipY),i.pixelStorei(o.UNPACK_PREMULTIPLY_ALPHA_WEBGL,M.premultiplyAlpha),i.pixelStorei(o.UNPACK_ALIGNMENT,M.unpackAlignment),i.pixelStorei(o.UNPACK_COLORSPACE_CONVERSION_WEBGL,Ee);const Ne=M.isCompressedTexture||M.image[0].isCompressedTexture,$e=M.image[0]&&M.image[0].isDataTexture,Be=[];for(let Te=0;Te<6;Te++)!Ne&&!$e?Be[Te]=y(M.image[Te],!0,c.maxCubemapSize):Be[Te]=$e?M.image[Te].image:M.image[Te],Be[Te]=nn(M,Be[Te]);const Pe=Be[0],ke=f.convert(M.format,M.colorSpace),nt=f.convert(M.type),lt=C(M.internalFormat,ke,nt,M.normalized,M.colorSpace),Z=M.isVideoTexture!==!0,Ce=Re.__version===void 0||le===!0,Me=_e.dataReady;let Ue=U(M,Pe);F(o.TEXTURE_CUBE_MAP,M);let Ve;if(Ne){Z&&Ce&&i.texStorage2D(o.TEXTURE_CUBE_MAP,Ue,lt,Pe.width,Pe.height);for(let Te=0;Te<6;Te++){Ve=Be[Te].mipmaps;for(let Je=0;Je<Ve.length;Je++){const Ge=Ve[Je];M.format!==Ii?ke!==null?Z?Me&&i.compressedTexSubImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Te,Je,0,0,Ge.width,Ge.height,ke,Ge.data):i.compressedTexImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Te,Je,lt,Ge.width,Ge.height,0,Ge.data):ot("WebGLRenderer: Attempt to load unsupported compressed texture format in .setTextureCube()"):Z?Me&&i.texSubImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Te,Je,0,0,Ge.width,Ge.height,ke,nt,Ge.data):i.texImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Te,Je,lt,Ge.width,Ge.height,0,ke,nt,Ge.data)}}}else{if(Ve=M.mipmaps,Z&&Ce){Ve.length>0&&Ue++;const Te=Ot(Be[0]);i.texStorage2D(o.TEXTURE_CUBE_MAP,Ue,lt,Te.width,Te.height)}for(let Te=0;Te<6;Te++)if($e){Z?Me&&i.texSubImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Te,0,0,0,Be[Te].width,Be[Te].height,ke,nt,Be[Te].data):i.texImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Te,0,lt,Be[Te].width,Be[Te].height,0,ke,nt,Be[Te].data);for(let Je=0;Je<Ve.length;Je++){const Rt=Ve[Je].image[Te].image;Z?Me&&i.texSubImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Te,Je+1,0,0,Rt.width,Rt.height,ke,nt,Rt.data):i.texImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Te,Je+1,lt,Rt.width,Rt.height,0,ke,nt,Rt.data)}}else{Z?Me&&i.texSubImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Te,0,0,0,ke,nt,Be[Te]):i.texImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Te,0,lt,ke,nt,Be[Te]);for(let Je=0;Je<Ve.length;Je++){const Ge=Ve[Je];Z?Me&&i.texSubImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Te,Je+1,0,0,ke,nt,Ge.image[Te]):i.texImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Te,Je+1,lt,ke,nt,Ge.image[Te])}}}S(M)&&z(o.TEXTURE_CUBE_MAP),Re.__version=_e.version,M.onUpdate&&M.onUpdate(M)}P.__version=M.version}function xe(P,M,ie,le,_e,Re){const De=f.convert(ie.format,ie.colorSpace),ve=f.convert(ie.type),Ee=C(ie.internalFormat,De,ve,ie.normalized,ie.colorSpace),Ne=r.get(M),$e=r.get(ie);if($e.__renderTarget=M,!Ne.__hasExternalTextures){const Be=Math.max(1,M.width>>Re),Pe=Math.max(1,M.height>>Re);_e===o.TEXTURE_3D||_e===o.TEXTURE_2D_ARRAY?i.texImage3D(_e,Re,Ee,Be,Pe,M.depth,0,De,ve,null):i.texImage2D(_e,Re,Ee,Be,Pe,0,De,ve,null)}i.bindFramebuffer(o.FRAMEBUFFER,P),en(M)?h.framebufferTexture2DMultisampleEXT(o.FRAMEBUFFER,le,_e,$e.__webglTexture,0,Pt(M)):(_e===o.TEXTURE_2D||_e>=o.TEXTURE_CUBE_MAP_POSITIVE_X&&_e<=o.TEXTURE_CUBE_MAP_NEGATIVE_Z)&&o.framebufferTexture2D(o.FRAMEBUFFER,le,_e,$e.__webglTexture,Re),i.bindFramebuffer(o.FRAMEBUFFER,null)}function we(P,M,ie){if(o.bindRenderbuffer(o.RENDERBUFFER,P),M.depthBuffer){const le=M.depthTexture,_e=le&&le.isDepthTexture?le.type:null,Re=D(M.stencilBuffer,_e),De=M.stencilBuffer?o.DEPTH_STENCIL_ATTACHMENT:o.DEPTH_ATTACHMENT;en(M)?h.renderbufferStorageMultisampleEXT(o.RENDERBUFFER,Pt(M),Re,M.width,M.height):ie?o.renderbufferStorageMultisample(o.RENDERBUFFER,Pt(M),Re,M.width,M.height):o.renderbufferStorage(o.RENDERBUFFER,Re,M.width,M.height),o.framebufferRenderbuffer(o.FRAMEBUFFER,De,o.RENDERBUFFER,P)}else{const le=M.textures;for(let _e=0;_e<le.length;_e++){const Re=le[_e],De=f.convert(Re.format,Re.colorSpace),ve=f.convert(Re.type),Ee=C(Re.internalFormat,De,ve,Re.normalized,Re.colorSpace);en(M)?h.renderbufferStorageMultisampleEXT(o.RENDERBUFFER,Pt(M),Ee,M.width,M.height):ie?o.renderbufferStorageMultisample(o.RENDERBUFFER,Pt(M),Ee,M.width,M.height):o.renderbufferStorage(o.RENDERBUFFER,Ee,M.width,M.height)}}o.bindRenderbuffer(o.RENDERBUFFER,null)}function Vt(P,M,ie){const le=M.isWebGLCubeRenderTarget===!0;if(i.bindFramebuffer(o.FRAMEBUFFER,P),!(M.depthTexture&&M.depthTexture.isDepthTexture))throw new Error("THREE.WebGLTextures: renderTarget.depthTexture must be an instance of THREE.DepthTexture.");const _e=r.get(M.depthTexture);if(_e.__renderTarget=M,(!_e.__webglTexture||M.depthTexture.image.width!==M.width||M.depthTexture.image.height!==M.height)&&(M.depthTexture.image.width=M.width,M.depthTexture.image.height=M.height,M.depthTexture.needsUpdate=!0),le){if(_e.__webglInit===void 0&&(_e.__webglInit=!0,M.depthTexture.addEventListener("dispose",I)),_e.__webglTexture===void 0){_e.__webglTexture=o.createTexture(),i.bindTexture(o.TEXTURE_CUBE_MAP,_e.__webglTexture),F(o.TEXTURE_CUBE_MAP,M.depthTexture);const Ne=f.convert(M.depthTexture.format),$e=f.convert(M.depthTexture.type);let Be;M.depthTexture.format===Ia?Be=o.DEPTH_COMPONENT24:M.depthTexture.format===Zs&&(Be=o.DEPTH24_STENCIL8);for(let Pe=0;Pe<6;Pe++)o.texImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Pe,0,Be,M.width,M.height,0,Ne,$e,null)}}else pe(M.depthTexture,0);const Re=_e.__webglTexture,De=Pt(M),ve=le?o.TEXTURE_CUBE_MAP_POSITIVE_X+ie:o.TEXTURE_2D,Ee=M.depthTexture.format===Zs?o.DEPTH_STENCIL_ATTACHMENT:o.DEPTH_ATTACHMENT;if(M.depthTexture.format===Ia)en(M)?h.framebufferTexture2DMultisampleEXT(o.FRAMEBUFFER,Ee,ve,Re,0,De):o.framebufferTexture2D(o.FRAMEBUFFER,Ee,ve,Re,0);else if(M.depthTexture.format===Zs)en(M)?h.framebufferTexture2DMultisampleEXT(o.FRAMEBUFFER,Ee,ve,Re,0,De):o.framebufferTexture2D(o.FRAMEBUFFER,Ee,ve,Re,0);else throw new Error("THREE.WebGLTextures: Unknown depthTexture format.")}function pt(P){const M=r.get(P),ie=P.isWebGLCubeRenderTarget===!0;if(M.__boundDepthTexture!==P.depthTexture){const le=P.depthTexture;if(M.__depthDisposeCallback&&M.__depthDisposeCallback(),le){const _e=()=>{delete M.__boundDepthTexture,delete M.__depthDisposeCallback,le.removeEventListener("dispose",_e)};le.addEventListener("dispose",_e),M.__depthDisposeCallback=_e}M.__boundDepthTexture=le}if(P.depthTexture&&!M.__autoAllocateDepthBuffer)if(ie)for(let le=0;le<6;le++)Vt(M.__webglFramebuffer[le],P,le);else{const le=P.texture.mipmaps;le&&le.length>0?Vt(M.__webglFramebuffer[0],P,0):Vt(M.__webglFramebuffer,P,0)}else if(ie){M.__webglDepthbuffer=[];for(let le=0;le<6;le++)if(i.bindFramebuffer(o.FRAMEBUFFER,M.__webglFramebuffer[le]),M.__webglDepthbuffer[le]===void 0)M.__webglDepthbuffer[le]=o.createRenderbuffer(),we(M.__webglDepthbuffer[le],P,!1);else{const _e=P.stencilBuffer?o.DEPTH_STENCIL_ATTACHMENT:o.DEPTH_ATTACHMENT,Re=M.__webglDepthbuffer[le];o.bindRenderbuffer(o.RENDERBUFFER,Re),o.framebufferRenderbuffer(o.FRAMEBUFFER,_e,o.RENDERBUFFER,Re)}}else{const le=P.texture.mipmaps;if(le&&le.length>0?i.bindFramebuffer(o.FRAMEBUFFER,M.__webglFramebuffer[0]):i.bindFramebuffer(o.FRAMEBUFFER,M.__webglFramebuffer),M.__webglDepthbuffer===void 0)M.__webglDepthbuffer=o.createRenderbuffer(),we(M.__webglDepthbuffer,P,!1);else{const _e=P.stencilBuffer?o.DEPTH_STENCIL_ATTACHMENT:o.DEPTH_ATTACHMENT,Re=M.__webglDepthbuffer;o.bindRenderbuffer(o.RENDERBUFFER,Re),o.framebufferRenderbuffer(o.FRAMEBUFFER,_e,o.RENDERBUFFER,Re)}}i.bindFramebuffer(o.FRAMEBUFFER,null)}function gt(P,M,ie){const le=r.get(P);M!==void 0&&xe(le.__webglFramebuffer,P,P.texture,o.COLOR_ATTACHMENT0,o.TEXTURE_2D,0),ie!==void 0&&pt(P)}function St(P){const M=P.texture,ie=r.get(P),le=r.get(M);P.addEventListener("dispose",b);const _e=P.textures,Re=P.isWebGLCubeRenderTarget===!0,De=_e.length>1;if(De||(le.__webglTexture===void 0&&(le.__webglTexture=o.createTexture()),le.__version=M.version,d.memory.textures++),Re){ie.__webglFramebuffer=[];for(let ve=0;ve<6;ve++)if(M.mipmaps&&M.mipmaps.length>0){ie.__webglFramebuffer[ve]=[];for(let Ee=0;Ee<M.mipmaps.length;Ee++)ie.__webglFramebuffer[ve][Ee]=o.createFramebuffer()}else ie.__webglFramebuffer[ve]=o.createFramebuffer()}else{if(M.mipmaps&&M.mipmaps.length>0){ie.__webglFramebuffer=[];for(let ve=0;ve<M.mipmaps.length;ve++)ie.__webglFramebuffer[ve]=o.createFramebuffer()}else ie.__webglFramebuffer=o.createFramebuffer();if(De)for(let ve=0,Ee=_e.length;ve<Ee;ve++){const Ne=r.get(_e[ve]);Ne.__webglTexture===void 0&&(Ne.__webglTexture=o.createTexture(),d.memory.textures++)}if(P.samples>0&&en(P)===!1){ie.__webglMultisampledFramebuffer=o.createFramebuffer(),ie.__webglColorRenderbuffer=[],i.bindFramebuffer(o.FRAMEBUFFER,ie.__webglMultisampledFramebuffer);for(let ve=0;ve<_e.length;ve++){const Ee=_e[ve];ie.__webglColorRenderbuffer[ve]=o.createRenderbuffer(),o.bindRenderbuffer(o.RENDERBUFFER,ie.__webglColorRenderbuffer[ve]);const Ne=f.convert(Ee.format,Ee.colorSpace),$e=f.convert(Ee.type),Be=C(Ee.internalFormat,Ne,$e,Ee.normalized,Ee.colorSpace,P.isXRRenderTarget===!0),Pe=Pt(P);o.renderbufferStorageMultisample(o.RENDERBUFFER,Pe,Be,P.width,P.height),o.framebufferRenderbuffer(o.FRAMEBUFFER,o.COLOR_ATTACHMENT0+ve,o.RENDERBUFFER,ie.__webglColorRenderbuffer[ve])}o.bindRenderbuffer(o.RENDERBUFFER,null),P.depthBuffer&&(ie.__webglDepthRenderbuffer=o.createRenderbuffer(),we(ie.__webglDepthRenderbuffer,P,!0)),i.bindFramebuffer(o.FRAMEBUFFER,null)}}if(Re){i.bindTexture(o.TEXTURE_CUBE_MAP,le.__webglTexture),F(o.TEXTURE_CUBE_MAP,M);for(let ve=0;ve<6;ve++)if(M.mipmaps&&M.mipmaps.length>0)for(let Ee=0;Ee<M.mipmaps.length;Ee++)xe(ie.__webglFramebuffer[ve][Ee],P,M,o.COLOR_ATTACHMENT0,o.TEXTURE_CUBE_MAP_POSITIVE_X+ve,Ee);else xe(ie.__webglFramebuffer[ve],P,M,o.COLOR_ATTACHMENT0,o.TEXTURE_CUBE_MAP_POSITIVE_X+ve,0);S(M)&&z(o.TEXTURE_CUBE_MAP),i.unbindTexture()}else if(De){for(let ve=0,Ee=_e.length;ve<Ee;ve++){const Ne=_e[ve],$e=r.get(Ne);let Be=o.TEXTURE_2D;(P.isWebGL3DRenderTarget||P.isWebGLArrayRenderTarget)&&(Be=P.isWebGL3DRenderTarget?o.TEXTURE_3D:o.TEXTURE_2D_ARRAY),i.bindTexture(Be,$e.__webglTexture),F(Be,Ne),xe(ie.__webglFramebuffer,P,Ne,o.COLOR_ATTACHMENT0+ve,Be,0),S(Ne)&&z(Be)}i.unbindTexture()}else{let ve=o.TEXTURE_2D;if((P.isWebGL3DRenderTarget||P.isWebGLArrayRenderTarget)&&(ve=P.isWebGL3DRenderTarget?o.TEXTURE_3D:o.TEXTURE_2D_ARRAY),i.bindTexture(ve,le.__webglTexture),F(ve,M),M.mipmaps&&M.mipmaps.length>0)for(let Ee=0;Ee<M.mipmaps.length;Ee++)xe(ie.__webglFramebuffer[Ee],P,M,o.COLOR_ATTACHMENT0,ve,Ee);else xe(ie.__webglFramebuffer,P,M,o.COLOR_ATTACHMENT0,ve,0);S(M)&&z(ve),i.unbindTexture()}P.depthBuffer&&pt(P)}function tt(P){const M=P.textures;for(let ie=0,le=M.length;ie<le;ie++){const _e=M[ie];if(S(_e)){const Re=W(P),De=r.get(_e).__webglTexture;i.bindTexture(Re,De),z(Re),i.unbindTexture()}}}const it=[],kt=[];function dn(P){if(P.samples>0){if(en(P)===!1){const M=P.textures,ie=P.width,le=P.height;let _e=o.COLOR_BUFFER_BIT;const Re=P.stencilBuffer?o.DEPTH_STENCIL_ATTACHMENT:o.DEPTH_ATTACHMENT,De=r.get(P),ve=M.length>1;if(ve)for(let Ne=0;Ne<M.length;Ne++)i.bindFramebuffer(o.FRAMEBUFFER,De.__webglMultisampledFramebuffer),o.framebufferRenderbuffer(o.FRAMEBUFFER,o.COLOR_ATTACHMENT0+Ne,o.RENDERBUFFER,null),i.bindFramebuffer(o.FRAMEBUFFER,De.__webglFramebuffer),o.framebufferTexture2D(o.DRAW_FRAMEBUFFER,o.COLOR_ATTACHMENT0+Ne,o.TEXTURE_2D,null,0);i.bindFramebuffer(o.READ_FRAMEBUFFER,De.__webglMultisampledFramebuffer);const Ee=P.texture.mipmaps;Ee&&Ee.length>0?i.bindFramebuffer(o.DRAW_FRAMEBUFFER,De.__webglFramebuffer[0]):i.bindFramebuffer(o.DRAW_FRAMEBUFFER,De.__webglFramebuffer);for(let Ne=0;Ne<M.length;Ne++){if(P.resolveDepthBuffer&&(P.depthBuffer&&(_e|=o.DEPTH_BUFFER_BIT),P.stencilBuffer&&P.resolveStencilBuffer&&(_e|=o.STENCIL_BUFFER_BIT)),ve){o.framebufferRenderbuffer(o.READ_FRAMEBUFFER,o.COLOR_ATTACHMENT0,o.RENDERBUFFER,De.__webglColorRenderbuffer[Ne]);const $e=r.get(M[Ne]).__webglTexture;o.framebufferTexture2D(o.DRAW_FRAMEBUFFER,o.COLOR_ATTACHMENT0,o.TEXTURE_2D,$e,0)}o.blitFramebuffer(0,0,ie,le,0,0,ie,le,_e,o.NEAREST),m===!0&&(it.length=0,kt.length=0,it.push(o.COLOR_ATTACHMENT0+Ne),P.depthBuffer&&P.storeMultisampledDepthBuffer===!1&&(it.push(Re),kt.push(Re),o.invalidateFramebuffer(o.DRAW_FRAMEBUFFER,kt)),o.invalidateFramebuffer(o.READ_FRAMEBUFFER,it))}if(i.bindFramebuffer(o.READ_FRAMEBUFFER,null),i.bindFramebuffer(o.DRAW_FRAMEBUFFER,null),ve)for(let Ne=0;Ne<M.length;Ne++){i.bindFramebuffer(o.FRAMEBUFFER,De.__webglMultisampledFramebuffer),o.framebufferRenderbuffer(o.FRAMEBUFFER,o.COLOR_ATTACHMENT0+Ne,o.RENDERBUFFER,De.__webglColorRenderbuffer[Ne]);const $e=r.get(M[Ne]).__webglTexture;i.bindFramebuffer(o.FRAMEBUFFER,De.__webglFramebuffer),o.framebufferTexture2D(o.DRAW_FRAMEBUFFER,o.COLOR_ATTACHMENT0+Ne,o.TEXTURE_2D,$e,0)}i.bindFramebuffer(o.DRAW_FRAMEBUFFER,De.__webglMultisampledFramebuffer)}else if(P.depthBuffer&&P.storeMultisampledDepthBuffer===!1&&m){const M=P.stencilBuffer?o.DEPTH_STENCIL_ATTACHMENT:o.DEPTH_ATTACHMENT;o.invalidateFramebuffer(o.DRAW_FRAMEBUFFER,[M])}}}function Pt(P){return Math.min(c.maxSamples,P.samples)}function en(P){const M=r.get(P);return P.samples>0&&t.has("WEBGL_multisampled_render_to_texture")===!0&&M.__useRenderToTexture!==!1}function K(P){const M=d.render.frame;x.get(P)!==M&&(x.set(P,M),P.update())}function nn(P,M){const ie=P.colorSpace,le=P.format,_e=P.type;return P.isCompressedTexture===!0||P.isVideoTexture===!0||ie!==Au&&ie!==vs&&(Ct.getTransfer(ie)===Yt?(le!==Ii||_e!==fi)&&ot("WebGLTextures: sRGB encoded textures have to use RGBAFormat and UnsignedByteType."):It("WebGLTextures: Unsupported texture color space:",ie)),M}function Ot(P){return typeof HTMLImageElement<"u"&&P instanceof HTMLImageElement?(g.width=P.naturalWidth||P.width,g.height=P.naturalHeight||P.height):typeof VideoFrame<"u"&&P instanceof VideoFrame?(g.width=P.displayWidth,g.height=P.displayHeight):(g.width=P.width,g.height=P.height),g}this.allocateTextureUnit=j,this.resetTextureUnits=O,this.getTextureUnits=q,this.setTextureUnits=ne,this.setTexture2D=pe,this.setTexture2DArray=ce,this.setTexture3D=me,this.setTextureCube=be,this.rebindTextures=gt,this.setupRenderTarget=St,this.updateRenderTargetMipmap=tt,this.updateMultisampleRenderTarget=dn,this.setupDepthRenderbuffer=pt,this.setupFrameBufferTexture=xe,this.useMultisampledRTT=en,this.isReversedDepthBuffer=function(){return i.buffers.depth.getReversed()}}function cw(o,t){function i(r,c=vs){let f;const d=Ct.getTransfer(c);if(r===fi)return o.UNSIGNED_BYTE;if(r===Vp)return o.UNSIGNED_SHORT_4_4_4_4;if(r===kp)return o.UNSIGNED_SHORT_5_5_5_1;if(r===Ux)return o.UNSIGNED_INT_5_9_9_9_REV;if(r===Lx)return o.UNSIGNED_INT_10F_11F_11F_REV;if(r===Nx)return o.BYTE;if(r===Dx)return o.SHORT;if(r===Sl)return o.UNSIGNED_SHORT;if(r===Gp)return o.INT;if(r===ra)return o.UNSIGNED_INT;if(r===ia)return o.FLOAT;if(r===oa)return o.HALF_FLOAT;if(r===Ox)return o.ALPHA;if(r===Px)return o.RGB;if(r===Ii)return o.RGBA;if(r===Ia)return o.DEPTH_COMPONENT;if(r===Zs)return o.DEPTH_STENCIL;if(r===Ix)return o.RED;if(r===Xp)return o.RED_INTEGER;if(r===Js)return o.RG;if(r===Wp)return o.RG_INTEGER;if(r===qp)return o.RGBA_INTEGER;if(r===xu||r===Su||r===yu||r===Mu)if(d===Yt)if(f=t.get("WEBGL_compressed_texture_s3tc_srgb"),f!==null){if(r===xu)return f.COMPRESSED_SRGB_S3TC_DXT1_EXT;if(r===Su)return f.COMPRESSED_SRGB_ALPHA_S3TC_DXT1_EXT;if(r===yu)return f.COMPRESSED_SRGB_ALPHA_S3TC_DXT3_EXT;if(r===Mu)return f.COMPRESSED_SRGB_ALPHA_S3TC_DXT5_EXT}else return null;else if(f=t.get("WEBGL_compressed_texture_s3tc"),f!==null){if(r===xu)return f.COMPRESSED_RGB_S3TC_DXT1_EXT;if(r===Su)return f.COMPRESSED_RGBA_S3TC_DXT1_EXT;if(r===yu)return f.COMPRESSED_RGBA_S3TC_DXT3_EXT;if(r===Mu)return f.COMPRESSED_RGBA_S3TC_DXT5_EXT}else return null;if(r===ap||r===sp||r===rp||r===op)if(f=t.get("WEBGL_compressed_texture_pvrtc"),f!==null){if(r===ap)return f.COMPRESSED_RGB_PVRTC_4BPPV1_IMG;if(r===sp)return f.COMPRESSED_RGB_PVRTC_2BPPV1_IMG;if(r===rp)return f.COMPRESSED_RGBA_PVRTC_4BPPV1_IMG;if(r===op)return f.COMPRESSED_RGBA_PVRTC_2BPPV1_IMG}else return null;if(r===lp||r===cp||r===up||r===fp||r===dp||r===bu||r===hp)if(f=t.get("WEBGL_compressed_texture_etc"),f!==null){if(r===lp||r===cp)return d===Yt?f.COMPRESSED_SRGB8_ETC2:f.COMPRESSED_RGB8_ETC2;if(r===up)return d===Yt?f.COMPRESSED_SRGB8_ALPHA8_ETC2_EAC:f.COMPRESSED_RGBA8_ETC2_EAC;if(r===fp)return f.COMPRESSED_R11_EAC;if(r===dp)return f.COMPRESSED_SIGNED_R11_EAC;if(r===bu)return f.COMPRESSED_RG11_EAC;if(r===hp)return f.COMPRESSED_SIGNED_RG11_EAC}else return null;if(r===pp||r===mp||r===gp||r===_p||r===vp||r===xp||r===Sp||r===yp||r===Mp||r===Ep||r===bp||r===Tp||r===Ap||r===Rp)if(f=t.get("WEBGL_compressed_texture_astc"),f!==null){if(r===pp)return d===Yt?f.COMPRESSED_SRGB8_ALPHA8_ASTC_4x4_KHR:f.COMPRESSED_RGBA_ASTC_4x4_KHR;if(r===mp)return d===Yt?f.COMPRESSED_SRGB8_ALPHA8_ASTC_5x4_KHR:f.COMPRESSED_RGBA_ASTC_5x4_KHR;if(r===gp)return d===Yt?f.COMPRESSED_SRGB8_ALPHA8_ASTC_5x5_KHR:f.COMPRESSED_RGBA_ASTC_5x5_KHR;if(r===_p)return d===Yt?f.COMPRESSED_SRGB8_ALPHA8_ASTC_6x5_KHR:f.COMPRESSED_RGBA_ASTC_6x5_KHR;if(r===vp)return d===Yt?f.COMPRESSED_SRGB8_ALPHA8_ASTC_6x6_KHR:f.COMPRESSED_RGBA_ASTC_6x6_KHR;if(r===xp)return d===Yt?f.COMPRESSED_SRGB8_ALPHA8_ASTC_8x5_KHR:f.COMPRESSED_RGBA_ASTC_8x5_KHR;if(r===Sp)return d===Yt?f.COMPRESSED_SRGB8_ALPHA8_ASTC_8x6_KHR:f.COMPRESSED_RGBA_ASTC_8x6_KHR;if(r===yp)return d===Yt?f.COMPRESSED_SRGB8_ALPHA8_ASTC_8x8_KHR:f.COMPRESSED_RGBA_ASTC_8x8_KHR;if(r===Mp)return d===Yt?f.COMPRESSED_SRGB8_ALPHA8_ASTC_10x5_KHR:f.COMPRESSED_RGBA_ASTC_10x5_KHR;if(r===Ep)return d===Yt?f.COMPRESSED_SRGB8_ALPHA8_ASTC_10x6_KHR:f.COMPRESSED_RGBA_ASTC_10x6_KHR;if(r===bp)return d===Yt?f.COMPRESSED_SRGB8_ALPHA8_ASTC_10x8_KHR:f.COMPRESSED_RGBA_ASTC_10x8_KHR;if(r===Tp)return d===Yt?f.COMPRESSED_SRGB8_ALPHA8_ASTC_10x10_KHR:f.COMPRESSED_RGBA_ASTC_10x10_KHR;if(r===Ap)return d===Yt?f.COMPRESSED_SRGB8_ALPHA8_ASTC_12x10_KHR:f.COMPRESSED_RGBA_ASTC_12x10_KHR;if(r===Rp)return d===Yt?f.COMPRESSED_SRGB8_ALPHA8_ASTC_12x12_KHR:f.COMPRESSED_RGBA_ASTC_12x12_KHR}else return null;if(r===wp||r===Cp||r===Np)if(f=t.get("EXT_texture_compression_bptc"),f!==null){if(r===wp)return d===Yt?f.COMPRESSED_SRGB_ALPHA_BPTC_UNORM_EXT:f.COMPRESSED_RGBA_BPTC_UNORM_EXT;if(r===Cp)return f.COMPRESSED_RGB_BPTC_SIGNED_FLOAT_EXT;if(r===Np)return f.COMPRESSED_RGB_BPTC_UNSIGNED_FLOAT_EXT}else return null;if(r===Dp||r===Up||r===Tu||r===Lp)if(f=t.get("EXT_texture_compression_rgtc"),f!==null){if(r===Dp)return f.COMPRESSED_RED_RGTC1_EXT;if(r===Up)return f.COMPRESSED_SIGNED_RED_RGTC1_EXT;if(r===Tu)return f.COMPRESSED_RED_GREEN_RGTC2_EXT;if(r===Lp)return f.COMPRESSED_SIGNED_RED_GREEN_RGTC2_EXT}else return null;return r===yl?o.UNSIGNED_INT_24_8:o[r]!==void 0?o[r]:null}return{convert:i}}const uw=`
void main() {

	gl_Position = vec4( position, 1.0 );

}`,fw=`
uniform sampler2DArray depthColor;
uniform float depthWidth;
uniform float depthHeight;

void main() {

	vec2 coord = vec2( gl_FragCoord.x / depthWidth, gl_FragCoord.y / depthHeight );

	if ( coord.x >= 1.0 ) {

		gl_FragDepth = texture( depthColor, vec3( coord.x - 1.0, coord.y, 1 ) ).r;

	} else {

		gl_FragDepth = texture( depthColor, vec3( coord.x, coord.y, 0 ) ).r;

	}

}`;class dw{constructor(){this.texture=null,this.mesh=null,this.depthNear=0,this.depthFar=0}init(t,i){if(this.texture===null){const r=new Wx(t.texture);(t.depthNear!==i.depthNear||t.depthFar!==i.depthFar)&&(this.depthNear=t.depthNear,this.depthFar=t.depthFar),this.texture=r}}getMesh(t){if(this.texture!==null&&this.mesh===null){const i=t.cameras[0].viewport,r=new ca({vertexShader:uw,fragmentShader:fw,uniforms:{depthColor:{value:this.texture},depthWidth:{value:i.z},depthHeight:{value:i.w}}});this.mesh=new la(new Uu(20,20),r)}return this.mesh}reset(){this.texture=null,this.mesh=null}getDepthTexture(){return this.texture}}class hw extends $s{constructor(t,i){super();const r=this;let c=null,f=1,d=null,h="local-floor",m=1,g=null,x=null,_=null,v=null,E=null,R=null;const N=typeof XRWebGLBinding<"u",y=new dw,S={},z=i.getContextAttributes();let W=null,C=null;const D=[],U=[],I=new Ut;let b=null,L=null;const k=new ui;k.viewport=new sn;const Y=new ui;Y.viewport=new sn;const $=[k,Y],O=new yb;let q=null,ne=null;this.cameraAutoUpdate=!0,this.enabled=!1,this.isPresenting=!1,this.getController=function(X){let ue=D[X];return ue===void 0&&(ue=new Th,D[X]=ue),ue.getTargetRaySpace()},this.getControllerGrip=function(X){let ue=D[X];return ue===void 0&&(ue=new Th,D[X]=ue),ue.getGripSpace()},this.getHand=function(X){let ue=D[X];return ue===void 0&&(ue=new Th,D[X]=ue),ue.getHandSpace()};function j(X){const ue=U.indexOf(X.inputSource);if(ue===-1)return;const Ae=D[ue];Ae!==void 0&&(Ae.update(X.inputSource,X.frame,g||d),Ae.dispatchEvent({type:X.type,data:X.inputSource}))}function Q(){c.removeEventListener("select",j),c.removeEventListener("selectstart",j),c.removeEventListener("selectend",j),c.removeEventListener("squeeze",j),c.removeEventListener("squeezestart",j),c.removeEventListener("squeezeend",j),c.removeEventListener("end",Q),c.removeEventListener("inputsourceschange",pe);for(let X=0;X<D.length;X++){const ue=U[X];ue!==null&&(U[X]=null,D[X].disconnect(ue))}q=null,ne=null,y.reset();for(const X in S)delete S[X];if(t.setRenderTarget(W),E=null,v=null,_=null,c=null,C=null,ge.stop(),r.isPresenting=!1,t.setPixelRatio(b),t.setSize(I.width,I.height,!1),L!==null){const X=L.camera;X.fov=L.fov,X.zoom=L.zoom,X.updateProjectionMatrix(),L=null}r.dispatchEvent({type:"sessionend"})}this.setFramebufferScaleFactor=function(X){f=X,r.isPresenting===!0&&ot("WebXRManager: Cannot change framebuffer scale while presenting.")},this.setReferenceSpaceType=function(X){h=X,r.isPresenting===!0&&ot("WebXRManager: Cannot change reference space type while presenting.")},this.getReferenceSpace=function(){return g||d},this.setReferenceSpace=function(X){g=X},this.getBaseLayer=function(){return v!==null?v:E},this.getBinding=function(){return _===null&&N&&(_=new XRWebGLBinding(c,i)),_},this.getFrame=function(){return R},this.getSession=function(){return c},this.setSession=async function(X){if(c=X,c!==null){if(W=t.getRenderTarget(),c.addEventListener("select",j),c.addEventListener("selectstart",j),c.addEventListener("selectend",j),c.addEventListener("squeeze",j),c.addEventListener("squeezestart",j),c.addEventListener("squeezeend",j),c.addEventListener("end",Q),c.addEventListener("inputsourceschange",pe),z.xrCompatible!==!0&&await i.makeXRCompatible(),b=t.getPixelRatio(),t.getSize(I),N&&"createProjectionLayer"in XRWebGLBinding.prototype){let Ae=null,Ie=null,xe=null;z.depth&&(xe=z.stencil?i.DEPTH24_STENCIL8:i.DEPTH_COMPONENT24,Ae=z.stencil?Zs:Ia,Ie=z.stencil?yl:ra);const we={colorFormat:i.RGBA8,depthFormat:xe,scaleFactor:f};_=this.getBinding(),v=_.createProjectionLayer(we),c.updateRenderState({layers:[v]}),t.setPixelRatio(1),t.setSize(v.textureWidth,v.textureHeight,!1),C=new Bi(v.textureWidth,v.textureHeight,{format:Ii,type:fi,depthTexture:new El(v.textureWidth,v.textureHeight,Ie,void 0,void 0,void 0,void 0,void 0,void 0,Ae),stencilBuffer:z.stencil,colorSpace:t.outputColorSpace,samples:z.antialias?4:0,resolveDepthBuffer:v.ignoreDepthValues===!1,resolveStencilBuffer:v.ignoreDepthValues===!1,storeMultisampledDepthBuffer:v.ignoreDepthValues===!1,storeMultisampledStencilBuffer:v.ignoreDepthValues===!1})}else{const Ae={antialias:z.antialias,alpha:!0,depth:z.depth,stencil:z.stencil,framebufferScaleFactor:f};E=new XRWebGLLayer(c,i,Ae),c.updateRenderState({baseLayer:E}),t.setPixelRatio(1),t.setSize(E.framebufferWidth,E.framebufferHeight,!1),C=new Bi(E.framebufferWidth,E.framebufferHeight,{format:Ii,type:fi,colorSpace:t.outputColorSpace,stencilBuffer:z.stencil,resolveDepthBuffer:E.ignoreDepthValues===!1,resolveStencilBuffer:E.ignoreDepthValues===!1,storeMultisampledDepthBuffer:E.ignoreDepthValues===!1,storeMultisampledStencilBuffer:E.ignoreDepthValues===!1})}C.isXRRenderTarget=!0,this.setFoveation(m),g=null,d=await c.requestReferenceSpace(h),ge.setContext(c),ge.start(),r.isPresenting=!0,r.dispatchEvent({type:"sessionstart"})}},this.getEnvironmentBlendMode=function(){if(c!==null)return c.environmentBlendMode},this.getDepthTexture=function(){return y.getDepthTexture()};function pe(X){for(let ue=0;ue<X.removed.length;ue++){const Ae=X.removed[ue],Ie=U.indexOf(Ae);Ie>=0&&(U[Ie]=null,D[Ie].disconnect(Ae))}for(let ue=0;ue<X.added.length;ue++){const Ae=X.added[ue];let Ie=U.indexOf(Ae);if(Ie===-1){for(let we=0;we<D.length;we++)if(we>=U.length){U.push(Ae),Ie=we;break}else if(U[we]===null){U[we]=Ae,Ie=we;break}if(Ie===-1)break}const xe=D[Ie];xe&&xe.connect(Ae)}}const ce=new he,me=new he;function be(X,ue,Ae){ce.setFromMatrixPosition(ue.matrixWorld),me.setFromMatrixPosition(Ae.matrixWorld);const Ie=ce.distanceTo(me),xe=ue.projectionMatrix.elements,we=Ae.projectionMatrix.elements,Vt=xe[14]/(xe[10]-1),pt=xe[14]/(xe[10]+1),gt=(xe[9]+1)/xe[5],St=(xe[9]-1)/xe[5],tt=(xe[8]-1)/xe[0],it=(we[8]+1)/we[0],kt=Vt*tt,dn=Vt*it,Pt=Ie/(-tt+it),en=Pt*-tt;if(ue.matrixWorld.decompose(X.position,X.quaternion,X.scale),X.translateX(en),X.translateZ(Pt),X.matrixWorld.compose(X.position,X.quaternion,X.scale),X.matrixWorldInverse.copy(X.matrixWorld).invert(),xe[10]===-1)X.projectionMatrix.copy(ue.projectionMatrix),X.projectionMatrixInverse.copy(ue.projectionMatrixInverse);else{const K=Vt+Pt,nn=pt+Pt,Ot=kt-en,P=dn+(Ie-en),M=gt*pt/nn*K,ie=St*pt/nn*K;X.projectionMatrix.makePerspective(Ot,P,M,ie,K,nn),X.projectionMatrixInverse.copy(X.projectionMatrix).invert()}}function Ze(X,ue){ue===null?X.matrixWorld.copy(X.matrix):X.matrixWorld.multiplyMatrices(ue.matrixWorld,X.matrix),X.matrixWorldInverse.copy(X.matrixWorld).invert()}this.updateCamera=function(X){if(c===null)return;let ue=X.near,Ae=X.far;y.texture!==null&&(y.depthNear>0&&(ue=y.depthNear),y.depthFar>0&&(Ae=y.depthFar)),O.near=Y.near=k.near=ue,O.far=Y.far=k.far=Ae,(q!==O.near||ne!==O.far)&&(c.updateRenderState({depthNear:O.near,depthFar:O.far}),q=O.near,ne=O.far),O.layers.mask=X.layers.mask|6,k.layers.mask=O.layers.mask&-5,Y.layers.mask=O.layers.mask&-3;const Ie=X.parent,xe=O.cameras;Ze(O,Ie);for(let we=0;we<xe.length;we++)Ze(xe[we],Ie);xe.length===2?be(O,k,Y):O.projectionMatrix.copy(k.projectionMatrix),L===null&&X.isPerspectiveCamera&&(L={camera:X,fov:X.fov,zoom:X.zoom}),je(X,O,Ie)};function je(X,ue,Ae){Ae===null?X.matrix.copy(ue.matrixWorld):(X.matrix.copy(Ae.matrixWorld),X.matrix.invert(),X.matrix.multiply(ue.matrixWorld)),X.matrix.decompose(X.position,X.quaternion,X.scale),X.updateMatrixWorld(!0),X.projectionMatrix.copy(ue.projectionMatrix),X.projectionMatrixInverse.copy(ue.projectionMatrixInverse),X.isPerspectiveCamera&&(X.fov=Pp*2*Math.atan(1/X.projectionMatrix.elements[5]),X.zoom=1)}this.getCamera=function(){return O},this.getFoveation=function(){if(!(v===null&&E===null))return m},this.setFoveation=function(X){m=X,v!==null&&(v.fixedFoveation=X),E!==null&&E.fixedFoveation!==void 0&&(E.fixedFoveation=X)},this.hasDepthSensing=function(){return y.texture!==null},this.getDepthSensingMesh=function(){return y.getMesh(O)},this.getCameraTexture=function(X){return S[X]};let H=null;function F(X,ue){if(x=ue.getViewerPose(g||d),R=ue,x!==null){const Ae=x.views;E!==null&&(t.setRenderTargetFramebuffer(C,E.framebuffer),t.setRenderTarget(C));let Ie=!1;Ae.length!==O.cameras.length&&(O.cameras.length=0,Ie=!0);for(let pt=0;pt<Ae.length;pt++){const gt=Ae[pt];let St=null;if(E!==null)St=E.getViewport(gt);else{const it=_.getViewSubImage(v,gt);St=it.viewport,pt===0&&(t.setRenderTargetTextures(C,it.colorTexture,it.depthStencilTexture),t.setRenderTarget(C))}let tt=$[pt];tt===void 0&&(tt=new ui,tt.layers.enable(pt),tt.viewport=new sn,$[pt]=tt),tt.matrix.fromArray(gt.transform.matrix),tt.matrix.decompose(tt.position,tt.quaternion,tt.scale),tt.projectionMatrix.fromArray(gt.projectionMatrix),tt.projectionMatrixInverse.copy(tt.projectionMatrix).invert(),tt.viewport.set(St.x,St.y,St.width,St.height),pt===0&&(O.matrix.copy(tt.matrix),O.matrix.decompose(O.position,O.quaternion,O.scale)),Ie===!0&&O.cameras.push(tt)}const xe=c.enabledFeatures;if(xe&&xe.includes("depth-sensing")&&c.depthUsage=="gpu-optimized"&&N){_=r.getBinding();const pt=_.getDepthInformation(Ae[0]);pt&&pt.isValid&&pt.texture&&y.init(pt,c.renderState)}if(xe&&xe.includes("camera-access")&&N){t.state.unbindTexture(),_=r.getBinding();for(let pt=0;pt<Ae.length;pt++){const gt=Ae[pt].camera;if(gt){let St=S[gt];St||(St=new Wx,S[gt]=St);const tt=_.getCameraImage(gt);St.sourceTexture=tt}}}}for(let Ae=0;Ae<D.length;Ae++){const Ie=U[Ae],xe=D[Ae];Ie!==null&&xe!==void 0&&xe.update(Ie,ue,g||d)}H&&H(X,ue),ue.detectedPlanes&&r.dispatchEvent({type:"planesdetected",data:ue}),R=null}const ge=new Kx;ge.setAnimationLoop(F),this.setAnimationLoop=function(X){H=X},this.dispose=function(){}}}const pw=new un,iS=new ft;iS.set(-1,0,0,0,1,0,0,0,1);function mw(o,t){function i(y,S){y.matrixAutoUpdate===!0&&y.updateMatrix(),S.value.copy(y.matrix)}function r(y,S){S.color.getRGB(y.fogColor.value,qx(o)),S.isFog?(y.fogNear.value=S.near,y.fogFar.value=S.far):S.isFogExp2&&(y.fogDensity.value=S.density)}function c(y,S,z,W,C){S.isNodeMaterial?S.uniformsNeedUpdate=!1:S.isMeshBasicMaterial?f(y,S):S.isMeshLambertMaterial?(f(y,S),S.envMap&&(y.envMapIntensity.value=S.envMapIntensity)):S.isMeshToonMaterial?(f(y,S),_(y,S)):S.isMeshPhongMaterial?(f(y,S),x(y,S),S.envMap&&(y.envMapIntensity.value=S.envMapIntensity)):S.isMeshStandardMaterial?(f(y,S),v(y,S),S.isMeshPhysicalMaterial&&E(y,S,C)):S.isMeshMatcapMaterial?(f(y,S),R(y,S)):S.isMeshDepthMaterial?f(y,S):S.isMeshDistanceMaterial?(f(y,S),N(y,S)):S.isMeshNormalMaterial?f(y,S):S.isLineBasicMaterial?(d(y,S),S.isLineDashedMaterial&&h(y,S)):S.isPointsMaterial?m(y,S,z,W):S.isSpriteMaterial?g(y,S):S.isShadowMaterial?(y.color.value.copy(S.color),y.opacity.value=S.opacity):S.isShaderMaterial&&(S.uniformsNeedUpdate=!1)}function f(y,S){y.opacity.value=S.opacity,S.color&&y.diffuse.value.copy(S.color),S.emissive&&y.emissive.value.copy(S.emissive).multiplyScalar(S.emissiveIntensity),S.map&&(y.map.value=S.map,i(S.map,y.mapTransform)),S.alphaMap&&(y.alphaMap.value=S.alphaMap,i(S.alphaMap,y.alphaMapTransform)),S.bumpMap&&(y.bumpMap.value=S.bumpMap,i(S.bumpMap,y.bumpMapTransform),y.bumpScale.value=S.bumpScale,S.side===Kn&&(y.bumpScale.value*=-1)),S.normalMap&&(y.normalMap.value=S.normalMap,i(S.normalMap,y.normalMapTransform),y.normalScale.value.copy(S.normalScale),S.side===Kn&&y.normalScale.value.negate()),S.displacementMap&&(y.displacementMap.value=S.displacementMap,i(S.displacementMap,y.displacementMapTransform),y.displacementScale.value=S.displacementScale,y.displacementBias.value=S.displacementBias),S.emissiveMap&&(y.emissiveMap.value=S.emissiveMap,i(S.emissiveMap,y.emissiveMapTransform)),S.specularMap&&(y.specularMap.value=S.specularMap,i(S.specularMap,y.specularMapTransform)),S.alphaTest>0&&(y.alphaTest.value=S.alphaTest);const z=t.get(S),W=z.envMap,C=z.envMapRotation;W&&(y.envMap.value=W,y.envMapRotation.value.setFromMatrix4(pw.makeRotationFromEuler(C)).transpose(),W.isCubeTexture&&W.isRenderTargetTexture===!1&&y.envMapRotation.value.premultiply(iS),y.reflectivity.value=S.reflectivity,y.ior.value=S.ior,y.refractionRatio.value=S.refractionRatio),S.lightMap&&(y.lightMap.value=S.lightMap,y.lightMapIntensity.value=S.lightMapIntensity,i(S.lightMap,y.lightMapTransform)),S.aoMap&&(y.aoMap.value=S.aoMap,y.aoMapIntensity.value=S.aoMapIntensity,i(S.aoMap,y.aoMapTransform))}function d(y,S){y.diffuse.value.copy(S.color),y.opacity.value=S.opacity,S.map&&(y.map.value=S.map,i(S.map,y.mapTransform))}function h(y,S){y.dashSize.value=S.dashSize,y.totalSize.value=S.dashSize+S.gapSize,y.scale.value=S.scale}function m(y,S,z,W){y.diffuse.value.copy(S.color),y.opacity.value=S.opacity,y.size.value=S.size*z,y.scale.value=W*.5,S.map&&(y.map.value=S.map,i(S.map,y.uvTransform)),S.alphaMap&&(y.alphaMap.value=S.alphaMap,i(S.alphaMap,y.alphaMapTransform)),S.alphaTest>0&&(y.alphaTest.value=S.alphaTest)}function g(y,S){y.diffuse.value.copy(S.color),y.opacity.value=S.opacity,y.rotation.value=S.rotation,S.map&&(y.map.value=S.map,i(S.map,y.mapTransform)),S.alphaMap&&(y.alphaMap.value=S.alphaMap,i(S.alphaMap,y.alphaMapTransform)),S.alphaTest>0&&(y.alphaTest.value=S.alphaTest)}function x(y,S){y.specular.value.copy(S.specular),y.shininess.value=Math.max(S.shininess,1e-4)}function _(y,S){S.gradientMap&&(y.gradientMap.value=S.gradientMap)}function v(y,S){y.metalness.value=S.metalness,S.metalnessMap&&(y.metalnessMap.value=S.metalnessMap,i(S.metalnessMap,y.metalnessMapTransform)),y.roughness.value=S.roughness,S.roughnessMap&&(y.roughnessMap.value=S.roughnessMap,i(S.roughnessMap,y.roughnessMapTransform)),S.envMap&&(y.envMapIntensity.value=S.envMapIntensity)}function E(y,S,z){y.ior.value=S.ior,S.sheen>0&&(y.sheenColor.value.copy(S.sheenColor).multiplyScalar(S.sheen),y.sheenRoughness.value=S.sheenRoughness,S.sheenColorMap&&(y.sheenColorMap.value=S.sheenColorMap,i(S.sheenColorMap,y.sheenColorMapTransform)),S.sheenRoughnessMap&&(y.sheenRoughnessMap.value=S.sheenRoughnessMap,i(S.sheenRoughnessMap,y.sheenRoughnessMapTransform))),S.clearcoat>0&&(y.clearcoat.value=S.clearcoat,y.clearcoatRoughness.value=S.clearcoatRoughness,S.clearcoatMap&&(y.clearcoatMap.value=S.clearcoatMap,i(S.clearcoatMap,y.clearcoatMapTransform)),S.clearcoatRoughnessMap&&(y.clearcoatRoughnessMap.value=S.clearcoatRoughnessMap,i(S.clearcoatRoughnessMap,y.clearcoatRoughnessMapTransform)),S.clearcoatNormalMap&&(y.clearcoatNormalMap.value=S.clearcoatNormalMap,i(S.clearcoatNormalMap,y.clearcoatNormalMapTransform),y.clearcoatNormalScale.value.copy(S.clearcoatNormalScale),S.side===Kn&&y.clearcoatNormalScale.value.negate())),S.dispersion>0&&(y.dispersion.value=S.dispersion),S.retroreflectivity>0&&(y.retroreflectivity.value=S.retroreflectivity),S.iridescence>0&&(y.iridescence.value=S.iridescence,y.iridescenceIOR.value=S.iridescenceIOR,y.iridescenceThicknessMinimum.value=S.iridescenceThicknessRange[0],y.iridescenceThicknessMaximum.value=S.iridescenceThicknessRange[1],S.iridescenceMap&&(y.iridescenceMap.value=S.iridescenceMap,i(S.iridescenceMap,y.iridescenceMapTransform)),S.iridescenceThicknessMap&&(y.iridescenceThicknessMap.value=S.iridescenceThicknessMap,i(S.iridescenceThicknessMap,y.iridescenceThicknessMapTransform))),S.transmission>0&&(y.transmission.value=S.transmission,y.transmissionSamplerMap.value=z.texture,y.transmissionSamplerSize.value.set(z.width,z.height),S.transmissionMap&&(y.transmissionMap.value=S.transmissionMap,i(S.transmissionMap,y.transmissionMapTransform)),y.thickness.value=S.thickness,S.thicknessMap&&(y.thicknessMap.value=S.thicknessMap,i(S.thicknessMap,y.thicknessMapTransform)),y.attenuationDistance.value=S.attenuationDistance,y.attenuationColor.value.copy(S.attenuationColor)),S.anisotropy>0&&(y.anisotropyVector.value.set(S.anisotropy*Math.cos(S.anisotropyRotation),S.anisotropy*Math.sin(S.anisotropyRotation)),S.anisotropyMap&&(y.anisotropyMap.value=S.anisotropyMap,i(S.anisotropyMap,y.anisotropyMapTransform))),y.specularIntensity.value=S.specularIntensity,y.specularColor.value.copy(S.specularColor),S.specularColorMap&&(y.specularColorMap.value=S.specularColorMap,i(S.specularColorMap,y.specularColorMapTransform)),S.specularIntensityMap&&(y.specularIntensityMap.value=S.specularIntensityMap,i(S.specularIntensityMap,y.specularIntensityMapTransform))}function R(y,S){S.matcap&&(y.matcap.value=S.matcap)}function N(y,S){const z=t.get(S).light;y.referencePosition.value.setFromMatrixPosition(z.matrixWorld),y.nearDistance.value=z.shadow.camera.near,y.farDistance.value=z.shadow.camera.far}return{refreshFogUniforms:r,refreshMaterialUniforms:c}}function gw(o,t,i,r){let c={},f={},d=[];const h=o.getParameter(o.MAX_UNIFORM_BUFFER_BINDINGS);function m(C,D){const U=D.program;r.uniformBlockBinding(C,U)}function g(C,D){let U=c[C.id];U===void 0&&(y(C),U=x(C),c[C.id]=U,C.addEventListener("dispose",z));const I=D.program;r.updateUBOMapping(C,I);const b=t.render.frame;f[C.id]!==b&&(v(C),f[C.id]=b)}function x(C){const D=_();C.__bindingPointIndex=D;const U=o.createBuffer(),I=C.__size,b=C.usage;return o.bindBuffer(o.UNIFORM_BUFFER,U),o.bufferData(o.UNIFORM_BUFFER,I,b),o.bindBuffer(o.UNIFORM_BUFFER,null),o.bindBufferBase(o.UNIFORM_BUFFER,D,U),U}function _(){for(let C=0;C<h;C++)if(d.indexOf(C)===-1)return d.push(C),C;return It("WebGLRenderer: Maximum number of simultaneously usable uniforms groups reached."),0}function v(C){const D=c[C.id],U=C.uniforms,I=C.__cache;o.bindBuffer(o.UNIFORM_BUFFER,D);for(let b=0,L=U.length;b<L;b++){const k=U[b];if(Array.isArray(k))for(let Y=0,$=k.length;Y<$;Y++)E(k[Y],b,Y,I);else E(k,b,0,I)}o.bindBuffer(o.UNIFORM_BUFFER,null)}function E(C,D,U,I){if(N(C,D,U,I)===!0){const b=C.__offset,L=C.value;if(Array.isArray(L)){let k=0;for(let Y=0;Y<L.length;Y++){const $=L[Y],O=S($);R($,C.__data,k),typeof $!="number"&&typeof $!="boolean"&&!$.isMatrix3&&!ArrayBuffer.isView($)&&(k+=O.storage/Float32Array.BYTES_PER_ELEMENT)}}else R(L,C.__data,0);o.bufferSubData(o.UNIFORM_BUFFER,b,C.__data)}}function R(C,D,U){typeof C=="number"||typeof C=="boolean"?D[0]=C:C.isMatrix3?(D[0]=C.elements[0],D[1]=C.elements[1],D[2]=C.elements[2],D[3]=0,D[4]=C.elements[3],D[5]=C.elements[4],D[6]=C.elements[5],D[7]=0,D[8]=C.elements[6],D[9]=C.elements[7],D[10]=C.elements[8],D[11]=0):ArrayBuffer.isView(C)?D.set(new C.constructor(C.buffer,C.byteOffset,D.length)):C.toArray(D,U)}function N(C,D,U,I){const b=C.value,L=D+"_"+U;if(I[L]===void 0)return typeof b=="number"||typeof b=="boolean"?I[L]=b:ArrayBuffer.isView(b)?I[L]=b.slice():I[L]=b.clone(),!0;{const k=I[L];if(typeof b=="number"||typeof b=="boolean"){if(k!==b)return I[L]=b,!0}else{if(ArrayBuffer.isView(b))return!0;if(k.equals(b)===!1)return k.copy(b),!0}}return!1}function y(C){const D=C.uniforms;let U=0;const I=16;for(let L=0,k=D.length;L<k;L++){const Y=Array.isArray(D[L])?D[L]:[D[L]];for(let $=0,O=Y.length;$<O;$++){const q=Y[$],ne=Array.isArray(q.value)?q.value:[q.value];for(let j=0,Q=ne.length;j<Q;j++){const pe=ne[j],ce=S(pe),me=U%I,be=me%ce.boundary,Ze=me+be;U+=be,Ze!==0&&I-Ze<ce.storage&&(U+=I-Ze),q.__data=new Float32Array(ce.storage/Float32Array.BYTES_PER_ELEMENT),q.__offset=U,U+=ce.storage}}}const b=U%I;return b>0&&(U+=I-b),C.__size=U,C.__cache={},this}function S(C){const D={boundary:0,storage:0};return typeof C=="number"||typeof C=="boolean"?(D.boundary=4,D.storage=4):C.isVector2?(D.boundary=8,D.storage=8):C.isVector3||C.isColor?(D.boundary=16,D.storage=12):C.isVector4?(D.boundary=16,D.storage=16):C.isMatrix3?(D.boundary=48,D.storage=48):C.isMatrix4?(D.boundary=64,D.storage=64):C.isTexture?ot("WebGLRenderer: Texture samplers can not be part of an uniforms group."):ArrayBuffer.isView(C)?(D.boundary=16,D.storage=C.byteLength):ot("WebGLRenderer: Unsupported uniform value type.",C),D}function z(C){const D=C.target;D.removeEventListener("dispose",z);const U=d.indexOf(D.__bindingPointIndex);d.splice(U,1),o.deleteBuffer(c[D.id]),delete c[D.id],delete f[D.id]}function W(){for(const C in c)o.deleteBuffer(c[C]);d=[],c={},f={}}return{bind:m,update:g,dispose:W}}const _w=new Uint16Array([12469,15057,12620,14925,13266,14620,13807,14376,14323,13990,14545,13625,14713,13328,14840,12882,14931,12528,14996,12233,15039,11829,15066,11525,15080,11295,15085,10976,15082,10705,15073,10495,13880,14564,13898,14542,13977,14430,14158,14124,14393,13732,14556,13410,14702,12996,14814,12596,14891,12291,14937,11834,14957,11489,14958,11194,14943,10803,14921,10506,14893,10278,14858,9960,14484,14039,14487,14025,14499,13941,14524,13740,14574,13468,14654,13106,14743,12678,14818,12344,14867,11893,14889,11509,14893,11180,14881,10751,14852,10428,14812,10128,14765,9754,14712,9466,14764,13480,14764,13475,14766,13440,14766,13347,14769,13070,14786,12713,14816,12387,14844,11957,14860,11549,14868,11215,14855,10751,14825,10403,14782,10044,14729,9651,14666,9352,14599,9029,14967,12835,14966,12831,14963,12804,14954,12723,14936,12564,14917,12347,14900,11958,14886,11569,14878,11247,14859,10765,14828,10401,14784,10011,14727,9600,14660,9289,14586,8893,14508,8533,15111,12234,15110,12234,15104,12216,15092,12156,15067,12010,15028,11776,14981,11500,14942,11205,14902,10752,14861,10393,14812,9991,14752,9570,14682,9252,14603,8808,14519,8445,14431,8145,15209,11449,15208,11451,15202,11451,15190,11438,15163,11384,15117,11274,15055,10979,14994,10648,14932,10343,14871,9936,14803,9532,14729,9218,14645,8742,14556,8381,14461,8020,14365,7603,15273,10603,15272,10607,15267,10619,15256,10631,15231,10614,15182,10535,15118,10389,15042,10167,14963,9787,14883,9447,14800,9115,14710,8665,14615,8318,14514,7911,14411,7507,14279,7198,15314,9675,15313,9683,15309,9712,15298,9759,15277,9797,15229,9773,15166,9668,15084,9487,14995,9274,14898,8910,14800,8539,14697,8234,14590,7790,14479,7409,14367,7067,14178,6621,15337,8619,15337,8631,15333,8677,15325,8769,15305,8871,15264,8940,15202,8909,15119,8775,15022,8565,14916,8328,14804,8009,14688,7614,14569,7287,14448,6888,14321,6483,14088,6171,15350,7402,15350,7419,15347,7480,15340,7613,15322,7804,15287,7973,15229,8057,15148,8012,15046,7846,14933,7611,14810,7357,14682,7069,14552,6656,14421,6316,14251,5948,14007,5528,15356,5942,15356,5977,15353,6119,15348,6294,15332,6551,15302,6824,15249,7044,15171,7122,15070,7050,14949,6861,14818,6611,14679,6349,14538,6067,14398,5651,14189,5311,13935,4958,15359,4123,15359,4153,15356,4296,15353,4646,15338,5160,15311,5508,15263,5829,15188,6042,15088,6094,14966,6001,14826,5796,14678,5543,14527,5287,14377,4985,14133,4586,13869,4257,15360,1563,15360,1642,15358,2076,15354,2636,15341,3350,15317,4019,15273,4429,15203,4732,15105,4911,14981,4932,14836,4818,14679,4621,14517,4386,14359,4156,14083,3795,13808,3437,15360,122,15360,137,15358,285,15355,636,15344,1274,15322,2177,15281,2765,15215,3223,15120,3451,14995,3569,14846,3567,14681,3466,14511,3305,14344,3121,14037,2800,13753,2467,15360,0,15360,1,15359,21,15355,89,15346,253,15325,479,15287,796,15225,1148,15133,1492,15008,1749,14856,1882,14685,1886,14506,1783,14324,1608,13996,1398,13702,1183]);let $i=null;function vw(){return $i===null&&($i=new rb(_w,16,16,Js,oa),$i.name="DFG_LUT",$i.minFilter=In,$i.magFilter=In,$i.wrapS=Ua,$i.wrapT=Ua,$i.generateMipmaps=!1,$i.needsUpdate=!0),$i}class xw{constructor(t={}){const{canvas:i=OE(),context:r=null,depth:c=!0,stencil:f=!1,alpha:d=!1,antialias:h=!1,premultipliedAlpha:m=!0,preserveDrawingBuffer:g=!1,powerPreference:x="default",failIfMajorPerformanceCaveat:_=!1,reversedDepthBuffer:v=!1,outputBufferType:E=fi}=t;this.isWebGLRenderer=!0;let R;if(r!==null){if(typeof WebGLRenderingContext<"u"&&r instanceof WebGLRenderingContext)throw new Error("THREE.WebGLRenderer: WebGL 1 is not supported since r163.");R=r.getContextAttributes().alpha}else R=d;const N=E,y=new Set([qp,Wp,Xp]),S=new Set([fi,ra,Sl,yl,Vp,kp]),z=new Uint32Array(4),W=new Int32Array(4),C=new he;let D=null,U=null;const I=[],b=[];let L=null;this.domElement=i,this.debug={checkShaderErrors:!0,diagnostics:{keywords:!1},onShaderError:null},this.autoClear=!0,this.autoClearColor=!0,this.autoClearDepth=!0,this.autoClearStencil=!0,this.sortObjects=!0,this.clippingPlanes=[],this.localClippingEnabled=!1,this.toneMapping=sa,this.toneMappingExposure=1,this.transmissionResolutionScale=1;const k=this;let Y=!1,$=null,O=null,q=null,ne=null;this._outputColorSpace=bi;let j=0,Q=0,pe=null,ce=-1,me=null;const be=new sn,Ze=new sn;let je=null;const H=new Dt(0);let F=0,ge=i.width,X=i.height,ue=1,Ae=null,Ie=null;const xe=new sn(0,0,ge,X),we=new sn(0,0,ge,X);let Vt=!1;const pt=new Qp;let gt=!1,St=!1;const tt=new un,it=new he,kt=new sn,dn={background:null,fog:null,environment:null,overrideMaterial:null,isScene:!0};let Pt=!1;function en(){return pe===null?ue:1}let K=r;function nn(T,G){return i.getContext(T,G)}let Ot,P,M,ie,le,_e,Re,De,ve,Ee,Ne,$e,Be,Pe,ke,nt,lt,Z,Ce,Me,Ue,Ve,Te;try{const T={alpha:!0,depth:c,stencil:f,antialias:h,premultipliedAlpha:m,preserveDrawingBuffer:g,powerPreference:x,failIfMajorPerformanceCaveat:_};if("setAttribute"in i&&i.setAttribute("data-engine",`three.js r${Hp}`),i.addEventListener("webglcontextlost",Rt,!1),i.addEventListener("webglcontextrestored",ct,!1),i.addEventListener("webglcontextcreationerror",Jn,!1),K===null){const G="webgl2";if(K=nn(G,T),K===null)throw nn(G)?new Error("THREE.WebGLRenderer: Error creating WebGL context with your selected attributes."):new Error("THREE.WebGLRenderer: Error creating WebGL context.")}Je()}catch(T){throw i.removeEventListener("webglcontextlost",Rt,!1),i.removeEventListener("webglcontextrestored",ct,!1),i.removeEventListener("webglcontextcreationerror",Jn,!1),It("WebGLRenderer: "+T.message),T}function Je(){Ot=new v1(K),Ot.init(),Ue=new cw(K,Ot),P=new l1(K,Ot,t,Ue),M=new ow(K,Ot),P.reversedDepthBuffer&&v&&M.buffers.depth.setReversed(!0),O=K.createFramebuffer(),q=K.createFramebuffer(),ne=K.createFramebuffer(),ie=new y1(K),le=new YR,_e=new lw(K,Ot,M,le,P,Ue,ie),Re=new _1(k),De=new Eb(K),Ve=new r1(K,De),ve=new x1(K,De,ie,Ve),Ee=new E1(K,ve,De,Ve,ie),Z=new M1(K,P,_e),ke=new c1(le),Ne=new qR(k,Re,Ot,P,Ve,ke),$e=new mw(k,le),Be=new ZR,Pe=new tw(Ot),lt=new s1(k,Re,M,Ee,R,m),nt=new rw(k,Ee,P),Te=new gw(K,ie,P,M),Ce=new o1(K,Ot,ie),Me=new S1(K,Ot,ie),ie.programs=Ne.programs,k.capabilities=P,k.extensions=Ot,k.properties=le,k.renderLists=Be,k.shadowMap=nt,k.state=M,k.info=ie}N!==fi&&(L=new T1(N,i.width,i.height,h,c,f));const Ge=new hw(k,K);this.xr=Ge,this.getContext=function(){return K},this.getContextAttributes=function(){return K.getContextAttributes()},this.forceContextLoss=function(){const T=Ot.get("WEBGL_lose_context");T&&T.loseContext()},this.forceContextRestore=function(){const T=Ot.get("WEBGL_lose_context");T&&T.restoreContext()},this.getPixelRatio=function(){return ue},this.setPixelRatio=function(T){T!==void 0&&(ue=T,this.setSize(ge,X,!1))},this.getSize=function(T){return T.set(ge,X)},this.setSize=function(T,G,de=!0){if(Ge.isPresenting){ot("WebGLRenderer: Can't change size while VR device is presenting.");return}ge=T,X=G,i.width=Math.floor(T*ue),i.height=Math.floor(G*ue),de===!0&&(i.style.width=T+"px",i.style.height=G+"px"),L!==null&&L.setSize(i.width,i.height),this.setViewport(0,0,T,G)},this.getDrawingBufferSize=function(T){return T.set(ge*ue,X*ue).floor()},this.setDrawingBufferSize=function(T,G,de){ge=T,X=G,ue=de,i.width=Math.floor(T*de),i.height=Math.floor(G*de),this.setViewport(0,0,T,G)},this.setEffects=function(T){if(N===fi){It("WebGLRenderer: setEffects() requires outputBufferType set to HalfFloatType or FloatType.");return}if(T){for(let G=0;G<T.length;G++)if(T[G].isOutputPass===!0){ot("WebGLRenderer: OutputPass is not needed in setEffects(). Tone mapping and color space conversion are applied automatically.");break}}L.setEffects(T||[])},this.getCurrentViewport=function(T){return T.copy(be)},this.getViewport=function(T){return T.copy(xe)},this.setViewport=function(T,G,de,se){T.isVector4?xe.set(T.x,T.y,T.z,T.w):xe.set(T,G,de,se),M.viewport(be.copy(xe).multiplyScalar(ue).round())},this.getScissor=function(T){return T.copy(we)},this.setScissor=function(T,G,de,se){T.isVector4?we.set(T.x,T.y,T.z,T.w):we.set(T,G,de,se),M.scissor(Ze.copy(we).multiplyScalar(ue).round())},this.getScissorTest=function(){return Vt},this.setScissorTest=function(T){M.setScissorTest(Vt=T)},this.setOpaqueSort=function(T){Ae=T},this.setTransparentSort=function(T){Ie=T},this.getClearColor=function(T){return T.copy(lt.getClearColor())},this.setClearColor=function(){lt.setClearColor(...arguments)},this.getClearAlpha=function(){return lt.getClearAlpha()},this.setClearAlpha=function(){lt.setClearAlpha(...arguments)},this.clear=function(T=!0,G=!0,de=!0){let se=0;if(T){let re=!1;if(pe!==null){const ze=pe.texture.format;re=y.has(ze)}if(re){const ze=pe.texture.type,Xe=S.has(ze),Le=lt.getClearColor(),Ye=lt.getClearAlpha(),Ke=Le.r,st=Le.g,ut=Le.b;Xe?(z[0]=Ke,z[1]=st,z[2]=ut,z[3]=Ye,K.clearBufferuiv(K.COLOR,0,z)):(W[0]=Ke,W[1]=st,W[2]=ut,W[3]=Ye,K.clearBufferiv(K.COLOR,0,W))}else se|=K.COLOR_BUFFER_BIT}G&&(se|=K.DEPTH_BUFFER_BIT,this.state.buffers.depth.setMask(!0)),de&&(se|=K.STENCIL_BUFFER_BIT,this.state.buffers.stencil.setMask(4294967295)),se!==0&&K.clear(se)},this.clearColor=function(){this.clear(!0,!1,!1)},this.clearDepth=function(){this.clear(!1,!0,!1)},this.clearStencil=function(){this.clear(!1,!1,!0)},this.setNodesHandler=function(T){T.setRenderer(this),$=T},this.dispose=function(){i.removeEventListener("webglcontextlost",Rt,!1),i.removeEventListener("webglcontextrestored",ct,!1),i.removeEventListener("webglcontextcreationerror",Jn,!1),lt.dispose(),Be.dispose(),Pe.dispose(),le.dispose(),Re.dispose(),Ee.dispose(),Ve.dispose(),Te.dispose(),Ne.dispose(),Ge.dispose(),Ge.removeEventListener("sessionstart",ys),Ge.removeEventListener("sessionend",za),zi.stop()};function Rt(T){T.preventDefault(),Ev("WebGLRenderer: Context Lost."),Y=!0}function ct(){Ev("WebGLRenderer: Context Restored."),Y=!1;const T=ie.autoReset,G=nt.enabled,de=nt.autoUpdate,se=nt.needsUpdate,re=nt.type;Je(),ie.autoReset=T,nt.enabled=G,nt.autoUpdate=de,nt.needsUpdate=se,nt.type=re}function Jn(T){It("WebGLRenderer: A WebGL context could not be created. Reason: ",T.statusMessage)}function di(T){const G=T.target;G.removeEventListener("dispose",di),Pu(G)}function Pu(T){er(T),le.remove(T)}function er(T){const G=le.get(T).programs;G!==void 0&&(G.forEach(function(de){Ne.releaseProgram(de)}),T.isShaderMaterial&&Ne.releaseShaderCache(T))}this.renderBufferDirect=function(T,G,de,se,re,ze){G===null&&(G=dn);const Xe=re.isMesh&&re.matrixWorld.determinantAffine()<0,Le=mo(T,G,de,se,re);M.setMaterial(se,Xe);let Ye=de.index,Ke=1;if(se.wireframe===!0){if(Ye=ve.getWireframeAttribute(de),Ye===void 0)return;Ke=2}const st=de.drawRange,ut=de.attributes.position;let We=st.start*Ke,yt=(st.start+st.count)*Ke;ze!==null&&(We=Math.max(We,ze.start*Ke),yt=Math.min(yt,(ze.start+ze.count)*Ke)),Ye!==null?(We=Math.max(We,0),yt=Math.min(yt,Ye.count)):ut!=null&&(We=Math.max(We,0),yt=Math.min(yt,ut.count));const _t=yt-We;if(_t<0||_t===1/0)return;Ve.setup(re,se,Le,de,Ye);let jt,Ft=Ce;if(Ye!==null&&(jt=De.get(Ye),Ft=Me,Ft.setIndex(jt)),re.isMesh)se.wireframe===!0?(M.setLineWidth(se.wireframeLinewidth*en()),Ft.setMode(K.LINES)):Ft.setMode(K.TRIANGLES);else if(re.isLine){let vn=se.linewidth;vn===void 0&&(vn=1),M.setLineWidth(vn*en()),re.isLineSegments?Ft.setMode(K.LINES):re.isLineLoop?Ft.setMode(K.LINE_LOOP):Ft.setMode(K.LINE_STRIP)}else re.isPoints?Ft.setMode(K.POINTS):re.isSprite&&Ft.setMode(K.TRIANGLES);if(re.isBatchedMesh)if(Ot.get("WEBGL_multi_draw"))Ft.renderMultiDraw(re._multiDrawStarts,re._multiDrawCounts,re._multiDrawCount);else{const vn=re._multiDrawStarts,Fe=re._multiDrawCounts,rn=re._multiDrawCount,wt=Ye?De.get(Ye).bytesPerElement:1,Bn=le.get(se).currentProgram.getUniforms();for(let $n=0;$n<rn;$n++)Bn.setValue(K,"_gl_DrawID",$n),Ft.render(vn[$n]/wt,Fe[$n])}else if(re.isInstancedMesh)Ft.renderInstances(We,_t,re.count);else if(de.isInstancedBufferGeometry){const vn=de._maxInstanceCount!==void 0?de._maxInstanceCount:1/0,Fe=Math.min(de.instanceCount,vn);Ft.renderInstances(We,_t,Fe)}else Ft.render(We,_t)};function Ss(T,G,de,se){$!==null&&T.isNodeMaterial&&$.setObject(se,T),gt===!0&&ke.setState(T,de,!1),T.transparent===!0&&T.side===na&&T.forceSinglePass===!1?(T.side=Kn,T.needsUpdate=!0,Ms(T,G,se),T.side=Ks,T.needsUpdate=!0,Ms(T,G,se),T.side=na):Ms(T,G,se)}this.compile=function(T,G,de=null){de===null&&(de=T),$!==null&&$.renderStart(T,G,de),U=Pe.get(de),U.init(G),b.push(U),de.traverseVisible(function(re){re.isLight&&re.layers.test(G.layers)&&(U.pushLight(re),re.castShadow&&U.pushShadow(re))}),T!==de&&T.traverseVisible(function(re){re.isLight&&re.layers.test(G.layers)&&(U.pushLight(re),re.castShadow&&U.pushShadow(re))}),U.setupLights(),$!==null&&$.updateLights(U.state.lightsArray),St=this.localClippingEnabled,gt=ke.init(this.clippingPlanes,St),gt===!0&&ke.setGlobalState(this.clippingPlanes,G),$!==null&&nt.render(U.state.shadowsArray,de,G);const se=new Set;return T.traverse(function(re){if(!(re.isMesh||re.isPoints||re.isLine||re.isSprite))return;const ze=re.material;if(ze)if(Array.isArray(ze))for(let Xe=0;Xe<ze.length;Xe++){const Le=ze[Xe];Ss(Le,de,G,re),se.add(Le)}else Ss(ze,de,G,re),se.add(ze)}),U=b.pop(),$!==null&&$.renderEnd(),se},this.compileAsync=function(T,G,de=null){const se=this.compile(T,G,de);return new Promise(re=>{function ze(){if(se.forEach(function(Xe){const Ye=le.get(Xe).currentProgram;(Ye===void 0||Ye.isReady())&&se.delete(Xe)}),se.size===0){re(T);return}setTimeout(ze,10)}Ot.get("KHR_parallel_shader_compile")!==null?ze():setTimeout(ze,10)})};let Ba=null;function fa(T){Ba&&Ba(T)}function ys(){zi.stop()}function za(){zi.start()}const zi=new Kx;zi.setAnimationLoop(fa),typeof self<"u"&&zi.setContext(self),this.setAnimationLoop=function(T){Ba=T,Ge.setAnimationLoop(T),T===null?zi.stop():zi.start()},Ge.addEventListener("sessionstart",ys),Ge.addEventListener("sessionend",za),this.render=function(T,G){if(G!==void 0&&G.isCamera!==!0){It("WebGLRenderer.render: camera is not an instance of THREE.Camera.");return}if(Y===!0)return;$!==null&&$.renderStart(T,G);const de=Ge.enabled===!0&&Ge.isPresenting===!0,se=L!==null&&(pe===null||de)&&L.begin(k,pe);if(T.matrixWorldAutoUpdate===!0&&T.updateMatrixWorld(),G.parent===null&&G.matrixWorldAutoUpdate===!0&&G.updateMatrixWorld(),Ge.enabled===!0&&Ge.isPresenting===!0&&(L===null||L.isCompositing()===!1)&&(Ge.cameraAutoUpdate===!0&&Ge.updateCamera(G),G=Ge.getCamera()),T.isScene===!0&&T.onBeforeRender(k,T,G,pe),U=Pe.get(T,b.length),U.init(G),U.state.textureUnits=_e.getTextureUnits(),b.push(U),tt.multiplyMatrices(G.projectionMatrix,G.matrixWorldInverse),pt.setFromProjectionMatrix(tt,aa,G.reversedDepth),St=this.localClippingEnabled,gt=ke.init(this.clippingPlanes,St),D=Be.get(T,I.length),D.init(),I.push(D),Ge.enabled===!0&&Ge.isPresenting===!0){const Xe=k.xr.getDepthSensingMesh();Xe!==null&&co(Xe,G,-1/0,k.sortObjects)}co(T,G,0,k.sortObjects),D.finish(),$!==null&&$.updateLights(U.state.lightsArray),k.sortObjects===!0&&D.sort(Ae,Ie),Pt=Ge.enabled===!1||Ge.isPresenting===!1||Ge.hasDepthSensing()===!1,Pt&&lt.addToRenderList(D,T),this.info.render.frame++,this.info.autoReset===!0&&this.info.reset(),gt===!0&&ke.beginShadows();const re=U.state.shadowsArray;if(nt.render(re,T,G),gt===!0&&ke.endShadows(),(se&&L.hasRenderPass())===!1){const Xe=D.opaque,Le=D.transmissive;if(U.setupLights(),G.isArrayCamera){const Ye=G.cameras;if(Le.length>0)for(let Ke=0,st=Ye.length;Ke<st;Ke++){const ut=Ye[Ke];tr(Xe,Le,T,ut)}Pt&&lt.render(T);for(let Ke=0,st=Ye.length;Ke<st;Ke++){const ut=Ye[Ke];uo(D,T,ut,ut.viewport)}}else Le.length>0&&tr(Xe,Le,T,G),Pt&&lt.render(T),uo(D,T,G)}pe!==null&&Q===0&&(_e.updateMultisampleRenderTarget(pe),_e.updateRenderTargetMipmap(pe)),se&&L.end(k),T.isScene===!0&&T.onAfterRender(k,T,G),Ve.resetDefaultState(),ce=-1,me=null,b.pop(),b.length>0?(U=b[b.length-1],_e.setTextureUnits(U.state.textureUnits),gt===!0&&ke.setGlobalState(k.clippingPlanes,U.state.camera)):U=null,I.pop(),I.length>0?D=I[I.length-1]:D=null,$!==null&&$.renderEnd()};function co(T,G,de,se){if(T.visible===!1)return;if(T.layers.test(G.layers)){if(T.isGroup)de=T.renderOrder;else if(T.isLOD)T.autoUpdate===!0&&T.update(G);else if(T.isLightProbeGrid)U.pushLightProbeGrid(T);else if(T.isLight)U.pushLight(T),T.castShadow&&U.pushShadow(T);else if(T.isSprite){if(!T.frustumCulled||T.intersectsFrustum(pt)){se&&kt.setFromMatrixPosition(T.matrixWorld).applyMatrix4(tt);const Xe=Ee.update(T),Le=T.material;Le.visible&&D.push(T,Xe,Le,de,kt.z,null,G)}}else if((T.isMesh||T.isLine||T.isPoints)&&(!T.frustumCulled||T.intersectsFrustum(pt))){const Xe=Ee.update(T),Le=T.material;if(se&&(T.boundingSphere!==void 0?(T.boundingSphere===null&&T.computeBoundingSphere(),kt.copy(T.boundingSphere.center)):(Xe.boundingSphere===null&&Xe.computeBoundingSphere(),kt.copy(Xe.boundingSphere.center)),kt.applyMatrix4(T.matrixWorld).applyMatrix4(tt)),Array.isArray(Le)){const Ye=Xe.groups;for(let Ke=0,st=Ye.length;Ke<st;Ke++){const ut=Ye[Ke],We=Le[ut.materialIndex];We&&We.visible&&D.push(T,Xe,We,de,kt.z,ut,G)}}else Le.visible&&D.push(T,Xe,Le,de,kt.z,null,G)}}const ze=T.children;for(let Xe=0,Le=ze.length;Xe<Le;Xe++)co(ze[Xe],G,de,se)}function uo(T,G,de,se){const{opaque:re,transmissive:ze,transparent:Xe}=T;U.setupLightsView(de),gt===!0&&ke.setGlobalState(k.clippingPlanes,de),se&&M.viewport(be.copy(se)),re.length>0&&Fi(re,G,de),ze.length>0&&Fi(ze,G,de),Xe.length>0&&Fi(Xe,G,de),M.buffers.depth.setTest(!0),M.buffers.depth.setMask(!0),M.buffers.color.setMask(!0),M.setPolygonOffset(!1)}function tr(T,G,de,se){if((de.isScene===!0?de.overrideMaterial:null)!==null)return;if(U.state.transmissionRenderTarget[se.id]===void 0){const We=Ot.has("EXT_color_buffer_half_float")||Ot.has("EXT_color_buffer_float");U.state.transmissionRenderTarget[se.id]=new Bi(1,1,{generateMipmaps:!0,type:We?oa:fi,minFilter:js,samples:Math.max(4,P.samples),stencilBuffer:f,resolveDepthBuffer:!1,resolveStencilBuffer:!1,storeMultisampledDepthBuffer:!1,storeMultisampledStencilBuffer:!1,colorSpace:Ct.workingColorSpace})}const ze=U.state.transmissionRenderTarget[se.id],Xe=se.viewport||be;ze.setSize(Xe.z*k.transmissionResolutionScale,Xe.w*k.transmissionResolutionScale);const Le=k.getRenderTarget(),Ye=k.getActiveCubeFace(),Ke=k.getActiveMipmapLevel();k.setRenderTarget(ze),k.getClearColor(H),F=k.getClearAlpha(),F<1&&k.setClearColor(16777215,.5),k.clear(),Pt&&lt.render(de);const st=k.toneMapping;k.toneMapping=sa;const ut=se.viewport;if(se.viewport!==void 0&&(se.viewport=void 0),U.setupLightsView(se),gt===!0&&ke.setGlobalState(k.clippingPlanes,se),Fi(T,de,se),_e.updateMultisampleRenderTarget(ze),_e.updateRenderTargetMipmap(ze),Ot.has("WEBGL_multisampled_render_to_texture")===!1){let We=!1;for(let yt=0,_t=G.length;yt<_t;yt++){const jt=G[yt],{object:Ft,geometry:vn,material:Fe,group:rn}=jt;if(Fe.side===na&&Ft.layers.test(se.layers)){const wt=Fe.side;Fe.side=Kn,Fe.needsUpdate=!0,Rl(Ft,de,se,vn,Fe,rn),Fe.side=wt,Fe.needsUpdate=!0,We=!0}}We===!0&&(_e.updateMultisampleRenderTarget(ze),_e.updateRenderTargetMipmap(ze))}k.setRenderTarget(Le,Ye,Ke),k.setClearColor(H,F),ut!==void 0&&(se.viewport=ut),k.toneMapping=st}function Fi(T,G,de){const se=G.isScene===!0?G.overrideMaterial:null;for(let re=0,ze=T.length;re<ze;re++){const Xe=T[re],{object:Le,geometry:Ye,group:Ke}=Xe;let st=Xe.material;st.allowOverride===!0&&se!==null&&(st=se),Le.layers.test(de.layers)&&Rl(Le,G,de,Ye,st,Ke)}}function Rl(T,G,de,se,re,ze){$!==null&&re.isNodeMaterial&&$.setObject(T,re),T.onBeforeRender(k,G,de,se,re,ze),T.modelViewMatrix.multiplyMatrices(de.matrixWorldInverse,T.matrixWorld),T.normalMatrix.getNormalMatrix(T.modelViewMatrix),re.onBeforeRender(k,G,de,se,T,ze),re.transparent===!0&&re.side===na&&re.forceSinglePass===!1?(re.side=Kn,re.needsUpdate=!0,k.renderBufferDirect(de,G,se,re,T,ze),re.side=Ks,re.needsUpdate=!0,k.renderBufferDirect(de,G,se,re,T,ze),re.side=na):k.renderBufferDirect(de,G,se,re,T,ze),T.onAfterRender(k,G,de,se,re,ze)}function Ms(T,G,de){G.isScene!==!0&&(G=dn);const se=le.get(T),re=U.state.lights,ze=U.state.shadowsArray,Xe=re.state.version,Le=Ne.getParameters(T,re.state,ze,G,de,U.state.lightProbeGridArray),Ye=Ne.getProgramCacheKey(Le);let Ke=se.programs;se.environment=T.isMeshStandardMaterial||T.isMeshLambertMaterial||T.isMeshPhongMaterial?G.environment:null,se.fog=G.fog;const st=T.isMeshStandardMaterial||T.isMeshLambertMaterial&&!T.envMap||T.isMeshPhongMaterial&&!T.envMap;se.envMap=Re.get(T.envMap||se.environment,st),se.envMapRotation=se.environment!==null&&T.envMap===null?G.environmentRotation:T.envMapRotation,Ke===void 0&&(T.addEventListener("dispose",di),Ke=new Map,se.programs=Ke);let ut=Ke.get(Ye);if(ut!==void 0){if(se.currentProgram===ut&&se.lightsStateVersion===Xe)return ho(T,Le),ut}else Le.uniforms=Ne.getUniforms(T),$!==null&&T.isNodeMaterial&&$.build(T,de,Le),T.onBeforeCompile(Le,k),ut=Ne.acquireProgram(Le,Ye),Ke.set(Ye,ut),se.uniforms=Le.uniforms;const We=se.uniforms;return(!T.isShaderMaterial&&!T.isRawShaderMaterial||T.clipping===!0)&&(We.clippingPlanes=ke.uniform),ho(T,Le),se.needsLights=Cl(T),se.lightsStateVersion=Xe,se.needsLights&&(We.ambientLightColor.value=re.state.ambient,We.lightProbe.value=re.state.probe,We.sunLights.value=re.state.sun,We.sunLightShadows.value=re.state.sunShadow,We.directionalLights.value=re.state.directional,We.directionalLightShadows.value=re.state.directionalShadow,We.spotLights.value=re.state.spot,We.spotLightShadows.value=re.state.spotShadow,We.rectAreaLights.value=re.state.rectArea,We.ltc_1.value=re.state.rectAreaLTC1,We.ltc_2.value=re.state.rectAreaLTC2,We.pointLights.value=re.state.point,We.pointLightShadows.value=re.state.pointShadow,We.hemisphereLights.value=re.state.hemi,We.sunShadowMatrix.value=re.state.sunShadowMatrix,We.sunShadowCascade.value=re.state.sunShadowCascade,We.directionalShadowMatrix.value=re.state.directionalShadowMatrix,We.spotLightMatrix.value=re.state.spotLightMatrix,We.spotLightMap.value=re.state.spotLightMap,We.pointShadowMatrix.value=re.state.pointShadowMatrix),se.lightProbeGrid=U.state.lightProbeGridArray.length>0,se.currentProgram=ut,se.uniformsList=null,ut}function fo(T){if(T.uniformsList===null){const G=T.currentProgram.getUniforms();T.uniformsList=Eu.seqWithValue(G.seq,T.uniforms)}return T.uniformsList}function ho(T,G){const de=le.get(T);de.outputColorSpace=G.outputColorSpace,de.batching=G.batching,de.batchingColor=G.batchingColor,de.instancing=G.instancing,de.instancingColor=G.instancingColor,de.instancingMorph=G.instancingMorph,de.skinning=G.skinning,de.morphTargets=G.morphTargets,de.morphNormals=G.morphNormals,de.morphColors=G.morphColors,de.morphTargetsCount=G.morphTargetsCount,de.numClippingPlanes=G.numClippingPlanes,de.numIntersection=G.numClipIntersection,de.vertexAlphas=G.vertexAlphas,de.vertexTangents=G.vertexTangents,de.toneMapping=G.toneMapping}function po(T,G){if(T.length===0)return null;if(T.length===1)return T[0].texture!==null?T[0]:null;C.setFromMatrixPosition(G.matrixWorld);for(let de=0,se=T.length;de<se;de++){const re=T[de];if(re.texture!==null&&re.boundingBox.containsPoint(C))return re}return null}function mo(T,G,de,se,re){G.isScene!==!0&&(G=dn),_e.resetTextureUnits();const ze=G.fog,Xe=se.isMeshStandardMaterial||se.isMeshLambertMaterial||se.isMeshPhongMaterial?G.environment:null,Le=pe===null?k.outputColorSpace:pe.isXRRenderTarget===!0?pe.texture.colorSpace:Ct.workingColorSpace,Ye=se.isMeshStandardMaterial||se.isMeshLambertMaterial&&!se.envMap||se.isMeshPhongMaterial&&!se.envMap,Ke=Re.get(se.envMap||Xe,Ye),st=se.vertexColors===!0&&!!de.attributes.color&&de.attributes.color.itemSize===4,ut=!!de.attributes.tangent&&(!!se.normalMap||se.anisotropy>0),We=!!de.morphAttributes.position,yt=!!de.morphAttributes.normal,_t=!!de.morphAttributes.color;let jt=sa;se.toneMapped&&(pe===null||pe.isXRRenderTarget===!0)&&(jt=k.toneMapping);const Ft=de.morphAttributes.position||de.morphAttributes.normal||de.morphAttributes.color,vn=Ft!==void 0?Ft.length:0,Fe=le.get(se),rn=U.state.lights;if(gt===!0&&(St===!0||T!==me)){const bt=T===me&&se.id===ce;ke.setState(se,T,bt)}let wt=!1;se.version===Fe.__version?(Fe.needsLights&&Fe.lightsStateVersion!==rn.state.version||Fe.outputColorSpace!==Le||re.isBatchedMesh&&Fe.batching===!1||!re.isBatchedMesh&&Fe.batching===!0||re.isBatchedMesh&&Fe.batchingColor===!0&&re._colorsTexture===null||re.isBatchedMesh&&Fe.batchingColor===!1&&re._colorsTexture!==null||re.isInstancedMesh&&Fe.instancing===!1||!re.isInstancedMesh&&Fe.instancing===!0||re.isSkinnedMesh&&Fe.skinning===!1||!re.isSkinnedMesh&&Fe.skinning===!0||re.isInstancedMesh&&Fe.instancingColor===!0&&re.instanceColor===null||re.isInstancedMesh&&Fe.instancingColor===!1&&re.instanceColor!==null||re.isInstancedMesh&&Fe.instancingMorph===!0&&re.morphTexture===null||re.isInstancedMesh&&Fe.instancingMorph===!1&&re.morphTexture!==null||Fe.envMap!==Ke||se.fog===!0&&Fe.fog!==ze||Fe.numClippingPlanes!==void 0&&(Fe.numClippingPlanes!==ke.numPlanes||Fe.numIntersection!==ke.numIntersection)||Fe.vertexAlphas!==st||Fe.vertexTangents!==ut||Fe.morphTargets!==We||Fe.morphNormals!==yt||Fe.morphColors!==_t||Fe.toneMapping!==jt||Fe.morphTargetsCount!==vn||!!Fe.lightProbeGrid!=U.state.lightProbeGridArray.length>0)&&(wt=!0):(wt=!0,Fe.__version=se.version);let Bn=Fe.currentProgram;wt===!0&&(Bn=Ms(se,G,re),$&&se.isNodeMaterial&&$.onUpdateProgram(se,Bn,Fe));let $n=!1,Hi=!1,vt=!1;const Bt=Bn.getUniforms(),Qt=Fe.uniforms;if(M.useProgram(Bn.program)&&($n=!0,Hi=!0,vt=!0),se.id!==ce&&(ce=se.id,Hi=!0),Fe.needsLights){const bt=po(U.state.lightProbeGridArray,re);Fe.lightProbeGrid!==bt&&(Fe.lightProbeGrid=bt,Hi=!0)}if($n||me!==T){M.buffers.depth.getReversed()&&T.reversedDepth!==!0&&(T._reversedDepth=!0,T.updateProjectionMatrix()),Bt.setValue(K,"projectionMatrix",T.projectionMatrix),Bt.setValue(K,"viewMatrix",T.matrixWorldInverse);const on=Bt.map.cameraPosition;on!==void 0&&on.setValue(K,it.setFromMatrixPosition(T.matrixWorld)),P.logarithmicDepthBuffer&&Bt.setValue(K,"logDepthBufFC",2/(Math.log(T.far+1)/Math.LN2)),(se.isMeshPhongMaterial||se.isMeshToonMaterial||se.isMeshLambertMaterial||se.isMeshBasicMaterial||se.isMeshStandardMaterial||se.isShaderMaterial)&&Bt.setValue(K,"isOrthographic",T.isOrthographicCamera===!0),me!==T&&(me=T,Hi=!0,vt=!0)}if(Fe.needsLights&&(rn.state.sunShadowMap.length>0&&Bt.setValue(K,"sunShadowMap",rn.state.sunShadowMap,_e),rn.state.directionalShadowMap.length>0&&Bt.setValue(K,"directionalShadowMap",rn.state.directionalShadowMap,_e),rn.state.spotShadowMap.length>0&&Bt.setValue(K,"spotShadowMap",rn.state.spotShadowMap,_e),rn.state.pointShadowMap.length>0&&Bt.setValue(K,"pointShadowMap",rn.state.pointShadowMap,_e)),re.isSkinnedMesh){Bt.setOptional(K,re,"bindMatrix"),Bt.setOptional(K,re,"bindMatrixInverse");const bt=re.skeleton;bt&&(bt.boneTexture===null&&bt.computeBoneTexture(),Bt.setValue(K,"boneTexture",bt.boneTexture,_e))}re.isBatchedMesh&&(Bt.setOptional(K,re,"batchingTexture"),Bt.setValue(K,"batchingTexture",re._matricesTexture,_e),Bt.setOptional(K,re,"batchingIdTexture"),Bt.setValue(K,"batchingIdTexture",re._indirectTexture,_e),Bt.setOptional(K,re,"batchingColorTexture"),re._colorsTexture!==null&&Bt.setValue(K,"batchingColorTexture",re._colorsTexture,_e));const ei=de.morphAttributes;if((ei.position!==void 0||ei.normal!==void 0||ei.color!==void 0)&&Z.update(re,de,Bn),(Hi||Fe.receiveShadow!==re.receiveShadow)&&(Fe.receiveShadow=re.receiveShadow,Bt.setValue(K,"receiveShadow",re.receiveShadow)),(se.isMeshStandardMaterial||se.isMeshLambertMaterial||se.isMeshPhongMaterial)&&se.envMap===null&&G.environment!==null&&(Qt.envMapIntensity.value=G.environmentIntensity),Qt.dfgLUT!==void 0&&(Qt.dfgLUT.value=vw()),Hi){if(Bt.setValue(K,"toneMappingExposure",k.toneMappingExposure),Fe.needsLights&&wl(Qt,vt),ze&&se.fog===!0&&$e.refreshFogUniforms(Qt,ze),$e.refreshMaterialUniforms(Qt,se,ue,X,U.state.transmissionRenderTarget[T.id]),Fe.needsLights&&Fe.lightProbeGrid){const bt=Fe.lightProbeGrid;Qt.probesSH.value=bt.texture,Qt.probesMin.value.copy(bt.boundingBox.min),Qt.probesMax.value.copy(bt.boundingBox.max),Qt.probesResolution.value.copy(bt.resolution)}Eu.upload(K,fo(Fe),Qt,_e)}if(se.isShaderMaterial&&se.uniformsNeedUpdate===!0&&(Eu.upload(K,fo(Fe),Qt,_e),se.uniformsNeedUpdate=!1),se.isSpriteMaterial&&Bt.setValue(K,"center",re.center),Bt.setValue(K,"modelViewMatrix",re.modelViewMatrix),Bt.setValue(K,"normalMatrix",re.normalMatrix),Bt.setValue(K,"modelMatrix",re.matrixWorld),se.uniformsGroups!==void 0){const bt=se.uniformsGroups;for(let on=0,da=bt.length;on<da;on++){const Nl=bt[on];Te.update(Nl,Bn),Te.bind(Nl,Bn)}}return Bn}function wl(T,G){T.ambientLightColor.needsUpdate=G,T.lightProbe.needsUpdate=G,T.sunLights.needsUpdate=G,T.sunLightShadows.needsUpdate=G,T.directionalLights.needsUpdate=G,T.directionalLightShadows.needsUpdate=G,T.pointLights.needsUpdate=G,T.pointLightShadows.needsUpdate=G,T.spotLights.needsUpdate=G,T.spotLightShadows.needsUpdate=G,T.rectAreaLights.needsUpdate=G,T.hemisphereLights.needsUpdate=G}function Cl(T){return T.isMeshLambertMaterial||T.isMeshToonMaterial||T.isMeshPhongMaterial||T.isMeshStandardMaterial||T.isShadowMaterial||T.isShaderMaterial&&T.lights===!0}this.getActiveCubeFace=function(){return j},this.getActiveMipmapLevel=function(){return Q},this.getRenderTarget=function(){return pe},this.setRenderTargetTextures=function(T,G,de){const se=le.get(T);se.__autoAllocateDepthBuffer=T.resolveDepthBuffer===!1,se.__autoAllocateDepthBuffer===!1&&(se.__useRenderToTexture=!1),le.get(T.texture).__webglTexture=G,le.get(T.depthTexture).__webglTexture=se.__autoAllocateDepthBuffer?void 0:de,se.__hasExternalTextures=!0},this.setRenderTargetFramebuffer=function(T,G){const de=le.get(T);de.__webglFramebuffer=G,de.__useDefaultFramebuffer=G===void 0},this.setRenderTarget=function(T,G=0,de=0){pe=T,j=G,Q=de;let se=null,re=!1,ze=!1;if(T){const Le=le.get(T);if(Le.__useDefaultFramebuffer!==void 0){M.bindFramebuffer(K.FRAMEBUFFER,Le.__webglFramebuffer),be.copy(T.viewport),Ze.copy(T.scissor),je=T.scissorTest,M.viewport(be),M.scissor(Ze),M.setScissorTest(je),ce=-1;return}else if(Le.__webglFramebuffer===void 0)_e.setupRenderTarget(T);else if(Le.__hasExternalTextures)_e.rebindTextures(T,le.get(T.texture).__webglTexture,le.get(T.depthTexture).__webglTexture);else if(T.depthBuffer){const st=T.depthTexture;if(Le.__boundDepthTexture!==st){if(st!==null&&le.has(st)&&(T.width!==st.image.width||T.height!==st.image.height))throw new Error("THREE.WebGLRenderer: Attached DepthTexture is initialized to the incorrect size.");_e.setupDepthRenderbuffer(T)}}const Ye=T.texture;(Ye.isData3DTexture||Ye.isDataArrayTexture||Ye.isCompressedArrayTexture)&&(ze=!0);const Ke=le.get(T).__webglFramebuffer;T.isWebGLCubeRenderTarget?(Array.isArray(Ke[G])?se=Ke[G][de]:se=Ke[G],re=!0):T.samples>0&&_e.useMultisampledRTT(T)===!1?se=le.get(T).__webglMultisampledFramebuffer:Array.isArray(Ke)?se=Ke[de]:se=Ke,be.copy(T.viewport),Ze.copy(T.scissor),je=T.scissorTest}else be.copy(xe).multiplyScalar(ue).floor(),Ze.copy(we).multiplyScalar(ue).floor(),je=Vt;if(de!==0&&(se=O),M.bindFramebuffer(K.FRAMEBUFFER,se)&&M.drawBuffers(T,se),M.viewport(be),M.scissor(Ze),M.setScissorTest(je),re){const Le=le.get(T.texture);K.framebufferTexture2D(K.FRAMEBUFFER,K.COLOR_ATTACHMENT0,K.TEXTURE_CUBE_MAP_POSITIVE_X+G,Le.__webglTexture,de)}else if(ze){const Le=G;for(let Ye=0;Ye<T.textures.length;Ye++){const Ke=le.get(T.textures[Ye]);K.framebufferTextureLayer(K.FRAMEBUFFER,K.COLOR_ATTACHMENT0+Ye,Ke.__webglTexture,de,Le)}}else if(T!==null&&de!==0){const Le=le.get(T.texture);K.framebufferTexture2D(K.FRAMEBUFFER,K.COLOR_ATTACHMENT0,K.TEXTURE_2D,Le.__webglTexture,de)}ce=-1};function hi(T){const G=le.get(T);return(G.__readFormat!==T.format||G.__readType!==T.type)&&(G.__readFormat=T.format,G.__readType=T.type,G.__formatReadable=P.textureFormatReadable(T.format),G.__typeReadable=P.textureTypeReadable(T.type)),G}this.readRenderTargetPixels=function(T,G,de,se,re,ze,Xe,Le=0){if(!(T&&T.isWebGLRenderTarget)){It("WebGLRenderer.readRenderTargetPixels: renderTarget is not THREE.WebGLRenderTarget.");return}let Ye=le.get(T).__webglFramebuffer;if(T.isWebGLCubeRenderTarget&&Xe!==void 0&&(Ye=Ye[Xe]),Ye){M.bindFramebuffer(K.FRAMEBUFFER,Ye);try{const Ke=T.textures[Le],st=Ke.format,ut=Ke.type;T.textures.length>1&&K.readBuffer(K.COLOR_ATTACHMENT0+Le);const We=hi(Ke);if(We.__formatReadable===!1){It("WebGLRenderer.readRenderTargetPixels: renderTarget is not in RGBA or implementation defined format.");return}if(We.__typeReadable===!1){It("WebGLRenderer.readRenderTargetPixels: renderTarget is not in UnsignedByteType or implementation defined type.");return}G>=0&&G<=T.width-se&&de>=0&&de<=T.height-re&&K.readPixels(G,de,se,re,Ue.convert(st),Ue.convert(ut),ze)}finally{const Ke=pe!==null?le.get(pe).__webglFramebuffer:null;M.bindFramebuffer(K.FRAMEBUFFER,Ke)}}},this.readRenderTargetPixelsAsync=async function(T,G,de,se,re,ze,Xe,Le=0){if(!(T&&T.isWebGLRenderTarget))throw new Error("THREE.WebGLRenderer.readRenderTargetPixels: renderTarget is not THREE.WebGLRenderTarget.");let Ye=le.get(T).__webglFramebuffer;if(T.isWebGLCubeRenderTarget&&Xe!==void 0&&(Ye=Ye[Xe]),Ye)if(G>=0&&G<=T.width-se&&de>=0&&de<=T.height-re){M.bindFramebuffer(K.FRAMEBUFFER,Ye);const Ke=T.textures[Le],st=Ke.format,ut=Ke.type;T.textures.length>1&&K.readBuffer(K.COLOR_ATTACHMENT0+Le);const We=hi(Ke);if(We.__formatReadable===!1)throw new Error("THREE.WebGLRenderer.readRenderTargetPixelsAsync: renderTarget is not in RGBA or implementation defined format.");if(We.__typeReadable===!1)throw new Error("THREE.WebGLRenderer.readRenderTargetPixelsAsync: renderTarget is not in UnsignedByteType or implementation defined type.");const yt=K.createBuffer();K.bindBuffer(K.PIXEL_PACK_BUFFER,yt),K.bufferData(K.PIXEL_PACK_BUFFER,ze.byteLength,K.STREAM_READ),K.readPixels(G,de,se,re,Ue.convert(st),Ue.convert(ut),0),K.bindBuffer(K.PIXEL_PACK_BUFFER,null);const _t=pe!==null?le.get(pe).__webglFramebuffer:null;M.bindFramebuffer(K.FRAMEBUFFER,_t);const jt=K.fenceSync(K.SYNC_GPU_COMMANDS_COMPLETE,0);return K.flush(),await PE(K,jt,4),K.bindBuffer(K.PIXEL_PACK_BUFFER,yt),K.getBufferSubData(K.PIXEL_PACK_BUFFER,0,ze),K.bindBuffer(K.PIXEL_PACK_BUFFER,null),K.deleteBuffer(yt),K.deleteSync(jt),ze}else throw new Error("THREE.WebGLRenderer.readRenderTargetPixelsAsync: requested read bounds are out of range.")},this.copyFramebufferToTexture=function(T,G=null,de=0){const se=Math.pow(2,-de),re=Math.floor(T.image.width*se),ze=Math.floor(T.image.height*se),Xe=G!==null?G.x:0,Le=G!==null?G.y:0;_e.setTexture2D(T,0),K.copyTexSubImage2D(K.TEXTURE_2D,de,0,0,Xe,Le,re,ze),M.unbindTexture()},this.copyTextureToTexture=function(T,G,de=null,se=null,re=0,ze=0){let Xe,Le,Ye,Ke,st,ut,We,yt,_t;const jt=T.isCompressedTexture?T.mipmaps[ze]:T.image;if(de!==null)Xe=de.max.x-de.min.x,Le=de.max.y-de.min.y,Ye=de.isBox3?de.max.z-de.min.z:1,Ke=de.min.x,st=de.min.y,ut=de.isBox3?de.min.z:0;else{const Qt=Math.pow(2,-re);Xe=Math.floor(jt.width*Qt),Le=Math.floor(jt.height*Qt),T.isDataArrayTexture?Ye=jt.depth:T.isData3DTexture?Ye=Math.floor(jt.depth*Qt):Ye=1,Ke=0,st=0,ut=0}se!==null?(We=se.x,yt=se.y,_t=se.z):(We=0,yt=0,_t=0);const Ft=Ue.convert(G.format),vn=Ue.convert(G.type);let Fe;G.isData3DTexture?(_e.setTexture3D(G,0),Fe=K.TEXTURE_3D):G.isDataArrayTexture||G.isCompressedArrayTexture?(_e.setTexture2DArray(G,0),Fe=K.TEXTURE_2D_ARRAY):(_e.setTexture2D(G,0),Fe=K.TEXTURE_2D),M.activeTexture(K.TEXTURE0),M.pixelStorei(K.UNPACK_FLIP_Y_WEBGL,G.flipY),M.pixelStorei(K.UNPACK_PREMULTIPLY_ALPHA_WEBGL,G.premultiplyAlpha),M.pixelStorei(K.UNPACK_ALIGNMENT,G.unpackAlignment);const rn=M.getParameter(K.UNPACK_ROW_LENGTH),wt=M.getParameter(K.UNPACK_IMAGE_HEIGHT),Bn=M.getParameter(K.UNPACK_SKIP_PIXELS),$n=M.getParameter(K.UNPACK_SKIP_ROWS),Hi=M.getParameter(K.UNPACK_SKIP_IMAGES);M.pixelStorei(K.UNPACK_ROW_LENGTH,jt.width),M.pixelStorei(K.UNPACK_IMAGE_HEIGHT,jt.height),M.pixelStorei(K.UNPACK_SKIP_PIXELS,Ke),M.pixelStorei(K.UNPACK_SKIP_ROWS,st),M.pixelStorei(K.UNPACK_SKIP_IMAGES,ut);const vt=T.isDataArrayTexture||T.isData3DTexture,Bt=G.isDataArrayTexture||G.isData3DTexture;if(T.isDepthTexture){const Qt=le.get(T),ei=le.get(G),bt=le.get(Qt.__renderTarget),on=le.get(ei.__renderTarget);M.bindFramebuffer(K.READ_FRAMEBUFFER,bt.__webglFramebuffer),M.bindFramebuffer(K.DRAW_FRAMEBUFFER,on.__webglFramebuffer);for(let da=0;da<Ye;da++)vt&&(K.framebufferTextureLayer(K.READ_FRAMEBUFFER,K.COLOR_ATTACHMENT0,le.get(T).__webglTexture,re,ut+da),K.framebufferTextureLayer(K.DRAW_FRAMEBUFFER,K.COLOR_ATTACHMENT0,le.get(G).__webglTexture,ze,_t+da)),K.blitFramebuffer(Ke,st,Xe,Le,We,yt,Xe,Le,K.DEPTH_BUFFER_BIT,K.NEAREST);M.bindFramebuffer(K.READ_FRAMEBUFFER,null),M.bindFramebuffer(K.DRAW_FRAMEBUFFER,null)}else if(re!==0||T.isRenderTargetTexture||le.has(T)){const Qt=le.get(T),ei=le.get(G);M.bindFramebuffer(K.READ_FRAMEBUFFER,q),M.bindFramebuffer(K.DRAW_FRAMEBUFFER,ne);for(let bt=0;bt<Ye;bt++)vt?K.framebufferTextureLayer(K.READ_FRAMEBUFFER,K.COLOR_ATTACHMENT0,Qt.__webglTexture,re,ut+bt):K.framebufferTexture2D(K.READ_FRAMEBUFFER,K.COLOR_ATTACHMENT0,K.TEXTURE_2D,Qt.__webglTexture,re),Bt?K.framebufferTextureLayer(K.DRAW_FRAMEBUFFER,K.COLOR_ATTACHMENT0,ei.__webglTexture,ze,_t+bt):K.framebufferTexture2D(K.DRAW_FRAMEBUFFER,K.COLOR_ATTACHMENT0,K.TEXTURE_2D,ei.__webglTexture,ze),re!==0?K.blitFramebuffer(Ke,st,Xe,Le,We,yt,Xe,Le,K.COLOR_BUFFER_BIT,K.NEAREST):Bt?K.copyTexSubImage3D(Fe,ze,We,yt,_t+bt,Ke,st,Xe,Le):K.copyTexSubImage2D(Fe,ze,We,yt,Ke,st,Xe,Le);M.bindFramebuffer(K.READ_FRAMEBUFFER,null),M.bindFramebuffer(K.DRAW_FRAMEBUFFER,null)}else Bt?T.isDataTexture||T.isData3DTexture?K.texSubImage3D(Fe,ze,We,yt,_t,Xe,Le,Ye,Ft,vn,jt.data):G.isCompressedArrayTexture?K.compressedTexSubImage3D(Fe,ze,We,yt,_t,Xe,Le,Ye,Ft,jt.data):K.texSubImage3D(Fe,ze,We,yt,_t,Xe,Le,Ye,Ft,vn,jt):T.isDataTexture?K.texSubImage2D(K.TEXTURE_2D,ze,We,yt,Xe,Le,Ft,vn,jt.data):T.isCompressedTexture?K.compressedTexSubImage2D(K.TEXTURE_2D,ze,We,yt,jt.width,jt.height,Ft,jt.data):K.texSubImage2D(K.TEXTURE_2D,ze,We,yt,Xe,Le,Ft,vn,jt);M.pixelStorei(K.UNPACK_ROW_LENGTH,rn),M.pixelStorei(K.UNPACK_IMAGE_HEIGHT,wt),M.pixelStorei(K.UNPACK_SKIP_PIXELS,Bn),M.pixelStorei(K.UNPACK_SKIP_ROWS,$n),M.pixelStorei(K.UNPACK_SKIP_IMAGES,Hi),ze===0&&G.generateMipmaps&&K.generateMipmap(Fe),M.unbindTexture()},this.initRenderTarget=function(T){le.get(T).__webglFramebuffer===void 0&&_e.setupRenderTarget(T)},this.initTexture=function(T){T.isCubeTexture?_e.setTextureCube(T,0):T.isData3DTexture?_e.setTexture3D(T,0):T.isDataArrayTexture||T.isCompressedArrayTexture?_e.setTexture2DArray(T,0):_e.setTexture2D(T,0),M.unbindTexture()},this.resetState=function(){j=0,Q=0,pe=null,M.reset(),Ve.reset()},typeof __THREE_DEVTOOLS__<"u"&&__THREE_DEVTOOLS__.dispatchEvent(new CustomEvent("observe",{detail:this}))}get coordinateSystem(){return aa}get outputColorSpace(){return this._outputColorSpace}set outputColorSpace(t){this._outputColorSpace=t;const i=this.getContext();i.drawingBufferColorSpace=Ct._getDrawingBufferColorSpace(t),i.unpackColorSpace=Ct._getUnpackColorSpace()}}function hx({lit:o}){const t=Gt.useRef(null),i=Gt.useRef(o);return i.current=o,Gt.useEffect(()=>{const r=t.current;if(!r)return;let c;try{c=new xw({canvas:r,antialias:!0,alpha:!1,powerPreference:"low-power"})}catch{return}c.setClearColor(1315086,1);const f=new QE;f.background=new Dt("#14110e");const d=new ui(32,1,.1,20);d.position.set(.15,.85,2.35),d.lookAt(0,.15,0);const h=[],m=[],g=(W,C,D)=>{h.push(W),m.push(C);const U=new la(W,C);return U.position.set(...D),f.add(U),U};g(new oo(2.4,.06,1.15),new Hh({color:3812386,roughness:.92}),[0,0,0]),g(new Cu(.012,.014,.42,12),new Hh({color:6048829,roughness:.8}),[.28,.24,.05]);const _=g(new Cu(.09,.14,.1,16,1,!0),new Hh({color:9272164,roughness:.7,side:na,emissive:new Dt("#f3e2bc"),emissiveIntensity:0}),[.28,.48,.05]).material,v=new xb(16052454,.18),E=new vb(15980452,0,3.2);E.position.set(.28,.42,.05),f.add(v,E);let R=0,N=i.current?1:0;const y=window.matchMedia("(prefers-reduced-motion: reduce)").matches,S=()=>{const W=r.clientWidth,C=r.clientHeight;if(W<2||C<2)return;const D=Math.min(window.devicePixelRatio||1,1.5);(r.width!==Math.floor(W*D)||r.height!==Math.floor(C*D))&&(c.setPixelRatio(D),c.setSize(W,C,!1),d.aspect=W/C,d.updateProjectionMatrix());const U=i.current?1:0;N+=(U-N)*(y?1:.08),E.intensity=.15+N*7.5,v.intensity=.16+N*.22,_.emissiveIntensity=N*.55,_.color.set(N>.5?"#f4e2bc":"#8d7b64"),c.render(f,d)},z=()=>{R=window.requestAnimationFrame(z),S(),y&&window.cancelAnimationFrame(R)};return z(),()=>{window.cancelAnimationFrame(R),h.forEach(W=>W.dispose()),m.forEach(W=>W.dispose()),c.dispose()}},[]),w.jsx("canvas",{ref:t,className:"shot-light"})}const Sw=["once","session","always","deny"];function px(o){const{t,scene:i,onScene:r}=o,c=3,f=i==="diff"?t.demo.review:i==="drama"?t.demo.video:i==="trust"?t.demo.harbor:t.demo.agent;return w.jsxs("div",{className:o.embed?"window is-embed":"window",children:[w.jsxs("div",{className:"w-title",children:[w.jsx("span",{className:"w-name",children:"yoyo"}),w.jsx("span",{className:"w-module",children:f}),w.jsx("span",{className:"w-model",children:t.model.modelName})]}),w.jsxs("div",{className:"w-body",children:[w.jsxs("aside",{className:"rail",children:[w.jsx("p",{className:"rail-label",children:t.demo.module}),w.jsx("button",{type:"button",className:i==="ask"?"is-on":void 0,onClick:()=>r("ask"),children:t.demo.agent}),w.jsx("button",{type:"button",className:i==="diff"?"is-on":void 0,onClick:()=>r("diff"),children:t.demo.review}),w.jsx("button",{type:"button",className:i==="drama"?"is-on":void 0,onClick:()=>r("drama"),children:t.demo.video}),w.jsx("button",{type:"button",className:i==="trust"?"is-on":void 0,onClick:()=>r("trust"),children:t.demo.harbor}),w.jsx("p",{className:"rail-label",children:t.demo.chats}),w.jsxs("button",{type:"button",className:i==="ask"?"thread is-live is-on":"thread is-live",onClick:()=>r("ask"),children:[t.demo.threadLive,w.jsx("small",{children:t.demo.threadLiveSub})]}),w.jsx("button",{type:"button",className:i==="trust"?"thread is-on":"thread",onClick:()=>r("trust"),children:t.demo.threadIdle}),w.jsx("p",{className:"thread is-quiet",children:t.demo.newChat})]}),w.jsxs("div",{className:"mid",children:[i==="ask"?w.jsx(yw,{t,phase:c,gate:o.gate,onGate:o.onGate}):null,i==="diff"?w.jsx(Mw,{t,hunks:o.hunks,onHunk:o.onHunk}):null,i==="drama"?w.jsx(Ew,{t,episode:o.episode,onEpisode:o.onEpisode,tab:o.videoTab,onTab:o.onVideoTab}):null,i==="trust"?w.jsx(bw,{t,promoted:o.promoted,onCheckout:o.onCheckout,refKey:o.refKey,onRef:o.onRef}):null]}),i==="ask"||i==="diff"?w.jsxs("aside",{className:"queue",children:[w.jsx("p",{className:"rail-label",children:t.demo.queue}),w.jsxs("button",{type:"button",onClick:()=>r("diff"),children:[w.jsx("b",{children:"loop.go"}),w.jsx("span",{children:"+24 −6"})]})]}):null]}),w.jsxs("div",{className:"w-status",children:[w.jsx("span",{children:t.demo.statusFolder}),w.jsx("span",{children:t.model.modelName}),w.jsx("span",{children:t.demo.local})]})]})}function yw({t:o,phase:t,gate:i,onGate:r}){const c={once:o.gate.onceHint,session:o.gate.sessionHint,always:o.gate.alwaysHint,deny:o.gate.denyHint}[i],f={once:o.demo.allowOnce,session:o.demo.allowSession,always:o.demo.allowAlways,deny:o.demo.reject},d={once:"1",session:"2",always:"3",deny:"Esc"},h=t>0&&t<3;return w.jsxs("div",{className:"pane",children:[w.jsxs("div",{className:"pane-scroll",children:[w.jsx("p",{className:"hello",children:o.demo.hello}),w.jsx("p",{className:"bubble",children:o.demo.bubble}),t>=1?w.jsxs("p",{className:"proc",role:"status",children:[w.jsx("i",{className:h&&t===1?"dot is-live":"dot","aria-hidden":"true"}),w.jsx("span",{children:t===1?o.demo.reading:o.demo.read}),w.jsx("code",{children:"loop.go"})]}):null,t>=2?w.jsxs("p",{className:"proc",children:[w.jsx("i",{className:t===2?"dot is-live":"dot is-done","aria-hidden":"true"}),w.jsx("span",{children:t===2?o.demo.patching:o.demo.patch})]}):null,t>=3?w.jsxs("div",{className:"gate-card",role:"status",children:[w.jsx("p",{className:"gate-kicker",children:o.demo.needsApproval}),w.jsxs("p",{className:"gate-action",children:[w.jsx("strong",{children:o.demo.action}),w.jsx("code",{children:o.demo.path})]}),w.jsx("div",{className:"chips",children:Sw.map(m=>w.jsxs("button",{type:"button",className:m===i?"chip is-hot":"chip","data-kind":m,"aria-pressed":m===i,onClick:()=>r(m),children:[f[m],w.jsx("kbd",{children:d[m]})]},m))}),w.jsx("p",{className:"verdict",children:c})]}):null]}),w.jsxs("div",{className:"composer","aria-hidden":"true",children:[w.jsx("span",{className:"plan",children:o.demo.plan}),w.jsx("span",{className:"composer-text",children:o.demo.composer}),w.jsx("span",{className:h?"send is-stop":"send"})]})]})}function Mw({t:o,hunks:t,onHunk:i}){const[r,c]=Gt.useState(!1),f=t.filter(Boolean).length;return Gt.useEffect(()=>{c(!1)},[t]),w.jsxs("div",{className:"pane diff-pane",children:[w.jsxs("div",{className:"diffhead",children:[w.jsx("b",{children:"loop.go"}),w.jsxs("span",{children:[f," ",o.demo.hunks]}),w.jsx("button",{type:"button",className:"apply",disabled:f===0||r,onClick:()=>c(!0),children:r?o.demo.kept:`${o.demo.apply}${f?` ${f}`:""}`})]}),w.jsxs("div",{className:"diff-file",children:[w.jsxs("p",{className:"ctx",children:["func parse(src string) error ","{"]}),w.jsx(mx,{on:t[0],onToggle:()=>i(0),del:"− if err != nil { return }",add:"+ if err != nil { return err }"}),w.jsx(mx,{on:t[1],onToggle:()=>i(1),del:"− log.Print(got)",add:'+ return fmt.Errorf("parse: %w", err)'}),w.jsx("p",{className:"ctx",children:"  return nil"}),w.jsx("p",{className:"ctx",children:"}"}),r?w.jsx("p",{className:"kept-note",children:o.demo.keptNote}):null]})]})}function mx({on:o,onToggle:t,del:i,add:r}){return w.jsxs("button",{type:"button",className:o?"hunk is-on":"hunk","aria-pressed":o,onClick:t,children:[w.jsx("i",{"aria-hidden":"true"}),w.jsx("span",{className:"del",children:i}),w.jsx("span",{className:"add",children:r})]})}function Ew({t:o,episode:t,onEpisode:i,tab:r,onTab:c}){const[f,d]=Gt.useState("video"),h=r??f,m=x=>{c==null||c(x),r===void 0&&d(x)},g=[["01",o.demo.ep1],["02",o.demo.ep2],["03",o.demo.ep3]];return w.jsxs("div",{className:"pane drama-pane",children:[w.jsx("div",{className:"vseg",role:"tablist","aria-label":o.demo.video,children:[["create",o.demo.createTab],["video",o.demo.video],["canvas",o.demo.canvasTab]].map(([x,_])=>w.jsx("button",{type:"button",role:"tab","aria-selected":h===x,onClick:()=>m(x),children:_},x))}),h==="create"?w.jsxs("div",{className:"vbody",children:[w.jsx("p",{className:"bubble",children:o.demo.createHint}),w.jsxs("figure",{className:"shot",children:[w.jsx(hx,{lit:!0}),w.jsx("figcaption",{children:o.demo.createHint})]}),w.jsx("p",{className:"vcopy",children:o.demo.createBody})]}):null,h==="video"?w.jsxs("div",{className:"vbody",children:[w.jsx("p",{className:"bubble",children:o.demo.dramaPrompt}),w.jsx("div",{className:"eps",children:g.map(([x,_])=>w.jsxs("button",{type:"button",className:t===x?"is-on":void 0,"aria-pressed":t===x,onClick:()=>i(x),children:[w.jsx("b",{children:x}),w.jsx("span",{children:_})]},x))}),w.jsxs("figure",{className:"shot",children:[w.jsx(hx,{lit:t==="02"}),w.jsx("figcaption",{children:t==="02"?o.demo.shotCap:o.demo.createHint})]})]}):null,h==="canvas"?w.jsxs("div",{className:"vbody",children:[w.jsxs("div",{className:"board",children:[g.map(([x,_])=>w.jsxs("button",{type:"button",className:t===x?"is-on":void 0,"aria-pressed":t===x,onClick:()=>i(x),children:[w.jsx("i",{}),w.jsxs("span",{children:[x," ",_]})]},x)),w.jsx("div",{className:"board-empty","aria-hidden":"true"})]}),w.jsx("p",{className:"vcopy",children:o.demo.canvasHint})]}):null]})}function bw({t:o,promoted:t,onCheckout:i,refKey:r,onRef:c}){const f=t&&r==="active"?o.trust.done:r==="staging"?o.trust.stagingHint:r==="canary"?o.trust.canaryHint:o.trust.activeHint,d=[["staging",o.trust.staging],["canary",o.trust.canary],["active",o.trust.active]];return w.jsxs("div",{className:"pane trust-pane",children:[w.jsx("div",{className:"w-refs",role:"radiogroup","aria-label":o.nav.trust,children:d.map(([h,m])=>w.jsxs("button",{type:"button",role:"radio","data-ref":h,"aria-checked":r===h,onClick:()=>c(h),children:[w.jsx("b",{children:m}),w.jsx("i",{children:h})]},h))}),w.jsx("p",{className:"w-ref-note","aria-live":"polite",children:f}),w.jsx("button",{type:"button",className:"apply",disabled:t||r!=="canary",onClick:i,children:o.trust.checkout})]})}const ea=["ask","diff","drama","trust"],gs="https://github.com/Shenchangxin/yoyo";function Tw(){const o=new URLSearchParams(window.location.search).get("theme");return o==="dark"||o==="light"?o:document.documentElement.dataset.theme==="dark"?"dark":"light"}function Aw(){const[o,t]=Gt.useState(()=>jM()),[i,r]=Gt.useState(()=>Tw()),[c,f]=Gt.useState(!1),[d,h]=Gt.useState("workstation"),[m,g]=Gt.useState(()=>{const F=new URLSearchParams(window.location.search).get("scene");return F==="ask"||F==="diff"||F==="drama"||F==="trust"?F:"ask"}),[x,_]=Gt.useState("once"),[v,E]=Gt.useState(!1),[R,N]=Gt.useState("canary"),[y,S]=Gt.useState([!0,!0]),[z,W]=Gt.useState("02"),[C,D]=Gt.useState(1),U=C===0?"create":C===2?"canvas":"video",[I,b]=Gt.useState(_l),[L,k]=Gt.useState(()=>jh(_l)),[Y,$]=Gt.useState("windows"),O=YM[o],q=Gt.useMemo(()=>L.find(F=>F.os===Y)||L[0],[L,Y]);Gt.useEffect(()=>{var F,ge,X;document.documentElement.dataset.lang=o,document.documentElement.lang=ZM(o),document.title=`${O.meta.title} · ${I}`,(F=document.querySelector('meta[name="description"]'))==null||F.setAttribute("content",O.meta.description),(ge=document.querySelector('meta[property="og:title"]'))==null||ge.setAttribute("content",O.meta.title),(X=document.querySelector('meta[property="og:description"]'))==null||X.setAttribute("content",O.meta.description)},[o,O.meta.description,O.meta.title,I]),Gt.useEffect(()=>{var F;document.documentElement.dataset.theme=i,document.documentElement.style.colorScheme=i,(F=document.querySelector('meta[name="theme-color"]'))==null||F.setAttribute("content",i==="dark"?"#141413":"#f3f0e8")},[i]),Gt.useEffect(()=>{const F=window.matchMedia("(prefers-color-scheme: dark)"),ge=()=>{try{if(localStorage.getItem("yoyo-site-theme")||new URLSearchParams(window.location.search).get("theme"))return}catch{return}r(F.matches?"dark":"light")};return F.addEventListener("change",ge),()=>F.removeEventListener("change",ge)},[]),Gt.useEffect(()=>{$(QM()),JM().then(F=>{b(F.version),k(F.assets)})},[]),Gt.useEffect(()=>(document.body.classList.toggle("menu-open",c),()=>document.body.classList.remove("menu-open")),[c]),Gt.useEffect(()=>{const F=()=>{var X;const ge=window.location.hash.replace(/^#/,"");ge&&((X=document.getElementById(ge))==null||X.scrollIntoView())};return F(),window.addEventListener("hashchange",F),()=>window.removeEventListener("hashchange",F)},[]),Gt.useEffect(()=>{const F=[...document.querySelectorAll("[data-zone]")],ge=new IntersectionObserver(X=>{const ue=X.filter(Ae=>Ae.isIntersecting).sort((Ae,Ie)=>Ie.intersectionRatio-Ae.intersectionRatio)[0];(ue==null?void 0:ue.target)instanceof HTMLElement&&ue.target.dataset.zone&&h(ue.target.dataset.zone)},{rootMargin:"-30% 0px -45% 0px",threshold:[.2,.45,.7]});return F.forEach(X=>ge.observe(X)),()=>ge.disconnect()},[]),Gt.useEffect(()=>{const F=ge=>{if(ge.metaKey||ge.ctrlKey||ge.altKey)return;if(c&&ge.key==="Escape"){f(!1);return}const X=ge.target;if(X!=null&&X.closest("input, textarea, select")||m!=="ask")return;const ue={1:"once",2:"session",3:"always",Escape:"deny"}[ge.key];ue&&_(ue)};return document.addEventListener("keydown",F),()=>document.removeEventListener("keydown",F)},[c,m]);const ne=F=>{t(F);try{localStorage.setItem("yoyo-site-lang",F)}catch{}},j=()=>{const F=i==="dark"?"light":"dark";r(F);try{localStorage.setItem("yoyo-site-theme",F)}catch{}},Q=F=>{const ge=ea.indexOf(m),X=F.key==="ArrowRight"?ea[(ge+1)%ea.length]:F.key==="ArrowLeft"?ea[(ge-1+ea.length)%ea.length]:F.key==="Home"?ea[0]:F.key==="End"?ea[ea.length-1]:null;X&&(F.preventDefault(),g(X),window.requestAnimationFrame(()=>{var ue;return(ue=document.getElementById(`scene-${X}`))==null?void 0:ue.focus()}))},pe=()=>{E(!0),N("active")},ce=O.stage[m],me={ask:O.demo.agent,diff:O.demo.review,drama:O.demo.video,trust:O.nav.trust},be=v&&R==="active"?O.trust.done:R==="staging"?O.trust.stagingHint:R==="canary"?O.trust.canaryHint:O.trust.activeHint,Ze=[["l1",O.model.l1,O.model.l1Body],["l2",O.model.l2,O.model.l2Body],["l3",O.model.l3,O.model.l3Body],["l4",O.model.l4,O.model.l4Body]],je=[["staging",O.trust.staging,O.trust.stagingHint],["canary",O.trust.canary,O.trust.canaryHint],["active",O.trust.active,O.trust.activeHint]],H=()=>f(!1);return w.jsxs(w.Fragment,{children:[w.jsx("a",{className:"skip",href:"#download",children:O.nav.download}),w.jsxs("header",{className:"nav",children:[w.jsxs("a",{className:"brand",href:"#top",onClick:H,children:[w.jsx(_x,{}),w.jsx("span",{children:"yoyo"})]}),w.jsxs("nav",{className:"nav-links","aria-label":"On this page",children:[w.jsx("a",{href:"#top","aria-current":d==="workstation"?"true":void 0,children:O.nav.workstation}),w.jsx("a",{href:"#video","aria-current":d==="video"?"true":void 0,children:O.nav.video}),w.jsx("a",{href:"#model","aria-current":d==="model"?"true":void 0,children:O.nav.model}),w.jsx("a",{href:"#trust","aria-current":d==="trust"?"true":void 0,children:O.nav.trust}),w.jsx("a",{href:"#doors","aria-current":d==="doors"?"true":void 0,children:O.nav.doors}),w.jsx("a",{href:"#download","aria-current":d==="download"?"true":void 0,children:O.nav.download})]}),w.jsxs("div",{className:"nav-end",children:[w.jsx(gx,{lang:o,onPick:ne}),w.jsx("button",{type:"button",className:"icon-btn","aria-pressed":i==="dark","aria-label":i==="dark"?O.nav.themeToLight:O.nav.themeToDark,onClick:j,children:i==="dark"?w.jsx(ww,{}):w.jsx(Cw,{})}),w.jsx("a",{className:"nav-github",href:gs,children:"GitHub"}),w.jsxs("a",{className:"btn btn-ink nav-cta",href:(q==null?void 0:q.href)||hh,"aria-label":O.nav.download,children:[w.jsx(vx,{}),w.jsx("span",{className:"nav-cta-label",children:O.nav.download})]}),w.jsx("button",{type:"button",className:"icon-btn menu-btn","aria-expanded":c,"aria-controls":"nav-panel","aria-label":c?O.nav.close:O.nav.menu,onClick:()=>f(F=>!F),children:w.jsx(Nw,{open:c})})]}),c?w.jsx("button",{type:"button",className:"nav-backdrop","aria-label":O.nav.close,onClick:H}):null,w.jsxs("div",{className:"nav-panel",id:"nav-panel",hidden:!c,children:[w.jsxs("nav",{"aria-label":"On this page",children:[w.jsx("a",{href:"#top",onClick:H,children:O.nav.workstation}),w.jsx("a",{href:"#video",onClick:H,children:O.nav.video}),w.jsx("a",{href:"#model",onClick:H,children:O.nav.model}),w.jsx("a",{href:"#trust",onClick:H,children:O.nav.trust}),w.jsx("a",{href:"#doors",onClick:H,children:O.nav.doors}),w.jsx("a",{href:"#download",onClick:H,children:O.nav.download}),w.jsx("a",{href:gs,children:O.hero.github})]}),w.jsxs("div",{className:"nav-panel-tools",children:[w.jsx(gx,{lang:o,onPick:ne}),w.jsx("button",{type:"button",className:"text-link",onClick:j,children:i==="dark"?O.nav.themeToLight:O.nav.themeToDark})]})]})]}),w.jsxs("main",{children:[w.jsx("section",{className:"hero",id:"top","data-zone":"workstation",children:w.jsxs("div",{className:"wrap hero-grid",children:[w.jsxs("div",{className:"hero-copy",children:[w.jsxs("p",{className:"kicker",children:["yoyo · ",I]}),w.jsx(Da,{as:"h1",text:O.hero.title}),w.jsx("p",{className:"dek",children:O.hero.dek}),w.jsxs("div",{className:"hero-actions",children:[w.jsxs("a",{className:"btn btn-ink",href:(q==null?void 0:q.href)||hh,children:[w.jsx(vx,{}),w.jsxs("span",{children:[O.hero.download," ",I,q?w.jsxs("span",{className:"btn-os",children:[" · ",q.label]}):null]})]}),w.jsx("a",{className:"text-link",href:"#download",children:O.hero.builds})]}),w.jsx("p",{className:"premise",children:O.get.premise}),w.jsx("ul",{className:"ledger",children:O.hero.chips.map(F=>w.jsx("li",{children:F},F))})]}),w.jsxs("div",{className:"hero-stage",children:[w.jsx("div",{id:"stage",role:"tabpanel","aria-labelledby":`scene-${m}`,"aria-describedby":"stage-note",children:w.jsx(px,{t:O,scene:m,onScene:g,gate:x,onGate:_,promoted:v,onCheckout:pe,refKey:R,onRef:N,hunks:y,onHunk:F=>S(ge=>F===0?[!ge[0],ge[1]]:[ge[0],!ge[1]]),episode:z,onEpisode:W})}),w.jsx("div",{className:"scene-tabs",role:"tablist","aria-label":O.nav.workstation,onKeyDown:Q,children:ea.map(F=>w.jsx("button",{id:`scene-${F}`,type:"button",role:"tab","aria-selected":m===F,tabIndex:m===F?0:-1,onClick:()=>g(F),children:me[F]},F))}),w.jsxs("p",{className:"stage-note",id:"stage-note",children:[ce,m==="drama"?w.jsx("a",{className:"text-link",href:"#video",children:O.nav.video}):null]})]})]})}),w.jsxs("section",{className:"section section-stack wrap",id:"work","data-zone":"workstation",children:[w.jsxs("header",{className:"section-head",children:[w.jsx("p",{className:"kicker",children:"01"}),w.jsx(Da,{text:O.turn.title}),w.jsx("p",{className:"dek",children:O.turn.dek})]}),w.jsx("ol",{className:"steps",children:O.turn.steps.map((F,ge)=>w.jsxs("li",{children:[w.jsx("span",{className:"step-n",children:String(ge+1).padStart(2,"0")}),w.jsx("h3",{children:F.title}),w.jsx("p",{children:F.body})]},F.title))})]}),w.jsxs("section",{className:"section section-stack wrap",id:"ability","data-zone":"workstation",children:[w.jsxs("header",{className:"section-head",children:[w.jsx("p",{className:"kicker",children:"02"}),w.jsx(Da,{text:O.pitch.title}),w.jsx("p",{className:"dek",children:O.pitch.dek})]}),w.jsx("div",{className:"ability-grid",children:O.pitch.cards.map((F,ge)=>w.jsxs("article",{children:[w.jsx("span",{className:"kicker",children:String(ge+1).padStart(2,"0")}),w.jsx("h3",{children:F.title}),w.jsx("p",{children:F.body})]},F.title))})]}),w.jsxs("section",{className:"section section-stack wrap",id:"video","data-zone":"video",children:[w.jsxs("header",{className:"section-head",children:[w.jsx("p",{className:"kicker",children:"03"}),w.jsx(Da,{text:O.film.title}),w.jsx("p",{className:"dek",children:O.film.dek})]}),w.jsxs("div",{className:"film-grid",children:[w.jsx(px,{t:O,scene:"drama",embed:!0,onScene:()=>{},videoTab:U,onVideoTab:F=>D(F==="create"?0:F==="canvas"?2:1),gate:x,onGate:_,promoted:v,onCheckout:pe,refKey:R,onRef:N,hunks:y,onHunk:F=>S(ge=>F===0?[!ge[0],ge[1]]:[ge[0],!ge[1]]),episode:z,onEpisode:W}),w.jsx("div",{className:"film-modes",role:"tablist","aria-label":O.nav.video,children:O.film.modes.map((F,ge)=>w.jsxs("button",{type:"button",role:"tab","aria-selected":C===ge,onClick:()=>D(ge),children:[w.jsx("b",{children:F.title}),w.jsx("span",{children:F.body})]},F.title))})]}),w.jsx("h3",{className:"phases-title",children:O.film.phasesTitle}),w.jsx("ol",{className:"phases",children:O.film.phases.map((F,ge)=>w.jsxs("li",{children:[w.jsx("span",{className:"step-n",children:String(ge+1).padStart(2,"0")}),w.jsx("h3",{children:F.title}),w.jsx("p",{children:F.body})]},F.title))}),w.jsx("p",{className:"apart",children:O.film.apart})]}),w.jsxs("section",{className:"section split wrap",id:"model","data-zone":"model",children:[w.jsxs("div",{className:"section-copy",children:[w.jsx("p",{className:"kicker",children:"04"}),w.jsx(Da,{text:O.model.title}),w.jsx("p",{className:"dek",children:O.model.dek})]}),w.jsxs("div",{className:"section-body",children:[w.jsxs("p",{className:"weights",children:[w.jsx("b",{children:O.model.modelName}),w.jsx("span",{children:O.model.weightsStill})]}),w.jsx("ul",{className:"layers",children:Ze.map(([F,ge,X])=>w.jsxs("li",{className:"layer","data-layer":F,children:[w.jsx("b",{children:ge}),w.jsx("p",{children:X})]},F))}),w.jsxs("p",{className:"fp",children:[w.jsx("span",{children:O.model.fp}),w.jsx("code",{children:"refs/models/<fingerprint>/active"})]}),w.jsx("p",{className:"fp-body",children:O.model.fpBody}),w.jsx("p",{className:"more",children:w.jsx("a",{href:`${gs}/blob/master/docs/architecture/invariants.md`,children:O.foot.invariants})})]})]}),w.jsxs("section",{className:"section split wrap",id:"trust","data-zone":"trust",children:[w.jsxs("div",{className:"section-copy",children:[w.jsx("p",{className:"kicker",children:"05"}),w.jsx(Da,{id:"trust-title",text:O.trust.title}),w.jsx("p",{className:"dek",children:O.trust.dek})]}),w.jsxs("div",{className:"section-body",children:[w.jsx("div",{className:"track",role:"radiogroup","aria-labelledby":"trust-title",children:je.map(([F,ge])=>w.jsxs("button",{type:"button",role:"radio","data-ref":F,"aria-checked":R===F,onClick:()=>N(F),children:[w.jsx("b",{children:ge}),w.jsx("i",{children:F})]},F))}),w.jsx("p",{className:"ref-note","aria-live":"polite",children:be}),w.jsx("button",{type:"button",className:"btn btn-ink",disabled:v||R!=="canary",onClick:pe,children:O.trust.checkout})]})]}),w.jsxs("section",{className:"section section-stack wrap",id:"doors","data-zone":"doors",children:[w.jsxs("header",{className:"section-head",children:[w.jsx("p",{className:"kicker",children:"06"}),w.jsx(Da,{text:O.doors.title}),w.jsx("p",{className:"dek",children:O.doors.dek})]}),w.jsxs("ol",{className:"door-grid",children:[w.jsxs("li",{children:[w.jsx("h3",{children:O.doors.desktop}),w.jsx("p",{children:O.doors.desktopBody})]}),w.jsxs("li",{children:[w.jsx("h3",{children:O.doors.browser}),w.jsx("p",{children:O.doors.browserBody}),w.jsx("code",{children:"yoyo serve"})]}),w.jsxs("li",{children:[w.jsx("h3",{children:O.doors.cli}),w.jsx("p",{children:O.doors.cliBody}),w.jsx("code",{children:"yoyo run"})]})]}),w.jsxs("div",{className:"aside",children:[w.jsxs("article",{children:[w.jsx("h3",{children:O.doors.skills}),w.jsx("p",{children:w.jsx(Rw,{text:O.doors.skillsBody})})]}),w.jsxs("article",{children:[w.jsx("h3",{children:O.doors.budget}),w.jsx("p",{children:O.doors.budgetBody})]})]})]}),w.jsxs("section",{className:"section split wrap",id:"limits","data-zone":"doors",children:[w.jsxs("div",{className:"section-copy",children:[w.jsx("p",{className:"kicker",children:"07"}),w.jsx(Da,{text:O.limits.title}),w.jsx("p",{className:"dek",children:O.limits.dek})]}),w.jsx("ul",{className:"limits",children:O.limits.items.map(F=>w.jsx("li",{children:F},F))})]}),w.jsxs("section",{className:"section split wrap",id:"download","data-zone":"download",children:[w.jsxs("div",{className:"section-copy",children:[w.jsxs("p",{className:"kicker",children:[O.get.version," ",I]}),w.jsx(Da,{text:O.get.title}),w.jsx("p",{className:"dek",children:O.get.dek}),w.jsx("ol",{className:"start",children:O.get.steps.map(F=>w.jsx("li",{children:F},F))})]}),w.jsxs("div",{className:"section-body",children:[w.jsx("div",{className:"downloads",children:L.map(F=>{const ge=F.os===Y;return w.jsxs("a",{className:ge?"asset is-you":"asset",href:F.href,children:[w.jsxs("span",{className:"asset-copy",children:[w.jsx("b",{children:F.label}),w.jsx("span",{className:"asset-name",children:F.name})]}),w.jsxs("span",{className:"asset-meta",children:[ge?w.jsx("em",{children:O.get.thisMachine}):null,F.size?w.jsx("span",{children:F.size}):null]})]},F.os)})}),w.jsx("p",{className:"note",children:O.get.note})]})]})]}),w.jsx("footer",{className:"foot",children:w.jsxs("div",{className:"wrap foot-row",children:[w.jsxs("a",{className:"brand",href:"#top",children:[w.jsx(_x,{}),w.jsx("span",{children:"yoyo"})]}),w.jsxs("nav",{children:[w.jsx("a",{href:gs,children:"GitHub"}),w.jsx("a",{href:hh,children:"Releases"}),w.jsx("a",{href:`${gs}/blob/master/docs/PRODUCT.md`,children:O.foot.product}),w.jsx("a",{href:`${gs}/blob/master/docs/architecture/invariants.md`,children:O.foot.invariants}),w.jsx("a",{href:`${gs}/issues`,children:O.foot.issues}),w.jsx("a",{href:`${gs}/blob/master/LICENSE`,children:O.foot.license})]}),w.jsxs("p",{children:[w.jsx("span",{children:O.foot.stack}),w.jsx("span",{children:"© 2026"})]})]})})]})}function Da({text:o,as:t="h2",id:i}){return w.jsx(t,{id:i,children:o.split(`
`).map((r,c)=>w.jsx("span",{className:"ln",children:r},c))})}function Rw({text:o}){return w.jsx(w.Fragment,{children:o.split("`").map((t,i)=>i%2?w.jsx("code",{children:t},i):w.jsx("span",{children:t},i))})}function gx({lang:o,onPick:t}){return w.jsx("div",{className:"lang",role:"group","aria-label":"Language",children:[["en","EN","English"],["zh","中","简体中文"],["ja","日","日本語"]].map(([i,r,c])=>w.jsx("button",{type:"button","aria-label":c,"aria-pressed":o===i,onClick:()=>t(i),children:r},i))})}function _x(){return w.jsxs("svg",{className:"brand-mark",viewBox:"0 0 32 32",width:"18",height:"18","aria-hidden":"true",children:[w.jsx("rect",{width:"32",height:"32",rx:"8",fill:"currentColor"}),w.jsxs("g",{fill:"none",stroke:"var(--mark-cut)",strokeLinecap:"round",children:[w.jsx("circle",{cx:"16",cy:"10.2",r:"4",strokeWidth:"2.2"}),w.jsx("path",{d:"M8.2 16.4 16 21.2 23.8 16.4",strokeWidth:"2.4"}),w.jsx("path",{d:"M16 21.2v5.2",strokeWidth:"2.4"})]})]})}function vx(){return w.jsx("svg",{viewBox:"0 0 16 16",width:"14",height:"14","aria-hidden":"true",children:w.jsx("path",{d:"M8 2.2v7.2M5.2 7.2 8 10l2.8-2.8M3.2 13.2h9.6",fill:"none",stroke:"currentColor",strokeWidth:"1.6",strokeLinecap:"round",strokeLinejoin:"round"})})}function ww(){return w.jsxs("svg",{viewBox:"0 0 24 24",width:"18",height:"18","aria-hidden":"true",children:[w.jsx("circle",{cx:"12",cy:"12",r:"3.5",fill:"none",stroke:"currentColor",strokeWidth:"1.6"}),w.jsx("path",{d:"M12 3v2.2M12 18.8V21M3 12h2.2M18.8 12H21M5.4 5.4l1.6 1.6M17 17l1.6 1.6M18.6 5.4 17 7M7 17l-1.6 1.6",fill:"none",stroke:"currentColor",strokeWidth:"1.6",strokeLinecap:"round"})]})}function Cw(){return w.jsx("svg",{viewBox:"0 0 24 24",width:"18",height:"18","aria-hidden":"true",children:w.jsx("path",{d:"M15.2 3.4A7.4 7.4 0 1 0 20.6 14 5.8 5.8 0 0 1 15.2 3.4Z",fill:"none",stroke:"currentColor",strokeWidth:"1.6",strokeLinejoin:"round"})})}function Nw({open:o}){return w.jsx("svg",{viewBox:"0 0 24 24",width:"18",height:"18","aria-hidden":"true",children:o?w.jsx("path",{d:"M6 6l12 12M18 6 6 18",fill:"none",stroke:"currentColor",strokeWidth:"1.6",strokeLinecap:"round"}):w.jsx("path",{d:"M4 8h16M4 16h16",fill:"none",stroke:"currentColor",strokeWidth:"1.6",strokeLinecap:"round"})})}qM.createRoot(document.getElementById("root")).render(w.jsx(Gt.StrictMode,{children:w.jsx(Aw,{})}));
