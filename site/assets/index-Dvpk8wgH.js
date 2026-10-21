(function(){const e=document.createElement("link").relList;if(e&&e.supports&&e.supports("modulepreload"))return;for(const c of document.querySelectorAll('link[rel="modulepreload"]'))r(c);new MutationObserver(c=>{for(const f of c)if(f.type==="childList")for(const d of f.addedNodes)d.tagName==="LINK"&&d.rel==="modulepreload"&&r(d)}).observe(document,{childList:!0,subtree:!0});function i(c){const f={};return c.integrity&&(f.integrity=c.integrity),c.referrerPolicy&&(f.referrerPolicy=c.referrerPolicy),c.crossOrigin==="use-credentials"?f.credentials="include":c.crossOrigin==="anonymous"?f.credentials="omit":f.credentials="same-origin",f}function r(c){if(c.ep)return;c.ep=!0;const f=i(c);fetch(c.href,f)}})();var xh={exports:{}},pl={};/**
 * @license React
 * react-jsx-runtime.production.js
 *
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var T_;function aE(){if(T_)return pl;T_=1;var o=Symbol.for("react.transitional.element"),e=Symbol.for("react.fragment");function i(r,c,f){var d=null;if(f!==void 0&&(d=""+f),c.key!==void 0&&(d=""+c.key),"key"in c){f={};for(var h in c)h!=="key"&&(f[h]=c[h])}else f=c;return c=f.ref,{$$typeof:o,type:r,key:d,ref:c!==void 0?c:null,props:f}}return pl.Fragment=e,pl.jsx=i,pl.jsxs=i,pl}var A_;function sE(){return A_||(A_=1,xh.exports=aE()),xh.exports}var w=sE(),Sh={exports:{}},re={};/**
 * @license React
 * react.production.js
 *
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var R_;function rE(){if(R_)return re;R_=1;var o=Symbol.for("react.transitional.element"),e=Symbol.for("react.portal"),i=Symbol.for("react.fragment"),r=Symbol.for("react.strict_mode"),c=Symbol.for("react.profiler"),f=Symbol.for("react.consumer"),d=Symbol.for("react.context"),h=Symbol.for("react.forward_ref"),m=Symbol.for("react.suspense"),p=Symbol.for("react.memo"),x=Symbol.for("react.lazy"),v=Symbol.for("react.activity"),_=Symbol.for("react.view_transition"),M=Symbol.iterator;function R(H){return H===null||typeof H!="object"?null:(H=M&&H[M]||H["@@iterator"],typeof H=="function"?H:null)}var N={isMounted:function(){return!1},enqueueForceUpdate:function(){},enqueueReplaceState:function(){},enqueueSetState:function(){}},y=Object.assign,S={};function O(H,F,gt){this.props=H,this.context=F,this.refs=S,this.updater=gt||N}O.prototype.isReactComponent={},O.prototype.setState=function(H,F){if(typeof H!="object"&&typeof H!="function"&&H!=null)throw Error("takes an object of state variables to update or a function which returns an object of state variables.");this.updater.enqueueSetState(this,H,F,"setState")},O.prototype.forceUpdate=function(H){this.updater.enqueueForceUpdate(this,H,"forceUpdate")};function k(){}k.prototype=O.prototype;function C(H,F,gt){this.props=H,this.context=F,this.refs=S,this.updater=gt||N}var D=C.prototype=new k;D.constructor=C,y(D,O.prototype),D.isPureReactComponent=!0;var U=Array.isArray;function B(){}var b={H:null,A:null,T:null,S:null},L=Object.prototype.hasOwnProperty;function X(H,F,gt){var G=gt.ref;return{$$typeof:o,type:H,key:F,ref:G!==void 0?G:null,props:gt}}function j(H,F){return X(H.type,F,H.props)}function J(H){return typeof H=="object"&&H!==null&&H.$$typeof===o}function P(H){var F={"=":"=0",":":"=2"};return"$"+H.replace(/[=:]/g,function(gt){return F[gt]})}var q=/\/+/g;function $(H,F){return typeof H=="object"&&H!==null&&H.key!=null?P(""+H.key):F.toString(36)}function Y(H){switch(H.status){case"fulfilled":return H.value;case"rejected":throw H.reason;default:switch(typeof H.status=="string"?H.then(B,B):(H.status="pending",H.then(function(F){H.status==="pending"&&(H.status="fulfilled",H.value=F)},function(F){H.status==="pending"&&(H.status="rejected",H.reason=F)})),H.status){case"fulfilled":return H.value;case"rejected":throw H.reason}}throw H}function Q(H,F,gt,G,st){var At=typeof H;(At==="undefined"||At==="boolean")&&(H=null);var Dt=!1;if(H===null)Dt=!0;else switch(At){case"bigint":case"string":case"number":Dt=!0;break;case"object":switch(H.$$typeof){case o:case e:Dt=!0;break;case x:return Dt=H._init,Q(Dt(H._payload),F,gt,G,st)}}if(Dt)return st=st(H),Dt=G===""?"."+$(H,0):G,U(st)?(gt="",Dt!=null&&(gt=Dt.replace(q,"$&/")+"/"),Q(st,F,gt,"",function(Fe){return Fe})):st!=null&&(J(st)&&(st=j(st,gt+(st.key==null||H&&H.key===st.key?"":(""+st.key).replace(q,"$&/")+"/")+Dt)),F.push(st)),1;Dt=0;var xt=G===""?".":G+":";if(U(H))for(var Ct=0;Ct<H.length;Ct++)G=H[Ct],At=xt+$(G,Ct),Dt+=Q(G,F,gt,At,st);else if(Ct=R(H),typeof Ct=="function")for(H=Ct.call(H),Ct=0;!(G=H.next()).done;)G=G.value,At=xt+$(G,Ct++),Dt+=Q(G,F,gt,At,st);else if(At==="object"){if(typeof H.then=="function")return Q(Y(H),F,gt,G,st);throw F=String(H),Error("Objects are not valid as a React child (found: "+(F==="[object Object]"?"object with keys {"+Object.keys(H).join(", ")+"}":F)+"). If you meant to render a collection of children, use an array instead.")}return Dt}function pt(H,F,gt){if(H==null)return H;var G=[],st=0;return Q(H,G,"","",function(At){return F.call(gt,At,st++)}),G}function ot(H){if(H._status===-1){var F=H._result,gt=F();gt.then(function(G){(H._status===0||H._status===-1)&&(H._status=1,H._result=G,gt.status===void 0&&(gt.status="fulfilled",gt.value=G))},function(G){(H._status===0||H._status===-1)&&(H._status=2,H._result=G,gt.status===void 0&&(gt.status="rejected",gt.reason=G))}),H._status===-1&&(H._status=0,H._result=gt)}if(H._status===1)return H._result.default;throw H._result}var vt=typeof reportError=="function"?reportError:function(H){if(typeof window=="object"&&typeof window.ErrorEvent=="function"){var F=new window.ErrorEvent("error",{bubbles:!0,cancelable:!0,message:typeof H=="object"&&H!==null&&typeof H.message=="string"?String(H.message):String(H),error:H});if(!window.dispatchEvent(F))return}else if(typeof process=="object"&&typeof process.emit=="function"){process.emit("uncaughtException",H);return}console.error(H)};function Tt(H){var F=b.T,gt={};gt.types=F!==null?F.types:null,b.T=gt;try{var G=H(),st=b.S;st!==null&&st(gt,G),typeof G=="object"&&G!==null&&typeof G.then=="function"&&G.then(B,vt)}catch(At){vt(At)}finally{F!==null&&gt.types!==null&&(F.types=gt.types),b.T=F}}function Zt(H){var F=b.T;if(F!==null){var gt=F.types;gt===null?F.types=[H]:gt.indexOf(H)===-1&&gt.push(H)}else Tt(Zt.bind(null,H))}var Yt={map:pt,forEach:function(H,F,gt){pt(H,function(){F.apply(this,arguments)},gt)},count:function(H){var F=0;return pt(H,function(){F++}),F},toArray:function(H){return pt(H,function(F){return F})||[]},only:function(H){if(!J(H))throw Error("React.Children.only expected to receive a single React element child.");return H}};return re.Activity=v,re.Children=Yt,re.Component=O,re.Fragment=i,re.Profiler=c,re.PureComponent=C,re.StrictMode=r,re.Suspense=m,re.ViewTransition=_,re.__CLIENT_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE=b,re.__COMPILER_RUNTIME={__proto__:null,c:function(H){return b.H.useMemoCache(H)}},re.addTransitionType=Zt,re.cache=function(H){return function(){return H.apply(null,arguments)}},re.cacheSignal=function(){return null},re.cloneElement=function(H,F,gt){if(H==null)throw Error("The argument must be a React element, but you passed "+H+".");var G=y({},H.props),st=H.key;if(F!=null)for(At in F.key!==void 0&&(st=""+F.key),F)!L.call(F,At)||At==="key"||At==="__self"||At==="__source"||At==="ref"&&F.ref===void 0||(G[At]=F[At]);var At=arguments.length-2;if(At===1)G.children=gt;else if(1<At){for(var Dt=Array(At),xt=0;xt<At;xt++)Dt[xt]=arguments[xt+2];G.children=Dt}return X(H.type,st,G)},re.createContext=function(H){return H={$$typeof:d,_currentValue:H,_currentValue2:H,_threadCount:0,Provider:null,Consumer:null},H.Provider=H,H.Consumer={$$typeof:f,_context:H},H},re.createElement=function(H,F,gt){var G,st={},At=null;if(F!=null)for(G in F.key!==void 0&&(At=""+F.key),F)L.call(F,G)&&G!=="key"&&G!=="__self"&&G!=="__source"&&(st[G]=F[G]);var Dt=arguments.length-2;if(Dt===1)st.children=gt;else if(1<Dt){for(var xt=Array(Dt),Ct=0;Ct<Dt;Ct++)xt[Ct]=arguments[Ct+2];st.children=xt}if(H&&H.defaultProps)for(G in Dt=H.defaultProps,Dt)st[G]===void 0&&(st[G]=Dt[G]);return X(H,At,st)},re.createRef=function(){return{current:null}},re.forwardRef=function(H){return{$$typeof:h,render:H}},re.isValidElement=J,re.lazy=function(H){return{$$typeof:x,_payload:{_status:-1,_result:H},_init:ot}},re.memo=function(H,F){return{$$typeof:p,type:H,compare:F===void 0?null:F}},re.startTransition=Tt,re.unstable_useCacheRefresh=function(){return b.H.useCacheRefresh()},re.use=function(H){return b.H.use(H)},re.useActionState=function(H,F,gt){return b.H.useActionState(H,F,gt)},re.useCallback=function(H,F){return b.H.useCallback(H,F)},re.useContext=function(H){return b.H.useContext(H)},re.useDebugValue=function(){},re.useDeferredValue=function(H,F){return b.H.useDeferredValue(H,F)},re.useEffect=function(H,F){return b.H.useEffect(H,F)},re.useEffectEvent=function(H){return b.H.useEffectEvent(H)},re.useId=function(){return b.H.useId()},re.useImperativeHandle=function(H,F,gt){return b.H.useImperativeHandle(H,F,gt)},re.useInsertionEffect=function(H,F){return b.H.useInsertionEffect(H,F)},re.useLayoutEffect=function(H,F){return b.H.useLayoutEffect(H,F)},re.useMemo=function(H,F){return b.H.useMemo(H,F)},re.useOptimistic=function(H,F){return b.H.useOptimistic(H,F)},re.useReducer=function(H,F,gt){return b.H.useReducer(H,F,gt)},re.useRef=function(H){return b.H.useRef(H)},re.useState=function(H){return b.H.useState(H)},re.useSyncExternalStore=function(H,F,gt){return b.H.useSyncExternalStore(H,F,gt)},re.useTransition=function(){return b.H.useTransition()},re.version="19.3.0",re}var w_;function Kp(){return w_||(w_=1,Sh.exports=rE()),Sh.exports}var Ae=Kp(),yh={exports:{}},ml={},Mh={exports:{}},Eh={};/**
 * @license React
 * scheduler.production.js
 *
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var C_;function oE(){return C_||(C_=1,(function(o){function e(Y,Q){var pt=Y.length;Y.push(Q);t:for(;0<pt;){var ot=pt-1>>>1,vt=Y[ot];if(0<c(vt,Q))Y[ot]=Q,Y[pt]=vt,pt=ot;else break t}}function i(Y){return Y.length===0?null:Y[0]}function r(Y){if(Y.length===0)return null;var Q=Y[0],pt=Y.pop();if(pt!==Q){Y[0]=pt;t:for(var ot=0,vt=Y.length,Tt=vt>>>1;ot<Tt;){var Zt=2*(ot+1)-1,Yt=Y[Zt],H=Zt+1,F=Y[H];if(0>c(Yt,pt))H<vt&&0>c(F,Yt)?(Y[ot]=F,Y[H]=pt,ot=H):(Y[ot]=Yt,Y[Zt]=pt,ot=Zt);else if(H<vt&&0>c(F,pt))Y[ot]=F,Y[H]=pt,ot=H;else break t}}return Q}function c(Y,Q){var pt=Y.sortIndex-Q.sortIndex;return pt!==0?pt:Y.id-Q.id}if(o.unstable_now=void 0,typeof performance=="object"&&typeof performance.now=="function"){var f=performance;o.unstable_now=function(){return f.now()}}else{var d=Date,h=d.now();o.unstable_now=function(){return d.now()-h}}var m=[],p=[],x=1,v=null,_=3,M=!1,R=!1,N=!1,y=!1,S=typeof setTimeout=="function"?setTimeout:null,O=typeof clearTimeout=="function"?clearTimeout:null,k=typeof setImmediate<"u"?setImmediate:null;function C(Y){for(var Q=i(p);Q!==null;){if(Q.callback===null)r(p);else if(Q.startTime<=Y)r(p),Q.sortIndex=Q.expirationTime,e(m,Q);else break;Q=i(p)}}function D(Y){if(N=!1,C(Y),!R)if(i(m)!==null)R=!0,U||(U=!0,J());else{var Q=i(p);Q!==null&&$(D,Q.startTime-Y)}}var U=!1,B=-1,b=5,L=-1;function X(){return y?!0:!(o.unstable_now()-L<b)}function j(){if(y=!1,U){var Y=o.unstable_now();L=Y;var Q=!0;try{t:{R=!1,N&&(N=!1,O(B),B=-1),M=!0;var pt=_;try{e:{for(C(Y),v=i(m);v!==null&&!(v.expirationTime>Y&&X());){var ot=v.callback;if(typeof ot=="function"){v.callback=null,_=v.priorityLevel;var vt=ot(v.expirationTime<=Y);if(Y=o.unstable_now(),typeof vt=="function"){v.callback=vt,C(Y),Q=!0;break e}v===i(m)&&r(m),C(Y)}else r(m);v=i(m)}if(v!==null)Q=!0;else{var Tt=i(p);Tt!==null&&$(D,Tt.startTime-Y),Q=!1}}break t}finally{v=null,_=pt,M=!1}Q=void 0}}finally{Q?J():U=!1}}}var J;if(typeof k=="function")J=function(){k(j)};else if(typeof MessageChannel<"u"){var P=new MessageChannel,q=P.port2;P.port1.onmessage=j,J=function(){q.postMessage(null)}}else J=function(){S(j,0)};function $(Y,Q){B=S(function(){Y(o.unstable_now())},Q)}o.unstable_IdlePriority=5,o.unstable_ImmediatePriority=1,o.unstable_LowPriority=4,o.unstable_NormalPriority=3,o.unstable_Profiling=null,o.unstable_UserBlockingPriority=2,o.unstable_cancelCallback=function(Y){Y.callback=null},o.unstable_forceFrameRate=function(Y){0>Y||125<Y?console.error("forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported"):b=0<Y?Math.floor(1e3/Y):5},o.unstable_getCurrentPriorityLevel=function(){return _},o.unstable_next=function(Y){switch(_){case 1:case 2:case 3:var Q=3;break;default:Q=_}var pt=_;_=Q;try{return Y()}finally{_=pt}},o.unstable_requestPaint=function(){y=!0},o.unstable_runWithPriority=function(Y,Q){switch(Y){case 1:case 2:case 3:case 4:case 5:break;default:Y=3}var pt=_;_=Y;try{return Q()}finally{_=pt}},o.unstable_scheduleCallback=function(Y,Q,pt){var ot=o.unstable_now();switch(typeof pt=="object"&&pt!==null?(pt=pt.delay,pt=typeof pt=="number"&&0<pt?ot+pt:ot):pt=ot,Y){case 1:var vt=-1;break;case 2:vt=250;break;case 5:vt=1073741823;break;case 4:vt=1e4;break;default:vt=5e3}return vt=pt+vt,Y={id:x++,callback:Q,priorityLevel:Y,startTime:pt,expirationTime:vt,sortIndex:-1},pt>ot?(Y.sortIndex=pt,e(p,Y),i(m)===null&&Y===i(p)&&(N?(O(B),B=-1):N=!0,$(D,pt-ot))):(Y.sortIndex=vt,e(m,Y),R||M||(R=!0,U||(U=!0,J()))),Y},o.unstable_shouldYield=X,o.unstable_wrapCallback=function(Y){var Q=_;return function(){var pt=_;_=Q;try{return Y.apply(this,arguments)}finally{_=pt}}}})(Eh)),Eh}var N_;function lE(){return N_||(N_=1,Mh.exports=oE()),Mh.exports}var bh={exports:{}},Ln={};/**
 * @license React
 * react-dom.production.js
 *
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var D_;function cE(){if(D_)return Ln;D_=1;var o=Kp();function e(x){var v="https://react.dev/errors/"+x;if(1<arguments.length){v+="?args[]="+encodeURIComponent(arguments[1]);for(var _=2;_<arguments.length;_++)v+="&args[]="+encodeURIComponent(arguments[_])}return"Minified React error #"+x+"; visit "+v+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}function i(){}var r={d:{f:i,r:function(){throw Error(e(522))},D:i,C:i,L:i,m:i,X:i,S:i,M:i},p:0,findDOMNode:null},c=Symbol.for("react.portal"),f=Symbol.for("react.recoverable"),d=Symbol.for("react.optimistic_key");function h(x,v,_){var M=3<arguments.length&&arguments[3]!==void 0?arguments[3]:null;return{$$typeof:c,key:M==null?null:M===d?d:""+M,children:x,containerInfo:v,implementation:_}}var m=o.__CLIENT_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE;function p(x,v){if(x==="font")return"";if(typeof v=="string")return v==="use-credentials"?v:""}return Ln.__DOM_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE=r,Ln.browser=function(x){return{$$typeof:f,_reason:x}},Ln.createPortal=function(x,v){var _=2<arguments.length&&arguments[2]!==void 0?arguments[2]:null;if(!v||v.nodeType!==1&&v.nodeType!==9&&v.nodeType!==11)throw Error(e(299));return h(x,v,null,_)},Ln.flushSync=function(x){var v=m.T,_=r.p;try{if(m.T=null,r.p=2,x)return x()}finally{m.T=v,r.p=_,r.d.f()}},Ln.preconnect=function(x,v){typeof x=="string"&&(v?(v=v.crossOrigin,v=typeof v=="string"?v==="use-credentials"?v:"":void 0):v=null,r.d.C(x,v))},Ln.prefetchDNS=function(x){typeof x=="string"&&r.d.D(x)},Ln.preinit=function(x,v){if(typeof x=="string"&&v&&typeof v.as=="string"){var _=v.as,M=p(_,v.crossOrigin),R=typeof v.integrity=="string"?v.integrity:void 0,N=typeof v.fetchPriority=="string"?v.fetchPriority:void 0;_==="style"?r.d.S(x,typeof v.precedence=="string"?v.precedence:void 0,{crossOrigin:M,integrity:R,fetchPriority:N}):_==="script"&&r.d.X(x,{crossOrigin:M,integrity:R,fetchPriority:N,nonce:typeof v.nonce=="string"?v.nonce:void 0})}},Ln.preinitModule=function(x,v){if(typeof x=="string")if(typeof v=="object"&&v!==null){if(v.as==null||v.as==="script"){var _=p(v.as,v.crossOrigin);r.d.M(x,{crossOrigin:_,integrity:typeof v.integrity=="string"?v.integrity:void 0,nonce:typeof v.nonce=="string"?v.nonce:void 0,fetchPriority:typeof v.fetchPriority=="string"?v.fetchPriority:void 0})}}else v==null&&r.d.M(x)},Ln.preload=function(x,v){if(typeof x=="string"&&typeof v=="object"&&v!==null&&typeof v.as=="string"){var _=v.as,M=p(_,v.crossOrigin);r.d.L(x,_,{crossOrigin:M,integrity:typeof v.integrity=="string"?v.integrity:void 0,nonce:typeof v.nonce=="string"?v.nonce:void 0,type:typeof v.type=="string"?v.type:void 0,fetchPriority:typeof v.fetchPriority=="string"?v.fetchPriority:void 0,referrerPolicy:typeof v.referrerPolicy=="string"?v.referrerPolicy:void 0,imageSrcSet:typeof v.imageSrcSet=="string"?v.imageSrcSet:void 0,imageSizes:typeof v.imageSizes=="string"?v.imageSizes:void 0,media:typeof v.media=="string"?v.media:void 0})}},Ln.preloadModule=function(x,v){if(typeof x=="string")if(v){var _=p(v.as,v.crossOrigin);r.d.m(x,{as:typeof v.as=="string"&&v.as!=="script"?v.as:void 0,crossOrigin:_,integrity:typeof v.integrity=="string"?v.integrity:void 0,nonce:typeof v.nonce=="string"?v.nonce:void 0,fetchPriority:typeof v.fetchPriority=="string"?v.fetchPriority:void 0})}else r.d.m(x)},Ln.requestFormReset=function(x){r.d.r(x)},Ln.unstable_batchedUpdates=function(x,v){return x(v)},Ln.useFormState=function(x,v,_){return m.H.useFormState(x,v,_)},Ln.useFormStatus=function(){return m.H.useHostTransitionStatus()},Ln.version="19.3.0",Ln}var U_;function uE(){if(U_)return bh.exports;U_=1;function o(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(o)}catch(e){console.error(e)}}return o(),bh.exports=cE(),bh.exports}/**
 * @license React
 * react-dom-client.production.js
 *
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var L_;function fE(){if(L_)return ml;L_=1;var o=lE(),e=Kp(),i=uE();function r(t){var n="https://react.dev/errors/"+t;if(1<arguments.length){n+="?args[]="+encodeURIComponent(arguments[1]);for(var a=2;a<arguments.length;a++)n+="&args[]="+encodeURIComponent(arguments[a])}return"Minified React error #"+t+"; visit "+n+" for the full message or use the non-minified dev environment for full errors and additional helpful warnings."}function c(t){return!(!t||t.nodeType!==1&&t.nodeType!==9&&t.nodeType!==11)}function f(t){for(var n=t,a=n;a&&!a.alternate;)n=a,(n.flags&4098)!==0&&(t=n.return),a=n.return;for(;n.return;)n=n.return;return n.tag===3?t:null}function d(t){if(t.tag===13){var n=t.memoizedState;if(n===null&&(t=t.alternate,t!==null&&(n=t.memoizedState)),n!==null)return n.dehydrated}return null}function h(t){if(t.tag===31){var n=t.memoizedState;if(n===null&&(t=t.alternate,t!==null&&(n=t.memoizedState)),n!==null)return n.dehydrated}return null}function m(t){if(f(t)!==t)throw Error(r(188))}function p(t){var n=t.alternate;if(!n){if(n=f(t),n===null)throw Error(r(188));return n!==t?null:t}for(var a=t,s=n;;){var l=a.return;if(l===null)break;var u=l.alternate;if(u===null){if(s=l.return,s!==null){a=s;continue}break}if(l.child===u.child){for(u=l.child;u;){if(u===a)return m(l),t;if(u===s)return m(l),n;u=u.sibling}throw Error(r(188))}if(a.return!==s.return)a=l,s=u;else{for(var g=!1,A=l.child;A;){if(A===a){g=!0,a=l,s=u;break}if(A===s){g=!0,s=l,a=u;break}A=A.sibling}if(!g){for(A=u.child;A;){if(A===a){g=!0,a=u,s=l;break}if(A===s){g=!0,s=u,a=l;break}A=A.sibling}if(!g)throw Error(r(189))}}if(a.alternate!==s)throw Error(r(190))}if(a.tag!==3)throw Error(r(188));return a.stateNode.current===a?t:n}function x(t){var n=t.tag;if(n===5||n===26||n===27||n===6)return t;for(t=t.child;t!==null;){if(n=x(t),n!==null)return n;t=t.sibling}return null}function v(t,n,a,s,l,u){for(;t!==null;){if((t.tag===5||t.tag===27||t.tag===6)&&a(t,s,l,u)||(t.tag!==22||t.memoizedState===null)&&(n||t.tag!==5&&t.tag!==27)&&v(t.child,n,a,s,l,u))return!0;t=t.sibling}return!1}function _(t){for(t=t.return;t!==null;){if(t.tag===3||t.tag===5||t.tag===27)return t;t=t.return}return null}function M(t){var n=!1;for(t=t.return;t!==null&&(t.tag===4&&(n=!0),!(t.tag===3||t.tag===5||t.tag===27));)t=t.return;return n}function R(t){var n=[null,null],a=_(t);return a===null||N(n,t,a.child,{foundSelf:!1}),n}function N(t,n,a,s){for(;a!==null;){if(a===n)s.foundSelf=!0;else if(a.tag===5||a.tag===27||a.tag===6){if(s.foundSelf)return t[1]=a,!0;t[0]=a}else if((a.tag!==22||a.memoizedState===null)&&N(t,n,a.child,s))return!0;a=a.sibling}return!1}function y(t){switch(t.tag){case 5:case 27:case 6:return t.stateNode;case 3:return t.stateNode.containerInfo;default:throw Error(r(559))}}var S=null,O=null;function k(t,n,a){return t===a?!0:t===n?(S=t,!0):!1}function C(t,n,a){return t===a?(O=t,!1):t===n?(O!==null&&(S=t),!0):!1}function D(t){if(t===null)return null;do t=t===null?null:t.return;while(t&&t.tag!==5&&t.tag!==27&&t.tag!==3);return t||null}function U(t,n,a){for(var s=0,l=t;l;l=a(l))s++;l=0;for(var u=n;u;u=a(u))l++;for(;0<s-l;)t=a(t),s--;for(;0<l-s;)n=a(n),l--;for(;s--;){if(t===n||n!==null&&t===n.alternate)return t;t=a(t),n=a(n)}return null}var B=Object.assign,b=Symbol.for("react.element"),L=Symbol.for("react.transitional.element"),X=Symbol.for("react.portal"),j=Symbol.for("react.fragment"),J=Symbol.for("react.strict_mode"),P=Symbol.for("react.profiler"),q=Symbol.for("react.consumer"),$=Symbol.for("react.context"),Y=Symbol.for("react.forward_ref"),Q=Symbol.for("react.suspense"),pt=Symbol.for("react.suspense_list"),ot=Symbol.for("react.memo"),vt=Symbol.for("react.lazy"),Tt=Symbol.for("react.activity"),Zt=Symbol.for("react.legacy_hidden"),Yt=Symbol.for("react.memo_cache_sentinel"),H=Symbol.for("react.view_transition"),F=Symbol.for("react.recoverable"),gt=Symbol.iterator;function G(t){return t===null||typeof t!="object"?null:(t=gt&&t[gt]||t["@@iterator"],typeof t=="function"?t:null)}var st=Symbol.for("react.client.reference");function At(t){if(t==null)return null;if(typeof t=="function")return t.$$typeof===st?null:t.displayName||t.name||null;if(typeof t=="string")return t;switch(t){case j:return"Fragment";case P:return"Profiler";case J:return"StrictMode";case Q:return"Suspense";case pt:return"SuspenseList";case Tt:return"Activity";case H:return"ViewTransition"}if(typeof t=="object")switch(t.$$typeof){case X:return"Portal";case $:return t.displayName||"Context";case q:return(t._context.displayName||"Context")+".Consumer";case Y:var n=t.render;return t=t.displayName,t||(t=n.displayName||n.name||"",t=t!==""?"ForwardRef("+t+")":"ForwardRef"),t;case ot:return n=t.displayName||null,n!==null?n:At(t.type)||"Memo";case vt:n=t._payload,t=t._init;try{return At(t(n))}catch{}}return null}var Dt=Array.isArray,xt=e.__CLIENT_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE,Ct=i.__DOM_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE,Fe={pending:!1,data:null,method:null,action:null},le=[],pe=-1;function ve(t){return{current:t}}function te(t){0>pe||(t.current=le[pe],le[pe]=null,pe--)}function ne(t,n){pe++,le[pe]=t.current,t.current=n}var Be=ve(null),sn=ve(null),Ue=ve(null),je=ve(null);function Z(t,n){switch(ne(Ue,n),ne(sn,t),ne(Be,null),n.nodeType){case 9:case 11:t=(t=n.documentElement)&&(t=t.namespaceURI)?Ov(t):0;break;default:if(t=n.tagName,n=n.namespaceURI)n=Ov(n),t=Pv(n,t);else switch(t){case"svg":t=1;break;case"math":t=2;break;default:t=0}}te(Be),ne(Be,t)}function Je(){te(Be),te(sn),te(Ue)}function Me(t){var n=t.memoizedState;n!==null&&(kr._currentValue=n.memoizedState,ne(je,t)),n=Be.current;var a=Pv(n,t.type);n!==a&&(ne(sn,t),ne(Be,a))}function I(t){sn.current===t&&(te(Be),te(sn)),je.current===t&&(te(je),kr._currentValue=Fe)}var E,at;function lt(t){if(E===void 0)try{throw Error()}catch(a){var n=a.stack.trim().match(/\n( *(at )?)/);E=n&&n[1]||"",at=-1<a.stack.indexOf(`
    at`)?" (<anonymous>)":-1<a.stack.indexOf("@")?"@unknown:0:0":""}return`
`+E+t+at}var et=!1;function St(t,n){if(!t||et)return"";et=!0;var a=Error.prepareStackTrace;Error.prepareStackTrace=void 0;try{var s={DetermineComponentFrameRoot:function(){try{if(n){var Et=function(){throw Error()};if(Object.defineProperty(Et.prototype,"props",{set:function(){throw Error()}}),typeof Reflect=="object"&&Reflect.construct){try{Reflect.construct(Et,[])}catch(Pt){var tt=Pt}Reflect.construct(t,[],Et)}else{try{Et.call()}catch(Pt){tt=Pt}Et=!1;try{var ft=Object.getOwnPropertyDescriptor(t.prototype,"props");Object.defineProperty(t.prototype,"props",{configurable:!0,set:function(){throw Error()}}),Et=!0,new t}finally{Et&&(ft!==void 0?Object.defineProperty(t.prototype,"props",ft):delete t.prototype.props)}}}else{try{throw Error()}catch(Pt){tt=Pt}(Et=t())&&typeof Et.catch=="function"&&Et.catch(function(){})}}catch(Pt){if(Pt&&tt&&typeof Pt.stack=="string")return[Pt.stack,tt.stack]}return[null,null]}};s.DetermineComponentFrameRoot.displayName="DetermineComponentFrameRoot";var l=Object.getOwnPropertyDescriptor(s.DetermineComponentFrameRoot,"name");l&&l.configurable&&Object.defineProperty(s.DetermineComponentFrameRoot,"name",{value:"DetermineComponentFrameRoot"});var u=s.DetermineComponentFrameRoot(),g=u[0],A=u[1];if(g&&A){var z=g.split(`
`),it=A.split(`
`);for(l=s=0;s<z.length&&!z[s].includes("DetermineComponentFrameRoot");)s++;for(;l<it.length&&!it[l].includes("DetermineComponentFrameRoot");)l++;if(s===z.length||l===it.length)for(s=z.length-1,l=it.length-1;1<=s&&0<=l&&z[s]!==it[l];)l--;for(;1<=s&&0<=l;s--,l--)if(z[s]!==it[l]){if(s!==1||l!==1)do if(s--,l--,0>l||z[s]!==it[l]){var ht=`
`+z[s].replace(" at new "," at ");return t.displayName&&ht.includes("<anonymous>")&&(ht=ht.replace("<anonymous>",t.displayName)),ht}while(1<=s&&0<=l);break}}}finally{et=!1,Error.prepareStackTrace=a}return(a=t?t.displayName||t.name:"")?lt(a):""}function wt(t,n){switch(t.tag){case 26:case 27:case 5:return lt(t.type);case 16:return lt("Lazy");case 13:return t.child!==n&&n!==null?lt("Suspense Fallback"):lt("Suspense");case 19:return lt("SuspenseList");case 0:case 15:return St(t.type,!1);case 11:return St(t.type.render,!1);case 1:return St(t.type,!0);case 31:return lt("Activity");case 30:return lt("ViewTransition");default:return""}}function _t(t){try{var n="",a=null;do n+=wt(t,a),a=t,t=t.return;while(t);return n}catch(s){return`
Error generating stack: `+s.message+`
`+s.stack}}var yt=Object.prototype.hasOwnProperty,Nt=o.unstable_scheduleCallback,Jt=o.unstable_cancelCallback,Bt=o.unstable_shouldYield,It=o.unstable_requestPaint,kt=o.unstable_now,ie=o.unstable_getCurrentPriorityLevel,ce=o.unstable_ImmediatePriority,K=o.unstable_UserBlockingPriority,Ut=o.unstable_NormalPriority,bt=o.unstable_LowPriority,Lt=o.unstable_IdlePriority,Vt=o.log,Rt=o.unstable_setDisableYieldValue,$t=null,Gt=null;function Le(t){if(typeof Vt=="function"&&Rt(t),Gt&&typeof Gt.setStrictMode=="function")try{Gt.setStrictMode($t,t)}catch{}}var ue=Math.clz32?Math.clz32:Yu,ni=Math.log,mi=Math.LN2;function Yu(t){return t>>>=0,t===0?32:31-(ni(t)/mi|0)|0}var sr=256,Es=262144,Ha=4194304;function pa(t){var n=t&42;if(n!==0)return n;switch(t&-t){case 1:return 1;case 2:return 2;case 4:return 4;case 8:return 8;case 16:return 16;case 32:return 32;case 64:return 64;case 128:return 128;case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:return t&-t;case 262144:case 524288:case 1048576:case 2097152:return t&3932160;case 4194304:case 8388608:case 16777216:case 33554432:return t&62914560;case 67108864:return 67108864;case 134217728:return 134217728;case 268435456:return 268435456;case 536870912:return 536870912;case 1073741824:return 0;default:return t}}function bs(t,n,a){var s=t.pendingLanes;if(s===0)return 0;var l=0,u=t.suspendedLanes,g=t.pingedLanes;t=t.warmLanes;var A=s&134217727;return A!==0?(s=A&~u,s!==0?l=pa(s):(g&=A,g!==0?l=pa(g):a||(a=A&~t,a!==0&&(l=pa(a))))):(A=s&~u,A!==0?l=pa(A):g!==0?l=pa(g):a||(a=s&~t,a!==0&&(l=pa(a)))),l===0?0:n!==0&&n!==l&&(n&u)===0&&(u=l&-l,a=n&-n,u>=a||u===32&&(a&4194048)!==0)?n:l}function Ga(t,n){return(t.pendingLanes&~(t.suspendedLanes&~t.pingedLanes)&n)===0}function Xi(t,n){(n&8)!==0&&(n|=n&32);var a=t.entangledLanes;if(a!==0)for(t=t.entanglements,a&=n;0<a;){var s=31-ue(a),l=1<<s;n|=t[s],a&=~l}return n}function So(t,n){switch(t){case 1:case 2:case 4:case 8:case 64:return n+250;case 16:case 32:case 128:case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:return n+5e3;case 4194304:case 8388608:case 16777216:case 33554432:return-1;case 67108864:case 134217728:case 268435456:case 536870912:case 1073741824:return-1;default:return-1}}function yo(){var t=Ha;return Ha<<=1,(Ha&62914560)===0&&(Ha=4194304),t}function rr(t){for(var n=[],a=0;31>a;a++)n.push(t);return n}function Wi(t,n){t.pendingLanes|=n,n!==268435456&&(t.suspendedLanes=0,t.pingedLanes=0,t.warmLanes=0)}function Fl(t,n,a,s,l,u){var g=t.pendingLanes;t.pendingLanes=a,t.suspendedLanes=0,t.pingedLanes=0,t.warmLanes=0,t.expiredLanes&=a,t.entangledLanes&=a,t.errorRecoveryDisabledLanes&=a,t.shellSuspendCounter=0;var A=t.entanglements,z=t.expirationTimes,it=t.hiddenUpdates;for(a=g&~a;0<a;){var ht=31-ue(a),Et=1<<ht;A[ht]=0,z[ht]=-1;var tt=it[ht];if(tt!==null)for(it[ht]=null,ht=0;ht<tt.length;ht++){var ft=tt[ht];ft!==null&&(ft.lane&=-536870913)}a&=~Et}s!==0&&Ts(t,s,0),u!==0&&l===0&&t.tag!==0&&(t.suspendedLanes|=u&~(g&~n))}function Ts(t,n,a){t.pendingLanes|=n,t.suspendedLanes&=~n;var s=31-ue(n);t.entangledLanes|=n,t.entanglements[s]=t.entanglements[s]|1073741824|a&261930}function Mo(t,n){var a=t.entangledLanes|=n;for(t=t.entanglements;a;){var s=31-ue(a),l=1<<s;l&n|t[s]&n&&(t[s]|=n),a&=~l}}function Eo(t,n){var a=n&-n;return a=(a&42)!==0?1:bo(a),(a&(t.suspendedLanes|n))!==0?0:a}function bo(t){switch(t){case 2:t=1;break;case 8:t=4;break;case 32:t=16;break;case 256:case 512:case 1024:case 2048:case 4096:case 8192:case 16384:case 32768:case 65536:case 131072:case 262144:case 524288:case 1048576:case 2097152:case 4194304:case 8388608:case 16777216:case 33554432:t=128;break;case 268435456:t=134217728;break;default:t=0}return t}function To(t){return t&=-t,2<t?8<t?(t&134217727)!==0?32:268435456:8:2}function Hl(){var t=Ct.p;return t!==0?t:(t=window.event,t===void 0?32:__(t.type))}function Gl(t,n){var a=Ct.p;try{return Ct.p=t,n()}finally{Ct.p=a}}var gi=Math.random().toString(36).slice(2),T="__reactFiber$"+gi,V="__reactProps$"+gi,mt="__reactContainer$"+gi,ct="__reactEvents$"+gi,ut="__reactListeners$"+gi,zt="__reactHandles$"+gi,Xt="__reactResources$"+gi,Ot="__reactMarker$"+gi,jt="__reactLoad$"+gi;function Kt(t){delete t[T],delete t[V],delete t[ut],delete t[zt]}function se(t){var n;if(n=t[T])return n;for(var a=t.parentNode;a;){if(n=a[mt]||a[T]){if(a=n.alternate,n.child!==null||a!==null&&a.child!==null)for(t=Jv(t);t!==null;){if(a=t[T])return a;t=Jv(t)}return n}t=a,a=t.parentNode}return null}function fe(t){if(t=t[T]||t[mt]){var n=t.tag;if(n===5||n===6||n===13||n===31||n===26||n===27||n===3)return t}return null}function Wt(t){var n=t.tag;if(n===5||n===26||n===27||n===6)return t.stateNode;throw Error(r(33))}function Ee(t){var n=t[Xt];return n||(n=t[Xt]={hoistableStyles:new Map,hoistableScripts:new Map}),n}function xe(t){t[Ot]=!0}function Ze(t){t[jt]=void 0}var Ve=new Set,xn={};function Ft(t,n){ln(t,n),ln(t+"Capture",n)}function ln(t,n){for(xn[t]=n,t=0;t<n.length;t++)Ve.add(n[t])}var Oe=RegExp("^[:A-Z_a-z\\u00C0-\\u00D6\\u00D8-\\u00F6\\u00F8-\\u02FF\\u0370-\\u037D\\u037F-\\u1FFF\\u200C-\\u200D\\u2070-\\u218F\\u2C00-\\u2FEF\\u3001-\\uD7FF\\uF900-\\uFDCF\\uFDF0-\\uFFFD][:A-Z_a-z\\u00C0-\\u00D6\\u00D8-\\u00F6\\u00F8-\\u02FF\\u0370-\\u037D\\u037F-\\u1FFF\\u200C-\\u200D\\u2070-\\u218F\\u2C00-\\u2FEF\\u3001-\\uD7FF\\uF900-\\uFDCF\\uFDF0-\\uFFFD\\-.0-9\\u00B7\\u0300-\\u036F\\u203F-\\u2040]*$"),Hn={},ii={};function qi(t){return yt.call(ii,t)?!0:yt.call(Hn,t)?!1:Oe.test(t)?ii[t]=!0:(Hn[t]=!0,!1)}var Se=!1;function He(){var t=Se;return Se=!1,t}function $e(t,n,a){if(qi(n))if(a===null)t.removeAttribute(n);else{switch(typeof a){case"undefined":case"function":case"symbol":t.removeAttribute(n);return;case"boolean":var s=n.toLowerCase().slice(0,5);if(s!=="data-"&&s!=="aria-"){t.removeAttribute(n);return}}t.setAttribute(n,a)}}function ai(t,n,a){if(a===null)t.removeAttribute(n);else{switch(typeof a){case"undefined":case"function":case"symbol":case"boolean":t.removeAttribute(n);return}t.setAttribute(n,a)}}function we(t,n,a,s){if(s===null)t.removeAttribute(a);else{switch(typeof s){case"undefined":case"function":case"symbol":case"boolean":t.removeAttribute(a);return}t.setAttributeNS(n,a,s)}}function cn(t){switch(typeof t){case"bigint":case"boolean":case"number":case"string":case"undefined":return t;case"object":return t;default:return""}}function ma(t){var n=t.type;return(t=t.nodeName)&&t.toLowerCase()==="input"&&(n==="checkbox"||n==="radio")}function Vl(t,n,a){var s=Object.getOwnPropertyDescriptor(t.constructor.prototype,n);if(!t.hasOwnProperty(n)&&typeof s<"u"&&typeof s.get=="function"&&typeof s.set=="function"){var l=s.get,u=s.set;return Object.defineProperty(t,n,{configurable:!0,get:function(){return l.call(this)},set:function(g){a=""+g,u.call(this,g)}}),Object.defineProperty(t,n,{enumerable:s.enumerable}),{getValue:function(){return a},setValue:function(g){a=""+g},stopTracking:function(){t._valueTracker=null,delete t[n]}}}}function ju(t){if(!t._valueTracker){var n=ma(t)?"checked":"value";t._valueTracker=Vl(t,n,""+t[n])}}function Mm(t){if(!t)return!1;var n=t._valueTracker;if(!n)return!0;var a=n.getValue(),s="";return t&&(s=ma(t)?t.checked?"true":"false":t.value),t=s,t!==a?(n.setValue(t),!0):!1}var TS=/[\n"\\]/g;function vi(t){return t.replace(TS,function(n){return"\\"+n.charCodeAt(0).toString(16)+" "})}function Zu(t,n,a,s,l,u,g,A){t.name="",g!=null&&typeof g!="function"&&typeof g!="symbol"&&typeof g!="boolean"?t.type=g:t.removeAttribute("type"),n!=null?g==="number"?(n===0&&t.value===""||t.value!=n)&&(t.value=""+cn(n)):t.value!==""+cn(n)&&(t.value=""+cn(n)):g!=="submit"&&g!=="reset"||t.removeAttribute("value"),n!=null?g==="number"&&t.value==n?Ku(t,cn(t.value)):Ku(t,cn(n)):a!=null?Ku(t,cn(a)):s!=null&&t.removeAttribute("value"),l==null&&u!=null&&(t.defaultChecked=!!u),l!=null&&(t.checked=l&&typeof l!="function"&&typeof l!="symbol"),A!=null&&typeof A!="function"&&typeof A!="symbol"&&typeof A!="boolean"?t.name=""+cn(A):t.removeAttribute("name")}function Em(t,n,a,s,l,u,g,A){if(u!=null&&typeof u!="function"&&typeof u!="symbol"&&typeof u!="boolean"&&(t.type=u),n!=null||a!=null){if(!(u!=="submit"&&u!=="reset"||n!=null)){ju(t);return}a=a!=null?""+cn(a):"",n=n!=null?""+cn(n):a,A||n===t.value||(t.value=n),t.defaultValue=n}s=s??l,s=typeof s!="function"&&typeof s!="symbol"&&!!s,t.checked=A?t.checked:!!s,t.defaultChecked=!!s,g!=null&&typeof g!="function"&&typeof g!="symbol"&&typeof g!="boolean"&&(t.name=g),ju(t)}function Ku(t,n){t.defaultValue!==""+n&&(t.defaultValue=""+n)}function or(t,n,a,s){if(t=t.options,n){n={};for(var l=0;l<a.length;l++)n["$"+a[l]]=!0;for(a=0;a<t.length;a++)l=n.hasOwnProperty("$"+t[a].value),t[a].selected!==l&&(t[a].selected=l),l&&s&&(t[a].defaultSelected=!0)}else{for(a=""+cn(a),n=null,l=0;l<t.length;l++){if(t[l].value===a){t[l].selected=!0,s&&(t[l].defaultSelected=!0);return}n!==null||t[l].disabled||(n=t[l])}n!==null&&(n.selected=!0)}}function bm(t,n,a){if(n!=null&&(n=""+cn(n),n!==t.value&&(t.value=n),a==null)){t.defaultValue!==n&&(t.defaultValue=n);return}t.defaultValue=a!=null?""+cn(a):""}function Tm(t,n,a,s){if(n==null){if(s!=null){if(a!=null)throw Error(r(92));if(Dt(s)){if(1<s.length)throw Error(r(93));s=s[0]}a=s}a==null&&(a=""),n=a}a=cn(n),t.defaultValue=a,s=t.textContent,s===a&&s!==""&&s!==null&&(t.value=s),ju(t)}function lr(t,n){if(n){var a=t.firstChild;if(a&&a===t.lastChild&&a.nodeType===3){a.nodeValue=n;return}}t.textContent=n}var AS=new Set("animationIterationCount aspectRatio borderImageOutset borderImageSlice borderImageWidth boxFlex boxFlexGroup boxOrdinalGroup columnCount columns flex flexGrow flexPositive flexShrink flexNegative flexOrder gridArea gridRow gridRowEnd gridRowSpan gridRowStart gridColumn gridColumnEnd gridColumnSpan gridColumnStart fontWeight lineClamp lineHeight opacity order orphans scale tabSize widows zIndex zoom fillOpacity floodOpacity stopOpacity strokeDasharray strokeDashoffset strokeMiterlimit strokeOpacity strokeWidth MozAnimationIterationCount MozBoxFlex MozBoxFlexGroup MozLineClamp msAnimationIterationCount msFlex msZoom msFlexGrow msFlexNegative msFlexOrder msFlexPositive msFlexShrink msGridColumn msGridColumnSpan msGridRow msGridRowSpan WebkitAnimationIterationCount WebkitBoxFlex WebKitBoxFlexGroup WebkitBoxOrdinalGroup WebkitColumnCount WebkitColumns WebkitFlex WebkitFlexGrow WebkitFlexPositive WebkitFlexShrink WebkitLineClamp".split(" "));function Am(t,n,a){var s=n.indexOf("--")===0;a==null||typeof a=="boolean"||a===""?s?t.setProperty(n,""):n==="float"?t.cssFloat="":t[n]="":s?t.setProperty(n,a):typeof a!="number"||a===0||AS.has(n)?n==="float"?t.cssFloat=a:t[n]=(""+a).trim():t[n]=a+"px"}function Rm(t,n,a){if(n!=null&&typeof n!="object")throw Error(r(62));if(t=t.style,a!=null){for(var s in a)!a.hasOwnProperty(s)||n!=null&&n.hasOwnProperty(s)||(s.indexOf("--")===0?t.setProperty(s,""):s==="float"?t.cssFloat="":t[s]="",Se=!0);for(var l in n)s=n[l],n.hasOwnProperty(l)&&a[l]!==s&&(Am(t,l,s),Se=!0)}else for(var u in n)n.hasOwnProperty(u)&&Am(t,u,n[u])}function Qu(t){if(t.indexOf("-")===-1)return!1;switch(t){case"annotation-xml":case"color-profile":case"font-face":case"font-face-src":case"font-face-uri":case"font-face-format":case"font-face-name":case"missing-glyph":return!1;default:return!0}}var RS=new Map([["acceptCharset","accept-charset"],["htmlFor","for"],["httpEquiv","http-equiv"],["crossOrigin","crossorigin"],["accentHeight","accent-height"],["alignmentBaseline","alignment-baseline"],["arabicForm","arabic-form"],["baselineShift","baseline-shift"],["capHeight","cap-height"],["clipPath","clip-path"],["clipRule","clip-rule"],["colorInterpolation","color-interpolation"],["colorInterpolationFilters","color-interpolation-filters"],["colorProfile","color-profile"],["colorRendering","color-rendering"],["dominantBaseline","dominant-baseline"],["enableBackground","enable-background"],["fillOpacity","fill-opacity"],["fillRule","fill-rule"],["floodColor","flood-color"],["floodOpacity","flood-opacity"],["fontFamily","font-family"],["fontSize","font-size"],["fontSizeAdjust","font-size-adjust"],["fontStretch","font-stretch"],["fontStyle","font-style"],["fontVariant","font-variant"],["fontWeight","font-weight"],["glyphName","glyph-name"],["glyphOrientationHorizontal","glyph-orientation-horizontal"],["glyphOrientationVertical","glyph-orientation-vertical"],["horizAdvX","horiz-adv-x"],["horizOriginX","horiz-origin-x"],["imageRendering","image-rendering"],["letterSpacing","letter-spacing"],["lightingColor","lighting-color"],["markerEnd","marker-end"],["markerMid","marker-mid"],["markerStart","marker-start"],["maskType","mask-type"],["overlinePosition","overline-position"],["overlineThickness","overline-thickness"],["paintOrder","paint-order"],["panose-1","panose-1"],["pointerEvents","pointer-events"],["renderingIntent","rendering-intent"],["shapeRendering","shape-rendering"],["stopColor","stop-color"],["stopOpacity","stop-opacity"],["strikethroughPosition","strikethrough-position"],["strikethroughThickness","strikethrough-thickness"],["strokeDasharray","stroke-dasharray"],["strokeDashoffset","stroke-dashoffset"],["strokeLinecap","stroke-linecap"],["strokeLinejoin","stroke-linejoin"],["strokeMiterlimit","stroke-miterlimit"],["strokeOpacity","stroke-opacity"],["strokeWidth","stroke-width"],["textAnchor","text-anchor"],["textDecoration","text-decoration"],["textRendering","text-rendering"],["transformOrigin","transform-origin"],["underlinePosition","underline-position"],["underlineThickness","underline-thickness"],["unicodeBidi","unicode-bidi"],["unicodeRange","unicode-range"],["unitsPerEm","units-per-em"],["vAlphabetic","v-alphabetic"],["vHanging","v-hanging"],["vIdeographic","v-ideographic"],["vMathematical","v-mathematical"],["vectorEffect","vector-effect"],["vertAdvY","vert-adv-y"],["vertOriginX","vert-origin-x"],["vertOriginY","vert-origin-y"],["wordSpacing","word-spacing"],["writingMode","writing-mode"],["xmlnsXlink","xmlns:xlink"],["xHeight","x-height"]]),wS=/^[\u0000-\u001F ]*j[\r\n\t]*a[\r\n\t]*v[\r\n\t]*a[\r\n\t]*s[\r\n\t]*c[\r\n\t]*r[\r\n\t]*i[\r\n\t]*p[\r\n\t]*t[\r\n\t]*:/i;function kl(t){return wS.test(""+t)?"javascript:throw new Error('React has blocked a javascript: URL as a security precaution.')":t}function Yi(){}var Ju=null;function $u(t){return t=t.target||t.srcElement||window,t.correspondingUseElement&&(t=t.correspondingUseElement),t.nodeType===3?t.parentNode:t}var cr=null,ur=null;function wm(t){var n=fe(t);if(n&&(t=n.stateNode)){var a=t[V]||null;t:switch(t=n.stateNode,n.type){case"input":if(Zu(t,a.value,a.defaultValue,a.defaultValue,a.checked,a.defaultChecked,a.type,a.name),n=a.name,a.type==="radio"&&n!=null){for(a=t;a.parentNode;)a=a.parentNode;for(a=a.querySelectorAll('input[name="'+vi(""+n)+'"][type="radio"]'),n=0;n<a.length;n++){var s=a[n];if(s!==t&&s.form===t.form){var l=s[V]||null;if(!l)throw Error(r(90));Zu(s,l.value,l.defaultValue,l.defaultValue,l.checked,l.defaultChecked,l.type,l.name)}}for(n=0;n<a.length;n++)s=a[n],s.form===t.form&&Mm(s)}break t;case"textarea":bm(t,a.value,a.defaultValue);break t;case"select":n=a.value,n!=null&&or(t,!!a.multiple,n,!1)}}}var tf=!1;function Cm(t,n,a){if(tf)return t(n,a);tf=!0;try{var s=t(n);return s}finally{if(tf=!1,(cr!==null||ur!==null)&&(kc(),cr&&(n=cr,t=ur,ur=cr=null,wm(n),t)))for(n=0;n<t.length;n++)wm(t[n])}}function Ao(t,n){var a=t.stateNode;if(a===null)return null;var s=a[V]||null;if(s===null)return null;a=s[n];t:switch(n){case"onClick":case"onClickCapture":case"onDoubleClick":case"onDoubleClickCapture":case"onMouseDown":case"onMouseDownCapture":case"onMouseMove":case"onMouseMoveCapture":case"onMouseUp":case"onMouseUpCapture":case"onMouseEnter":(s=!s.disabled)||(t=t.type,s=!(t==="button"||t==="input"||t==="select"||t==="textarea")),t=!s;break t;default:t=!1}if(t)return null;if(a&&typeof a!="function")throw Error(r(231,n,typeof a));return a}var ga=!(typeof window>"u"||typeof window.document>"u"||typeof window.document.createElement>"u"),ef=!1;if(ga)try{var Ro={};Object.defineProperty(Ro,"passive",{get:function(){ef=!0}}),window.addEventListener("test",Ro,Ro),window.removeEventListener("test",Ro,Ro)}catch{ef=!1}var Va=null,nf=null,Xl=null;function Nm(){if(Xl)return Xl;var t,n=nf,a=n.length,s,l="value"in Va?Va.value:Va.textContent,u=l.length;for(t=0;t<a&&n[t]===l[t];t++);var g=a-t;for(s=1;s<=g&&n[a-s]===l[u-s];s++);return Xl=l.slice(t,1<s?1-s:void 0)}function Wl(t){var n=t.keyCode;return"charCode"in t?(t=t.charCode,t===0&&n===13&&(t=13)):t=n,t===10&&(t=13),32<=t||t===13?t:0}function ql(){return!0}function Dm(){return!1}function Gn(t){function n(a,s,l,u,g){this._reactName=a,this._targetInst=l,this.type=s,this.nativeEvent=u,this.target=g,this.currentTarget=null;for(var A in t)t.hasOwnProperty(A)&&(a=t[A],this[A]=a?a(u):u[A]);return this.isDefaultPrevented=(u.defaultPrevented!=null?u.defaultPrevented:u.returnValue===!1)?ql:Dm,this.isPropagationStopped=Dm,this}return B(n.prototype,{preventDefault:function(){this.defaultPrevented=!0;var a=this.nativeEvent;a&&(a.preventDefault?a.preventDefault():typeof a.returnValue!="unknown"&&(a.returnValue=!1),this.isDefaultPrevented=ql)},stopPropagation:function(){var a=this.nativeEvent;a&&(a.stopPropagation?a.stopPropagation():typeof a.cancelBubble!="unknown"&&(a.cancelBubble=!0),this.isPropagationStopped=ql)},persist:function(){},isPersistent:ql}),n}var ka={eventPhase:0,bubbles:0,cancelable:0,timeStamp:function(t){return t.timeStamp||Date.now()},defaultPrevented:0,isTrusted:0},Yl=Gn(ka),wo=B({},ka,{view:0,detail:0}),CS=Gn(wo),af,sf,Co,jl=B({},wo,{screenX:0,screenY:0,clientX:0,clientY:0,pageX:0,pageY:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,getModifierState:of,button:0,buttons:0,relatedTarget:function(t){return t.relatedTarget===void 0?t.fromElement===t.srcElement?t.toElement:t.fromElement:t.relatedTarget},movementX:function(t){return"movementX"in t?t.movementX:(t!==Co&&(Co&&t.type==="mousemove"?(af=t.screenX-Co.screenX,sf=t.screenY-Co.screenY):sf=af=0,Co=t),af)},movementY:function(t){return"movementY"in t?t.movementY:sf}}),Um=Gn(jl),NS=B({},jl,{dataTransfer:0}),DS=Gn(NS),US=B({},wo,{relatedTarget:0}),rf=Gn(US),LS=B({},ka,{animationName:0,elapsedTime:0,pseudoElement:0}),OS=Gn(LS),PS=B({},ka,{clipboardData:function(t){return"clipboardData"in t?t.clipboardData:window.clipboardData}}),IS=Gn(PS),BS=B({},ka,{data:0}),Lm=Gn(BS),zS={Esc:"Escape",Spacebar:" ",Left:"ArrowLeft",Up:"ArrowUp",Right:"ArrowRight",Down:"ArrowDown",Del:"Delete",Win:"OS",Menu:"ContextMenu",Apps:"ContextMenu",Scroll:"ScrollLock",MozPrintableKey:"Unidentified"},FS={8:"Backspace",9:"Tab",12:"Clear",13:"Enter",16:"Shift",17:"Control",18:"Alt",19:"Pause",20:"CapsLock",27:"Escape",32:" ",33:"PageUp",34:"PageDown",35:"End",36:"Home",37:"ArrowLeft",38:"ArrowUp",39:"ArrowRight",40:"ArrowDown",45:"Insert",46:"Delete",112:"F1",113:"F2",114:"F3",115:"F4",116:"F5",117:"F6",118:"F7",119:"F8",120:"F9",121:"F10",122:"F11",123:"F12",144:"NumLock",145:"ScrollLock",224:"Meta"},HS={Alt:"altKey",Control:"ctrlKey",Meta:"metaKey",Shift:"shiftKey"};function GS(t){var n=this.nativeEvent;return n.getModifierState?n.getModifierState(t):(t=HS[t])?!!n[t]:!1}function of(){return GS}var VS=B({},wo,{key:function(t){if(t.key){var n=zS[t.key]||t.key;if(n!=="Unidentified")return n}return t.type==="keypress"?(t=Wl(t),t===13?"Enter":String.fromCharCode(t)):t.type==="keydown"||t.type==="keyup"?FS[t.keyCode]||"Unidentified":""},code:0,location:0,ctrlKey:0,shiftKey:0,altKey:0,metaKey:0,repeat:0,locale:0,getModifierState:of,charCode:function(t){return t.type==="keypress"?Wl(t):0},keyCode:function(t){return t.type==="keydown"||t.type==="keyup"?t.keyCode:0},which:function(t){return t.type==="keypress"?Wl(t):t.type==="keydown"||t.type==="keyup"?t.keyCode:0}}),kS=Gn(VS),XS=B({},jl,{pointerId:0,width:0,height:0,pressure:0,tangentialPressure:0,tiltX:0,tiltY:0,twist:0,pointerType:0,isPrimary:0}),Om=Gn(XS),WS=B({},ka,{submitter:0}),qS=Gn(WS),YS=B({},wo,{touches:0,targetTouches:0,changedTouches:0,altKey:0,metaKey:0,ctrlKey:0,shiftKey:0,getModifierState:of}),jS=Gn(YS),ZS=B({},ka,{propertyName:0,elapsedTime:0,pseudoElement:0}),KS=Gn(ZS),QS=B({},jl,{deltaX:function(t){return"deltaX"in t?t.deltaX:"wheelDeltaX"in t?-t.wheelDeltaX:0},deltaY:function(t){return"deltaY"in t?t.deltaY:"wheelDeltaY"in t?-t.wheelDeltaY:"wheelDelta"in t?-t.wheelDelta:0},deltaZ:0,deltaMode:0}),JS=Gn(QS),$S=B({},ka,{newState:0,oldState:0,source:0}),ty=Gn($S),ey=[9,13,27,32],lf=ga&&"CompositionEvent"in window,No=null;ga&&"documentMode"in document&&(No=document.documentMode);var ny=ga&&"TextEvent"in window&&!No,Pm=ga&&(!lf||No&&8<No&&11>=No),Im=" ",Bm=!1;function zm(t,n){switch(t){case"keyup":return ey.indexOf(n.keyCode)!==-1;case"keydown":return n.keyCode!==229;case"keypress":case"mousedown":case"focusout":return!0;default:return!1}}function Fm(t){return t=t.detail,typeof t=="object"&&"data"in t?t.data:null}var fr=!1;function iy(t,n){switch(t){case"compositionend":return Fm(n);case"keypress":return n.which!==32?null:(Bm=!0,Im);case"textInput":return t=n.data,t===Im&&Bm?null:t;default:return null}}function ay(t,n){if(fr)return t==="compositionend"||!lf&&zm(t,n)?(t=Nm(),Xl=nf=Va=null,fr=!1,t):null;switch(t){case"paste":return null;case"keypress":if(!(n.ctrlKey||n.altKey||n.metaKey)||n.ctrlKey&&n.altKey){if(n.char&&1<n.char.length)return n.char;if(n.which)return String.fromCharCode(n.which)}return null;case"compositionend":return Pm&&n.locale!=="ko"?null:n.data;default:return null}}var sy={color:!0,date:!0,datetime:!0,"datetime-local":!0,email:!0,month:!0,number:!0,password:!0,range:!0,search:!0,tel:!0,text:!0,time:!0,url:!0,week:!0};function Hm(t){var n=t&&t.nodeName&&t.nodeName.toLowerCase();return n==="input"?!!sy[t.type]:n==="textarea"}function Gm(t,n,a,s){cr?ur?ur.push(s):ur=[s]:cr=s,n=Zc(n,"onChange"),0<n.length&&(a=new Yl("onChange","change",null,a,s),t.push({event:a,listeners:n}))}var Do=null,Uo=null;function ry(t){wv(t,0)}function Zl(t){var n=Wt(t);if(Mm(n))return t}function Vm(t,n){if(t==="change")return n}var km=!1;if(ga){var cf;if(ga){var uf="oninput"in document;if(!uf){var Xm=document.createElement("div");Xm.setAttribute("oninput","return;"),uf=typeof Xm.oninput=="function"}cf=uf}else cf=!1;km=cf&&(!document.documentMode||9<document.documentMode)}function Wm(){Do&&(Do.detachEvent("onpropertychange",qm),Uo=Do=null)}function qm(t){if(t.propertyName==="value"&&Zl(Uo)){var n=[];Gm(n,Uo,t,$u(t)),Cm(ry,n)}}function oy(t,n,a){t==="focusin"?(Wm(),Do=n,Uo=a,Do.attachEvent("onpropertychange",qm)):t==="focusout"&&Wm()}function ly(t){if(t==="selectionchange"||t==="keyup"||t==="keydown")return Zl(Uo)}function cy(t,n){if(t==="click")return Zl(n)}function uy(t,n){if(t==="input"||t==="change")return Zl(n)}function fy(t,n){return t===n&&(t!==0||1/t===1/n)||t!==t&&n!==n}var si=typeof Object.is=="function"?Object.is:fy;function Lo(t,n){if(si(t,n))return!0;if(typeof t!="object"||t===null||typeof n!="object"||n===null)return!1;var a=Object.keys(t),s=Object.keys(n);if(a.length!==s.length)return!1;for(s=0;s<a.length;s++){var l=a[s];if(!yt.call(n,l)||!si(t[l],n[l]))return!1}return!0}function ff(t){if(t=t||(typeof document<"u"?document:void 0),typeof t>"u")return null;try{return t.activeElement||t.body}catch{return t.body}}function Ym(t){for(;t&&t.firstChild;)t=t.firstChild;return t}function jm(t,n){var a=Ym(t);t=0;for(var s;a;){if(a.nodeType===3){if(s=t+a.textContent.length,t<=n&&s>=n)return{node:a,offset:n-t};t=s}t:{for(;a;){if(a.nextSibling){a=a.nextSibling;break t}a=a.parentNode}a=void 0}a=Ym(a)}}function Zm(t,n){return t&&n?t===n?!0:t&&t.nodeType===3?!1:n&&n.nodeType===3?Zm(t,n.parentNode):"contains"in t?t.contains(n):t.compareDocumentPosition?!!(t.compareDocumentPosition(n)&16):!1:!1}function Km(t){t=t!=null&&t.ownerDocument!=null&&t.ownerDocument.defaultView!=null?t.ownerDocument.defaultView:window;for(var n=ff(t.document);n instanceof t.HTMLIFrameElement;){try{var a=typeof n.contentWindow.location.href=="string"}catch{a=!1}if(a)t=n.contentWindow;else break;n=ff(t.document)}return n}function df(t){var n=t&&t.nodeName&&t.nodeName.toLowerCase();return n&&(n==="input"&&(t.type==="text"||t.type==="search"||t.type==="tel"||t.type==="url"||t.type==="password")||n==="textarea"||t.contentEditable==="true")}var dy=ga&&"documentMode"in document&&11>=document.documentMode,dr=null,hf=null,Oo=null,pf=!1;function Qm(t,n,a){var s=a.window===a?a.document:a.nodeType===9?a:a.ownerDocument;pf||dr==null||dr!==ff(s)||(s=dr,"selectionStart"in s&&df(s)?s={start:s.selectionStart,end:s.selectionEnd}:(s=(s.ownerDocument&&s.ownerDocument.defaultView||window).getSelection(),s={anchorNode:s.anchorNode,anchorOffset:s.anchorOffset,focusNode:s.focusNode,focusOffset:s.focusOffset}),Oo&&Lo(Oo,s)||(Oo=s,s=Zc(hf,"onSelect"),0<s.length&&(n=new Yl("onSelect","select",null,n,a),t.push({event:n,listeners:s}),n.target=dr)))}function As(t,n){var a={};return a[t.toLowerCase()]=n.toLowerCase(),a["Webkit"+t]="webkit"+n,a["Moz"+t]="moz"+n,a}var hr={animationend:As("Animation","AnimationEnd"),animationiteration:As("Animation","AnimationIteration"),animationstart:As("Animation","AnimationStart"),transitionrun:As("Transition","TransitionRun"),transitionstart:As("Transition","TransitionStart"),transitioncancel:As("Transition","TransitionCancel"),transitionend:As("Transition","TransitionEnd")},mf={},Jm={};ga&&(Jm=document.createElement("div").style,"AnimationEvent"in window||(delete hr.animationend.animation,delete hr.animationiteration.animation,delete hr.animationstart.animation),"TransitionEvent"in window||delete hr.transitionend.transition);function Rs(t){if(mf[t])return mf[t];if(!hr[t])return t;var n=hr[t],a;for(a in n)if(n.hasOwnProperty(a)&&a in Jm)return mf[t]=n[a];return t}var $m=Rs("animationend"),tg=Rs("animationiteration"),eg=Rs("animationstart"),hy=Rs("transitionrun"),py=Rs("transitionstart"),my=Rs("transitioncancel"),ng=Rs("transitionend"),ig=new Map,gf="abort auxClick beforeToggle cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error fullscreenChange fullscreenError gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel".split(" ");gf.push("scrollEnd");function wi(t,n){ig.set(t,n),Ft(n,[t])}var gy=0;function va(t,n){if(t.name!=null&&t.name!=="auto")return t.name;if(n.autoName!==null)return n.autoName;t=Ui.identifierPrefix;var a=gy++;return t="_"+t+"t_"+a.toString(32)+"_",n.autoName=t}function ag(t){if(t==null||typeof t=="string")return t;var n=null,a=Lr;if(a!==null)for(var s=0;s<a.length;s++){var l=t[a[s]];if(l!=null){if(l==="none")return"none";n=n==null?l:n+(" "+l)}}return n??t.default}function _a(t,n){return t=ag(t),n=ag(n),n==null?t==="auto"?null:t:n==="auto"?null:n}var Kl=typeof reportError=="function"?reportError:function(t){if(typeof window=="object"&&typeof window.ErrorEvent=="function"){var n=new window.ErrorEvent("error",{bubbles:!0,cancelable:!0,message:typeof t=="object"&&t!==null&&typeof t.message=="string"?String(t.message):String(t),error:t});if(!window.dispatchEvent(n))return}else if(typeof process=="object"&&typeof process.emit=="function"){process.emit("uncaughtException",t);return}console.error(t)},_i=[],pr=0,vf=0;function Ql(){for(var t=pr,n=vf=pr=0;n<t;){var a=_i[n];_i[n++]=null;var s=_i[n];_i[n++]=null;var l=_i[n];_i[n++]=null;var u=_i[n];if(_i[n++]=null,s!==null&&l!==null){var g=s.pending;g===null?l.next=l:(l.next=g.next,g.next=l),s.pending=l}u!==0&&sg(a,l,u)}}function Jl(t,n,a,s){_i[pr++]=t,_i[pr++]=n,_i[pr++]=a,_i[pr++]=s,vf|=s,t.lanes|=s,t=t.alternate,t!==null&&(t.lanes|=s)}function _f(t,n,a,s){return Jl(t,n,a,s),$l(t)}function ws(t,n){return Jl(t,null,null,n),$l(t)}function sg(t,n,a){t.lanes|=a;var s=t.alternate;s!==null&&(s.lanes|=a);for(var l=!1,u=t.return;u!==null;)u.childLanes|=a,s=u.alternate,s!==null&&(s.childLanes|=a),u.tag===22&&(t=u.stateNode,t===null||t._visibility&1||(l=!0)),t=u,u=u.return;return t.tag===3?(u=t.stateNode,l&&n!==null&&(l=31-ue(a),t=u.hiddenUpdates,s=t[l],s===null?t[l]=[n]:s.push(n),n.lane=a|536870912),u):null}function $l(t){if(50<nl)throw nl=0,Vc=null,Error(r(185));for(var n=t.return;n!==null;)t=n,n=t.return;return t.tag===3?t.stateNode:null}var mr={};function vy(t,n,a,s){this.tag=t,this.key=a,this.sibling=this.child=this.return=this.stateNode=this.type=this.elementType=null,this.index=0,this.refCleanup=this.ref=null,this.pendingProps=n,this.dependencies=this.memoizedState=this.updateQueue=this.memoizedProps=null,this.mode=s,this.subtreeFlags=this.flags=0,this.deletions=null,this.childLanes=this.lanes=0,this.alternate=null}function jn(t,n,a,s){return new vy(t,n,a,s)}function xf(t){return t=t.prototype,!(!t||!t.isReactComponent)}function xa(t,n){var a=t.alternate;return a===null?(a=jn(t.tag,n,t.key,t.mode),a.elementType=t.elementType,a.type=t.type,a.stateNode=t.stateNode,a.alternate=t,t.alternate=a):(a.pendingProps=n,a.type=t.type,a.flags=0,a.subtreeFlags=0,a.deletions=null),a.flags=t.flags&1206910976,a.childLanes=t.childLanes,a.lanes=t.lanes,a.child=t.child,a.memoizedProps=t.memoizedProps,a.memoizedState=t.memoizedState,a.updateQueue=t.updateQueue,n=t.dependencies,a.dependencies=n===null?null:{lanes:n.lanes,firstContext:n.firstContext},a.sibling=t.sibling,a.index=t.index,a.ref=t.ref,a.refCleanup=t.refCleanup,a}function rg(t,n){t.flags&=1206910978;var a=t.alternate;return a===null?(t.childLanes=0,t.lanes=n,t.child=null,t.subtreeFlags=0,t.memoizedProps=null,t.memoizedState=null,t.updateQueue=null,t.dependencies=null,t.stateNode=null):(t.childLanes=a.childLanes,t.lanes=a.lanes,t.child=a.child,t.subtreeFlags=0,t.deletions=null,t.memoizedProps=a.memoizedProps,t.memoizedState=a.memoizedState,t.updateQueue=a.updateQueue,t.type=a.type,n=a.dependencies,t.dependencies=n===null?null:{lanes:n.lanes,firstContext:n.firstContext}),t}function tc(t,n,a,s,l,u){var g=0;if(s=t,typeof s=="function")xf(s)&&(g=1);else if(typeof s=="string")g=WM(t,a,Be.current)?26:t==="html"||t==="head"||t==="body"?27:5;else t:switch(s){case Tt:return t=jn(31,a,n,l),t.elementType=Tt,t.lanes=u,t;case j:return Cs(a.children,l,u,n);case J:g=8,l|=24;break;case P:return t=jn(12,a,n,l|2),t.elementType=P,t.lanes=u,t;case Q:return t=jn(13,a,n,l),t.elementType=Q,t.lanes=u,t;case pt:return t=jn(19,a,n,l),t.elementType=pt,t.lanes=u,t;case Zt:case H:return t=l|32,t=jn(30,a,n,t),t.elementType=H,t.lanes=u,t.stateNode={autoName:null,paired:null,clones:null,ref:null},t;default:if(typeof s=="object"&&s!==null)switch(s.$$typeof){case $:g=10;break t;case q:g=9;break t;case Y:g=11;break t;case ot:g=14;break t;case vt:g=16,s=null;break t}g=29,a=Error(r(130,t===null?"null":typeof t,"")),s=null}return n=jn(g,a,n,l),n.elementType=t,n.type=s,n.lanes=u,n}function Cs(t,n,a,s){return t=jn(7,t,s,n),t.lanes=a,t}function Sf(t,n,a){return t=jn(6,t,null,n),t.lanes=a,t}function og(t){var n=jn(18,null,null,0);return n.stateNode=t,n}function yf(t,n,a){return n=jn(4,t.children!==null?t.children:[],t.key,n),n.lanes=a,n.stateNode={containerInfo:t.containerInfo,pendingChildren:null,implementation:t.implementation},n}var lg=new WeakMap;function xi(t,n){if(typeof t=="object"&&t!==null){var a=lg.get(t);return a!==void 0?a:(n={value:t,source:n,stack:_t(n)},lg.set(t,n),n)}return{value:t,source:n,stack:_t(n)}}var gr=[],vr=0,ec=null,Po=0,Si=[],yi=0,Xa=null,ji=1,Zi="";function Sa(t,n){gr[vr++]=Po,gr[vr++]=ec,ec=t,Po=n}function cg(t,n,a){Si[yi++]=ji,Si[yi++]=Zi,Si[yi++]=Xa,Xa=t;var s=ji;t=Zi;var l=32-ue(s)-1;s&=~(1<<l),a+=1;var u=32-ue(n)+l;if(30<u){var g=l-l%5;u=(s&(1<<g)-1).toString(32),s>>=g,l-=g,ji=1<<32-ue(n)+l|a<<l|s,Zi=u+t}else ji=1<<u|a<<l|s,Zi=t}function nc(t){t.return!==null&&(Sa(t,1),cg(t,1,0))}function Mf(t){for(;t===ec;)ec=gr[--vr],gr[vr]=null,Po=gr[--vr],gr[vr]=null;for(;t===Xa;)Xa=Si[--yi],Si[yi]=null,Zi=Si[--yi],Si[yi]=null,ji=Si[--yi],Si[yi]=null}function ug(t,n){Si[yi++]=ji,Si[yi++]=Zi,Si[yi++]=Xa,ji=n.id,Zi=n.overflow,Xa=t}var En=null,tn=null,ye=!1,Wa=null,Mi=!1,Ef=Error(r(519));function qa(t){var n=Error(r(418,1<arguments.length&&arguments[1]!==void 0&&arguments[1]?"text":"HTML",""));throw Io(xi(n,t)),Ef}function fg(t){var n=t.stateNode,a=t.type,s=t.memoizedProps;switch(n[T]=t,n[V]=s,a){case"dialog":Te("cancel",n),Te("close",n);break;case"iframe":case"object":case"embed":Te("load",n);break;case"video":case"audio":for(a=0;a<al.length;a++)Te(al[a],n);break;case"source":Te("error",n);break;case"img":case"image":case"link":Te("error",n),Te("load",n);break;case"details":Te("toggle",n);break;case"input":Te("invalid",n),Em(n,s.value,s.defaultValue,s.checked,s.defaultChecked,s.type,s.name,!0);break;case"select":Te("invalid",n);break;case"textarea":Te("invalid",n),Tm(n,s.value,s.defaultValue,s.children)}a=s.children,typeof a!="string"&&typeof a!="number"&&typeof a!="bigint"||n.textContent===""+a||s.suppressHydrationWarning===!0||Uv(n.textContent,a)?(s.popover!=null&&(Te("beforetoggle",n),Te("toggle",n)),s.onScroll!=null&&Te("scroll",n),s.onScrollEnd!=null&&Te("scrollend",n),s.onClick!=null&&(n.onclick=Yi),n=!0):n=!1,n||qa(t,!0)}function ic(t){for(En=t.return;En;)switch(En.tag){case 5:case 31:case 13:Mi=!1;return;case 27:case 3:Mi=!0;return;default:En=En.return}}function _r(t){if(t!==En)return!1;if(!ye)return ic(t),ye=!0,!1;var n=t.tag,a;if((a=n!==3&&n!==27)&&((a=n===5)&&(a=t.type,a=!(a!=="form"&&a!=="button")||$d(t.type,t.memoizedProps)),a=!a),a&&tn&&qa(t),ic(t),n===13){if(t=t.memoizedState,t=t!==null?t.dehydrated:null,!t)throw Error(r(317));tn=Qv(t)}else if(n===31){if(t=t.memoizedState,t=t!==null?t.dehydrated:null,!t)throw Error(r(317));tn=Qv(t)}else n===27?(n=tn,ls(t.type)?(t=lh,lh=null,tn=t):tn=n):tn=En?bi(t.stateNode.nextSibling):null;return!0}function Ns(){tn=En=null,ye=!1}function bf(){var t=Wa;return t!==null&&(Qn===null?Qn=t:Qn.push.apply(Qn,t),Wa=null),t}function Io(t){Wa===null?Wa=[t]:Wa.push(t)}var Tf=ve(null),Ds=null,ya=null;function Ya(t,n,a){ne(Tf,n._currentValue),n._currentValue=a}function Ma(t){t._currentValue=Tf.current,te(Tf)}function ac(t,n,a){for(;t!==null;){var s=t.alternate;if((t.childLanes&n)!==n?(t.childLanes|=n,s!==null&&(s.childLanes|=n)):s!==null&&(s.childLanes&n)!==n&&(s.childLanes|=n),t===a)break;t=t.return}}function Af(t,n,a,s){var l=t.child;for(l!==null&&(l.return=t);l!==null;){var u=l.dependencies;if(u!==null){var g=l.child;u=u.firstContext;t:for(;u!==null;){var A=u;u=l;for(var z=0;z<n.length;z++)if(A.context===n[z]){u.lanes|=a,A=u.alternate,A!==null&&(A.lanes|=a),ac(u.return,a,t),s||(g=null);break t}u=A.next}}else if(l.tag===18){if(g=l.return,g===null)throw Error(r(341));g.lanes|=a,u=g.alternate,u!==null&&(u.lanes|=a),ac(g,a,t),g=null}else l.tag===13&&l.memoizedState!==null&&l.memoizedState.dehydrated===null?(l.lanes|=a,g=l.alternate,g!==null&&(g.lanes|=a),ac(l.return,a,t),g=l.child,g=g!==null?g.sibling:null):g=l.child;if(g!==null)g.return=l;else for(g=l;g!==null;){if(g===t){g=null;break}if(l=g.sibling,l!==null){l.return=g.return,g=l;break}g=g.return}l=g}}function Us(t,n,a,s){t=null;for(var l=n,u=!1;l!==null;){if(!u){if((l.flags&524288)!==0)u=!0;else if((l.flags&262144)!==0)break}if(l.tag===10){var g=l.alternate;if(g===null)throw Error(r(387));if(g=g.memoizedProps,g!==null){var A=l.type;si(l.pendingProps.value,g.value)||(t!==null?t.push(A):t=[A])}}else if(l===je.current){if(g=l.alternate,g===null)throw Error(r(387));g.memoizedState.memoizedState!==l.memoizedState.memoizedState&&(t!==null?t.push(kr):t=[kr])}l=l.return}return t!==null&&Af(n,t,a,s),n.flags|=262144,t!==null}function sc(t){for(t=t.firstContext;t!==null;){if(!si(t.context._currentValue,t.memoizedValue))return!0;t=t.next}return!1}function Ls(t){Ds=t,ya=null,t=t.dependencies,t!==null&&(t.firstContext=null)}function wn(t){return dg(Ds,t)}function rc(t,n){return Ds===null&&Ls(t),dg(t,n)}function dg(t,n){var a=n._currentValue;if(n={context:n,memoizedValue:a,next:null},ya===null){if(t===null)throw Error(r(308));ya=n,t.dependencies={lanes:0,firstContext:n},t.flags|=524288}else ya=ya.next=n;return a}var _y=typeof AbortController<"u"?AbortController:function(){var t=[],n=this.signal={aborted:!1,addEventListener:function(a,s){t.push(s)}};this.abort=function(){n.aborted=!0,t.forEach(function(a){return a()})}},xy=o.unstable_scheduleCallback,Sy=o.unstable_NormalPriority,hn={$$typeof:$,Consumer:null,Provider:null,_currentValue:null,_currentValue2:null,_threadCount:0};function Rf(){return{controller:new _y,data:new Map,refCount:0}}function Bo(t){t.refCount--,t.refCount===0&&xy(Sy,function(){t.controller.abort()})}function hg(t,n){if((t.pendingLanes&4194048)!==0){var a=t.transitionTypes;for(a===null&&(a=t.transitionTypes=[]),t=0;t<n.length;t++){var s=n[t];a.indexOf(s)===-1&&a.push(s)}}}var zo=null;function yy(t){var n=t.transitionTypes;return t.transitionTypes=null,n}var Fo=null,wf=0,Os=0,xr=null;function My(t,n){if(Fo===null){var a=Fo=[];wf=0,Os=Xd(),xr={status:"pending",value:void 0,then:function(s){a.push(s)}}}return wf++,n.then(pg,pg),n}function pg(){if(--wf===0&&(zo=null,Fo!==null)){xr!==null&&(xr.status="fulfilled");var t=Fo;Fo=null,Os=0,xr=null;for(var n=0;n<t.length;n++)(0,t[n])()}}function Ey(t,n){var a=[],s={status:"pending",value:null,reason:null,then:function(l){a.push(l)}};return t.then(function(){s.status="fulfilled",s.value=n;for(var l=0;l<a.length;l++)(0,a[l])(n)},function(l){for(s.status="rejected",s.reason=l,l=0;l<a.length;l++)(0,a[l])(void 0)}),s}var mg=xt.S;xt.S=function(t,n){if(ov=kt(),typeof n=="object"&&n!==null&&typeof n.then=="function"&&My(t,n),zo!==null)for(var a=Br;a!==null;)hg(a,zo),a=a.next;if(a=t.types,a!==null){for(var s=Br;s!==null;)hg(s,a),s=s.next;if(Os!==0){s=zo,s===null&&(s=zo=[]);for(var l=0;l<a.length;l++){var u=a[l];s.indexOf(u)===-1&&s.push(u)}}}mg!==null&&mg(t,n)};var Ps=ve(null);function Cf(){var t=Ps.current;return t!==null?t:Qe.pooledCache}function oc(t,n){n===null?ne(Ps,Ps.current):ne(Ps,n.pool)}function gg(){var t=Cf();return t===null?null:{parent:hn._currentValue,pool:t}}var Sr=Error(r(460)),Nf=Error(r(474)),lc=Error(r(542)),cc={then:function(){}};function vg(t){return t=t.status,t==="fulfilled"||t==="rejected"}function _g(t,n,a){switch(a=t[a],a===void 0?t.push(n):a!==n&&(n.then(Yi,Yi),n=a),n.status){case"fulfilled":return n.value;case"rejected":throw t=n.reason,Sg(t),t===void 0&&!("reason"in n)?Error(r(600)):t;default:if(typeof n.status=="string")n.then(Yi,Yi);else{if(t=Qe,t!==null&&100<t.shellSuspendCounter)throw Error(r(482));t=n,t.status="pending",t.then(function(s){if(n.status==="pending"){var l=n;l.status="fulfilled",l.value=s}},function(s){if(n.status==="pending"){var l=n;l.status="rejected",l.reason=s}})}switch(n.status){case"fulfilled":return n.value;case"rejected":throw t=n.reason,Sg(t),t}throw Bs=n,Sr}}function Is(t){try{var n=t._init;return n(t._payload)}catch(a){throw a!==null&&typeof a=="object"&&typeof a.then=="function"?(Bs=a,Sr):a}}var Bs=null;function xg(){if(Bs===null)throw Error(r(459));var t=Bs;return Bs=null,t}function Sg(t){if(t===Sr||t===lc)throw Error(r(483))}var yr=null,Ho=0;function uc(t){var n=Ho;return Ho+=1,yr===null&&(yr=[]),_g(yr,t,n)}function ja(t,n){n=n.props.ref,t.ref=n!==void 0?n:null}function fc(t,n){throw n.$$typeof===b?Error(r(525)):(t=Object.prototype.toString.call(n),Error(r(31,t==="[object Object]"?"object with keys {"+Object.keys(n).join(", ")+"}":t)))}function yg(t){function n(nt,W){if(t){var rt=nt.deletions;rt===null?(nt.deletions=[W],nt.flags|=16):rt.push(W)}}function a(nt,W){if(!t)return null;for(;W!==null;)n(nt,W),W=W.sibling;return null}function s(nt){for(var W=new Map;nt!==null;)nt.key===null?W.set(nt.index,nt):W.set(nt.key,nt),nt=nt.sibling;return W}function l(nt,W){return nt=xa(nt,W),nt.index=0,nt.sibling=null,nt}function u(nt,W,rt){return nt.index=rt,t?(rt=nt.alternate,rt!==null?(rt=rt.index,rt<W?(nt.flags|=2,W):rt):(nt.flags|=134217730,W)):(nt.flags|=1048576,W)}function g(nt){return t&&nt.alternate===null&&(nt.flags|=134217730),nt}function A(nt,W,rt,Mt){return W===null||W.tag!==6?(W=Sf(rt,nt.mode,Mt),W.return=nt,W):(W=l(W,rt),W.return=nt,W)}function z(nt,W,rt,Mt){var qt=rt.type;return qt===j?(nt=ht(nt,W,rt.props.children,Mt,rt.key),ja(nt,rt),nt):W!==null&&(W.elementType===qt||typeof qt=="object"&&qt!==null&&qt.$$typeof===vt&&Is(qt)===W.type)?(W=l(W,rt.props),ja(W,rt),W.return=nt,W):(W=tc(rt.type,rt.key,rt.props,null,nt.mode,Mt),ja(W,rt),W.return=nt,W)}function it(nt,W,rt,Mt){return W===null||W.tag!==4||W.stateNode.containerInfo!==rt.containerInfo||W.stateNode.implementation!==rt.implementation?(W=yf(rt,nt.mode,Mt),W.return=nt,W):(W=l(W,rt.children||[]),W.return=nt,W)}function ht(nt,W,rt,Mt,qt){return W===null||W.tag!==7?(W=Cs(rt,nt.mode,Mt,qt),W.return=nt,W):(W=l(W,rt),W.return=nt,W)}function Et(nt,W,rt){if(typeof W=="string"&&W!==""||typeof W=="number"||typeof W=="bigint")return W=Sf(""+W,nt.mode,rt),W.return=nt,W;if(typeof W=="object"&&W!==null){switch(W.$$typeof){case L:return rt=tc(W.type,W.key,W.props,null,nt.mode,rt),ja(rt,W),rt.return=nt,rt;case X:return W=yf(W,nt.mode,rt),W.return=nt,W;case vt:return W=Is(W),Et(nt,W,rt)}if(Dt(W)||G(W))return W=Cs(W,nt.mode,rt,null),W.return=nt,W;if(typeof W.then=="function")return Et(nt,uc(W),rt);if(W.$$typeof===$)return Et(nt,rc(nt,W),rt);fc(nt,W)}return null}function tt(nt,W,rt,Mt){var qt=W!==null?W.key:null;if(typeof rt=="string"&&rt!==""||typeof rt=="number"||typeof rt=="bigint")return qt!==null?null:A(nt,W,""+rt,Mt);if(typeof rt=="object"&&rt!==null){switch(rt.$$typeof){case L:return rt.key===qt?z(nt,W,rt,Mt):null;case X:return rt.key===qt?it(nt,W,rt,Mt):null;case vt:return rt=Is(rt),tt(nt,W,rt,Mt)}if(Dt(rt)||G(rt))return qt!==null?null:ht(nt,W,rt,Mt,null);if(typeof rt.then=="function")return tt(nt,W,uc(rt),Mt);if(rt.$$typeof===$)return tt(nt,W,rc(nt,rt),Mt);fc(nt,rt)}return null}function ft(nt,W,rt,Mt,qt){if(typeof Mt=="string"&&Mt!==""||typeof Mt=="number"||typeof Mt=="bigint")return nt=nt.get(rt)||null,A(W,nt,""+Mt,qt);if(typeof Mt=="object"&&Mt!==null){switch(Mt.$$typeof){case L:return nt=nt.get(Mt.key===null?rt:Mt.key)||null,z(W,nt,Mt,qt);case X:return nt=nt.get(Mt.key===null?rt:Mt.key)||null,it(W,nt,Mt,qt);case vt:return Mt=Is(Mt),ft(nt,W,rt,Mt,qt)}if(Dt(Mt)||G(Mt))return nt=nt.get(rt)||null,ht(W,nt,Mt,qt,null);if(typeof Mt.then=="function")return ft(nt,W,rt,uc(Mt),qt);if(Mt.$$typeof===$)return ft(nt,W,rt,rc(W,Mt),qt);fc(W,Mt)}return null}function Pt(nt,W,rt,Mt){for(var qt=null,Ne=null,ee=W,ae=W=0,gn=null;ee!==null&&ae<rt.length;ae++){ee.index>ae?(gn=ee,ee=null):gn=ee.sibling;var Ie=tt(nt,ee,rt[ae],Mt);if(Ie===null){ee===null&&(ee=gn);break}t&&ee&&Ie.alternate===null&&n(nt,ee),W=u(Ie,W,ae),Ne===null?qt=Ie:Ne.sibling=Ie,Ne=Ie,ee=gn}if(ae===rt.length)return a(nt,ee),ye&&Sa(nt,ae),qt;if(ee===null){for(;ae<rt.length;ae++)ee=Et(nt,rt[ae],Mt),ee!==null&&(W=u(ee,W,ae),Ne===null?qt=ee:Ne.sibling=ee,Ne=ee);return ye&&Sa(nt,ae),qt}for(ee=s(ee);ae<rt.length;ae++)gn=ft(ee,nt,ae,rt[ae],Mt),gn!==null&&(t&&(Ie=gn.alternate,Ie!==null&&ee.delete(Ie.key===null?ae:Ie.key)),W=u(gn,W,ae),Ne===null?qt=gn:Ne.sibling=gn,Ne=gn);return t&&ee.forEach(function(hs){return n(nt,hs)}),ye&&Sa(nt,ae),qt}function Qt(nt,W,rt,Mt){if(rt==null)throw Error(r(151));for(var qt=null,Ne=null,ee=W,ae=W=0,gn=null,Ie=rt.next();ee!==null&&!Ie.done;ae++,Ie=rt.next()){ee.index>ae?(gn=ee,ee=null):gn=ee.sibling;var hs=tt(nt,ee,Ie.value,Mt);if(hs===null){ee===null&&(ee=gn);break}t&&ee&&hs.alternate===null&&n(nt,ee),W=u(hs,W,ae),Ne===null?qt=hs:Ne.sibling=hs,Ne=hs,ee=gn}if(Ie.done)return a(nt,ee),ye&&Sa(nt,ae),qt;if(ee===null){for(;!Ie.done;ae++,Ie=rt.next())Ie=Et(nt,Ie.value,Mt),Ie!==null&&(W=u(Ie,W,ae),Ne===null?qt=Ie:Ne.sibling=Ie,Ne=Ie);return ye&&Sa(nt,ae),qt}for(ee=s(ee);!Ie.done;ae++,Ie=rt.next())Ie=ft(ee,nt,ae,Ie.value,Mt),Ie!==null&&(t&&(gn=Ie.alternate,gn!==null&&ee.delete(gn.key===null?ae:gn.key)),W=u(Ie,W,ae),Ne===null?qt=Ie:Ne.sibling=Ie,Ne=Ie);return t&&ee.forEach(function(iE){return n(nt,iE)}),ye&&Sa(nt,ae),qt}function ge(nt,W,rt,Mt){if(typeof rt=="object"&&rt!==null&&rt.type===j&&rt.key===null&&rt.props.ref===void 0&&(rt=rt.props.children),typeof rt=="object"&&rt!==null){switch(rt.$$typeof){case L:t:{for(var qt=rt.key;W!==null;){if(W.key===qt){if(qt=rt.type,qt===j){if(W.tag===7){a(nt,W.sibling),Mt=l(W,rt.props.children),ja(Mt,rt),Mt.return=nt,nt=Mt;break t}}else if(W.elementType===qt||typeof qt=="object"&&qt!==null&&qt.$$typeof===vt&&Is(qt)===W.type){a(nt,W.sibling),Mt=l(W,rt.props),ja(Mt,rt),Mt.return=nt,nt=Mt;break t}a(nt,W);break}else n(nt,W);W=W.sibling}rt.type===j?(Mt=Cs(rt.props.children,nt.mode,Mt,rt.key),ja(Mt,rt),Mt.return=nt,nt=Mt):(Mt=tc(rt.type,rt.key,rt.props,null,nt.mode,Mt),ja(Mt,rt),Mt.return=nt,nt=Mt)}return g(nt);case X:t:{for(qt=rt.key;W!==null;){if(W.key===qt)if(W.tag===4&&W.stateNode.containerInfo===rt.containerInfo&&W.stateNode.implementation===rt.implementation){a(nt,W.sibling),Mt=l(W,rt.children||[]),Mt.return=nt,nt=Mt;break t}else{a(nt,W);break}else n(nt,W);W=W.sibling}Mt=yf(rt,nt.mode,Mt),Mt.return=nt,nt=Mt}return g(nt);case vt:return rt=Is(rt),ge(nt,W,rt,Mt)}if(Dt(rt))return Pt(nt,W,rt,Mt);if(G(rt)){if(qt=G(rt),typeof qt!="function")throw Error(r(150));return rt=qt.call(rt),Qt(nt,W,rt,Mt)}if(typeof rt.then=="function")return ge(nt,W,uc(rt),Mt);if(rt.$$typeof===$)return ge(nt,W,rc(nt,rt),Mt);fc(nt,rt)}return typeof rt=="string"&&rt!==""||typeof rt=="number"||typeof rt=="bigint"?(rt=""+rt,W!==null&&W.tag===6?(a(nt,W.sibling),Mt=l(W,rt),Mt.return=nt,nt=Mt):(a(nt,W),Mt=Sf(rt,nt.mode,Mt),Mt.return=nt,nt=Mt),g(nt)):a(nt,W)}return function(nt,W,rt,Mt){try{Ho=0;var qt=ge(nt,W,rt,Mt);return yr=null,qt}catch(ee){if(ee===Sr||ee===lc)throw ee;var Ne=jn(29,ee,null,nt.mode);return Ne.lanes=Mt,Ne.return=nt,Ne}finally{}}}var zs=yg(!0),Mg=yg(!1),Za=!1;function Df(t){t.updateQueue={baseState:t.memoizedState,firstBaseUpdate:null,lastBaseUpdate:null,shared:{pending:null,lanes:0,hiddenCallbacks:null},callbacks:null}}function Uf(t,n){t=t.updateQueue,n.updateQueue===t&&(n.updateQueue={baseState:t.baseState,firstBaseUpdate:t.firstBaseUpdate,lastBaseUpdate:t.lastBaseUpdate,shared:t.shared,callbacks:null})}function Ka(t){return{lane:t,tag:0,payload:null,callback:null,next:null}}function Qa(t,n,a){var s=t.updateQueue;if(s===null)return null;if(s=s.shared,(Ge&2)!==0){var l=s.pending;return l===null?n.next=n:(n.next=l.next,l.next=n),s.pending=n,n=$l(t),sg(t,null,a),n}return Jl(t,s,n,a),$l(t)}function Go(t,n,a){if(n=n.updateQueue,n!==null&&(n=n.shared,(a&4194048)!==0)){var s=n.lanes;s&=t.pendingLanes,a|=s,n.lanes=a,Mo(t,a)}}function Lf(t,n){var a=t.updateQueue,s=t.alternate;if(s!==null&&(s=s.updateQueue,a===s)){var l=null,u=null;if(a=a.firstBaseUpdate,a!==null){do{var g={lane:a.lane,tag:a.tag,payload:a.payload,callback:null,next:null};u===null?l=u=g:u=u.next=g,a=a.next}while(a!==null);u===null?l=u=n:u=u.next=n}else l=u=n;a={baseState:s.baseState,firstBaseUpdate:l,lastBaseUpdate:u,shared:s.shared,callbacks:s.callbacks},t.updateQueue=a;return}t=a.lastBaseUpdate,t===null?a.firstBaseUpdate=n:t.next=n,a.lastBaseUpdate=n}var Of=!1;function Vo(){if(Of){var t=xr;if(t!==null)throw t}}function ko(t,n,a,s){Of=!1;var l=t.updateQueue;Za=!1;var u=l.firstBaseUpdate,g=l.lastBaseUpdate,A=l.shared.pending;if(A!==null){l.shared.pending=null;var z=A,it=z.next;z.next=null,g===null?u=it:g.next=it,g=z;var ht=t.alternate;ht!==null&&(ht=ht.updateQueue,A=ht.lastBaseUpdate,A!==g&&(A===null?ht.firstBaseUpdate=it:A.next=it,ht.lastBaseUpdate=z))}if(u!==null){var Et=l.baseState;g=0,ht=it=z=null,A=u;do{var tt=A.lane&-536870913,ft=tt!==A.lane;if(ft?(Ce&tt)===tt:(s&tt)===tt){tt!==0&&tt===Os&&(Of=!0),ht!==null&&(ht=ht.next={lane:0,tag:A.tag,payload:A.payload,callback:null,next:null});t:{var Pt=t,Qt=A;tt=n;var ge=a;switch(Qt.tag){case 1:if(Pt=Qt.payload,typeof Pt=="function"){Et=Pt.call(ge,Et,tt);break t}Et=Pt;break t;case 3:Pt.flags=Pt.flags&-65537|128;case 0:if(Pt=Qt.payload,tt=typeof Pt=="function"?Pt.call(ge,Et,tt):Pt,tt==null)break t;Et=B({},Et,tt);break t;case 2:Za=!0}}tt=A.callback,tt!==null&&(t.flags|=64,ft&&(t.flags|=8192),ft=l.callbacks,ft===null?l.callbacks=[tt]:ft.push(tt))}else ft={lane:tt,tag:A.tag,payload:A.payload,callback:A.callback,next:null},ht===null?(it=ht=ft,z=Et):ht=ht.next=ft,g|=tt;if(A=A.next,A===null){if(A=l.shared.pending,A===null)break;ft=A,A=ft.next,ft.next=null,l.lastBaseUpdate=ft,l.shared.pending=null}}while(!0);ht===null&&(z=Et),l.baseState=z,l.firstBaseUpdate=it,l.lastBaseUpdate=ht,u===null&&(l.shared.lanes=0),as|=g,t.lanes=g,t.memoizedState=Et}}function Eg(t,n){if(typeof t!="function")throw Error(r(191,t));t.call(n)}function bg(t,n){var a=t.callbacks;if(a!==null)for(t.callbacks=null,t=0;t<a.length;t++)Eg(a[t],n)}var Ja=ve(null),dc=ve(0);function Tg(t,n){t=Ra,ne(dc,t),ne(Ja,n),Ra=t|n.baseLanes}function Pf(){ne(dc,Ra),ne(Ja,Ja.current)}function If(){Ra=dc.current,te(Ja),te(dc)}var Cn=ve(null),Pn=null;function $a(t){var n=t.alternate;ne(Nn,Nn.current&1),ne(Cn,t),Pn===null&&(n===null||Ja.current!==null||n.memoizedState!==null)&&(Pn=t)}function Bf(t){ne(Nn,Nn.current),ne(Cn,t),Pn===null&&(Pn=t)}function Ag(t){t.tag===22?(ne(Nn,Nn.current),ne(Cn,t),Pn===null&&(Pn=t)):ts()}function ts(){ne(Nn,Nn.current),ne(Cn,Cn.current)}function ri(t){te(Cn),Pn===t&&(Pn=null),te(Nn)}var Nn=ve(0);function Xo(t,n){ne(Cn,Cn.current),ne(Nn,n)}function zf(t){te(Nn),te(Cn),Pn===t&&(Pn=null)}function hc(t){for(var n=t;n!==null;){if(n.tag===13){var a=n.memoizedState;if(a!==null&&(a=a.dehydrated,a===null||rh(a)||oh(a)))return n}else if(n.tag===19&&n.memoizedProps.revealOrder!=="independent"){if((n.flags&128)!==0)return n}else if(n.child!==null){n.child.return=n,n=n.child;continue}if(n===t)break;for(;n.sibling===null;){if(n.return===null||n.return===t)return null;n=n.return}n.sibling.return=n.return,n=n.sibling}return null}var Ea=0,me=null,Ke=null,pn=null,pc=!1,Mr=!1,Fs=!1,mc=0,Wo=0,Er=null,by=0;function un(){throw Error(r(321))}function Ff(t,n){if(n===null)return!1;for(var a=0;a<n.length&&a<t.length;a++)if(!si(t[a],n[a]))return!1;return!0}function Hf(t,n,a,s,l,u){return Ea=u,me=n,n.memoizedState=null,n.updateQueue=null,n.lanes=0,xt.H=t===null||t.memoizedState===null?c0:u0,Fs=!1,u=a(s,l),Fs=!1,Mr&&(u=wg(n,a,s,l)),Rg(t),u}function Rg(t){xt.H=Mc;var n=Ke!==null&&Ke.next!==null;if(Ea=0,pn=Ke=me=null,pc=!1,Wo=0,Er=null,n)throw Error(r(300));t===null||mn||(t=t.dependencies,t!==null&&sc(t)&&(mn=!0))}function wg(t,n,a,s){me=t;var l=0;do{if(Mr&&(Er=null),Wo=0,Mr=!1,25<=l)throw Error(r(301));if(l+=1,pn=Ke=null,t.updateQueue!=null){var u=t.updateQueue;u.lastEffect=null,u.events=null,u.stores=null,u.memoCache!=null&&(u.memoCache.index=0)}xt.H=Uy,u=n(a,s)}while(Mr);return u}function Ty(){var t=xt.H,n=t.useState()[0];return n=typeof n.then=="function"?qo(n):n,t=t.useState()[0],(Ke!==null?Ke.memoizedState:null)!==t&&(me.flags|=1024),n}function Gf(){var t=mc!==0;return mc=0,t}function Vf(t,n,a){n.updateQueue=t.updateQueue,n.flags&=-2053,t.lanes&=~a}function kf(t){if(pc){for(t=t.memoizedState;t!==null;){var n=t.queue;n!==null&&(n.pending=null),t=t.next}pc=!1}Ea=0,pn=Ke=me=null,Mr=!1,Wo=mc=0,Er=null}function Vn(){var t={memoizedState:null,baseState:null,baseQueue:null,queue:null,next:null};return pn===null?me.memoizedState=pn=t:pn=pn.next=t,pn}function dn(){if(Ke===null){var t=me.alternate;t=t!==null?t.memoizedState:null}else t=Ke.next;var n=pn===null?me.memoizedState:pn.next;if(n!==null)pn=n,Ke=t;else{if(t===null)throw me.alternate===null?Error(r(467)):Error(r(310));Ke=t,t={memoizedState:Ke.memoizedState,baseState:Ke.baseState,baseQueue:Ke.baseQueue,queue:Ke.queue,next:null},pn===null?me.memoizedState=pn=t:pn=pn.next=t}return pn}function gc(){return{lastEffect:null,events:null,stores:null,memoCache:null}}function qo(t){var n=Wo;return Wo+=1,Er===null&&(Er=[]),t=_g(Er,t,n),n=me,(pn===null?n.memoizedState:pn.next)===null&&(n=n.alternate,xt.H=n===null||n.memoizedState===null?c0:u0),t}function vc(t){if(t!==null&&typeof t=="object"){if(typeof t.then=="function")return qo(t);if(t.$$typeof===F)return;if(t.$$typeof===$)return wn(t)}throw Error(r(438,String(t)))}function Xf(t){var n=null,a=me.updateQueue;if(a!==null&&(n=a.memoCache),n==null){var s=me.alternate;s!==null&&(s=s.updateQueue,s!==null&&(s=s.memoCache,s!=null&&(n={data:s.data.map(function(l){return l.slice()}),index:0})))}if(n==null&&(n={data:[],index:0}),a===null&&(a=gc(),me.updateQueue=a),a.memoCache=n,a=n.data[n.index],a===void 0)for(a=n.data[n.index]=Array(t),s=0;s<t;s++)a[s]=Yt;return n.index++,a}function ba(t,n){return typeof n=="function"?n(t):n}function _c(t){var n=dn();return Wf(n,Ke,t)}function Wf(t,n,a){var s=t.queue;if(s===null)throw Error(r(311));s.lastRenderedReducer=a;var l=t.baseQueue,u=s.pending;if(u!==null){if(l!==null){var g=l.next;l.next=u.next,u.next=g}n.baseQueue=l=u,s.pending=null}if(u=t.baseState,l===null)t.memoizedState=u;else{n=l.next;var A=g=null,z=null,it=n,ht=!1;do{var Et=it.lane&-536870913;if(Et!==it.lane?(Ce&Et)===Et:(Ea&Et)===Et){var tt=it.revertLane;if(tt===0)z!==null&&(z=z.next={lane:0,revertLane:0,gesture:null,action:it.action,hasEagerState:it.hasEagerState,eagerState:it.eagerState,next:null}),Et===Os&&(ht=!0);else if((Ea&tt)===tt){it=it.next,tt===Os&&(ht=!0);continue}else Et={lane:0,revertLane:it.revertLane,gesture:null,action:it.action,hasEagerState:it.hasEagerState,eagerState:it.eagerState,next:null},z===null?(A=z=Et,g=u):z=z.next=Et,me.lanes|=tt,as|=tt;Et=it.action,Fs&&a(u,Et),u=it.hasEagerState?it.eagerState:a(u,Et)}else tt={lane:Et,revertLane:it.revertLane,gesture:it.gesture,action:it.action,hasEagerState:it.hasEagerState,eagerState:it.eagerState,next:null},z===null?(A=z=tt,g=u):z=z.next=tt,me.lanes|=Et,as|=Et;it=it.next}while(it!==null&&it!==n);if(z===null?g=u:z.next=A,!si(u,t.memoizedState)&&(mn=!0,ht&&(a=xr,a!==null)))throw a;t.memoizedState=u,t.baseState=g,t.baseQueue=z,s.lastRenderedState=u}return l===null&&(s.lanes=0),[t.memoizedState,s.dispatch]}function qf(t){var n=dn(),a=n.queue;if(a===null)throw Error(r(311));a.lastRenderedReducer=t;var s=a.dispatch,l=a.pending,u=n.memoizedState;if(l!==null){a.pending=null;var g=l=l.next;do u=t(u,g.action),g=g.next;while(g!==l);si(u,n.memoizedState)||(mn=!0),n.memoizedState=u,n.baseQueue===null&&(n.baseState=u),a.lastRenderedState=u}return[u,s]}function Cg(t,n,a){var s=me,l=dn(),u=ye;if(u){if(a===void 0)throw Error(r(407));a=a()}else a=n();var g=!si((Ke||l).memoizedState,a);if(g&&(l.memoizedState=a,mn=!0),l=l.queue,Zf(Ug.bind(null,s,l,t),[t]),t=l.getSnapshot!==n||g||pn!==null&&(pn.memoizedState.tag&1)!==0,br(t?9:8,{destroy:void 0},Dg.bind(null,s,l,a,n),null),t){if(s.flags|=2048,Qe===null)throw Error(r(349));u||(Ea&127)!==0||Ng(s,n,a)}return a}function Ng(t,n,a){t.flags|=16384,t={getSnapshot:n,value:a},n=me.updateQueue,n===null?(n=gc(),me.updateQueue=n,n.stores=[t]):(a=n.stores,a===null?n.stores=[t]:a.push(t))}function Dg(t,n,a,s){n.value=a,n.getSnapshot=s,Lg(n)&&Og(t)}function Ug(t,n,a){return a(function(){Lg(n)&&Og(t)})}function Lg(t){var n=t.getSnapshot;t=t.value;try{var a=n();return!si(t,a)}catch{return!0}}function Og(t){var n=ws(t,2);n!==null&&Jn(n,t,2)}function Yf(t){var n=Vn();if(typeof t=="function"){var a=t;if(t=a(),Fs){Le(!0);try{a()}finally{Le(!1)}}}return n.memoizedState=n.baseState=t,n.queue={pending:null,lanes:0,dispatch:null,lastRenderedReducer:ba,lastRenderedState:t},n}function Pg(t,n,a,s){return t.baseState=a,Wf(t,Ke,typeof s=="function"?s:ba)}function Ay(t,n,a,s,l){if(yc(t))throw Error(r(485));if(t=n.action,t!==null){var u={payload:l,action:t,next:null,isTransition:!0,status:"pending",value:null,reason:null,listeners:[],then:function(g){u.listeners.push(g)}};xt.T!==null?a(!0):u.isTransition=!1,s(u),a=n.pending,a===null?(u.next=n.pending=u,Ig(n,u)):(u.next=a.next,n.pending=a.next=u)}}function Ig(t,n){var a=n.action,s=n.payload,l=t.state;if(n.isTransition){var u=xt.T,g={};g.types=u!==null?u.types:null,xt.T=g;try{var A=a(l,s),z=xt.S;z!==null&&z(g,A),Bg(t,n,A)}catch(it){jf(t,n,it)}finally{u!==null&&g.types!==null&&(u.types=g.types),xt.T=u}}else try{u=a(l,s),Bg(t,n,u)}catch(it){jf(t,n,it)}}function Bg(t,n,a){a!==null&&typeof a=="object"&&typeof a.then=="function"?a.then(function(s){zg(t,n,s)},function(s){return jf(t,n,s)}):zg(t,n,a)}function zg(t,n,a){n.status="fulfilled",n.value=a,Fg(n),t.state=a,n=t.pending,n!==null&&(a=n.next,a===n?t.pending=null:(a=a.next,n.next=a,Ig(t,a)))}function jf(t,n,a){var s=t.pending;if(t.pending=null,s!==null){s=s.next;do n.status="rejected",n.reason=a,Fg(n),n=n.next;while(n!==s)}t.action=null}function Fg(t){t=t.listeners;for(var n=0;n<t.length;n++)(0,t[n])()}function Hg(t,n){return n}function Gg(t,n){if(ye){var a=Qe.formState;if(a!==null){t:{var s=me;if(ye){if(tn){e:{for(var l=tn,u=Mi;l.nodeType!==8;){if(!u){l=null;break e}if(l=bi(l.nextSibling),l===null){l=null;break e}}u=l.data,l=u==="F!"||u==="F"?l:null}if(l){tn=bi(l.nextSibling),s=l.data==="F!";break t}}qa(s)}s=!1}s&&(n=a[0])}}return a=Vn(),a.memoizedState=a.baseState=n,s={pending:null,lanes:0,dispatch:null,lastRenderedReducer:Hg,lastRenderedState:n},a.queue=s,a=r0.bind(null,me,s),s.dispatch=a,s=Yf(!1),u=td.bind(null,me,!1,s.queue),s=Vn(),l={state:n,dispatch:null,action:t,pending:null},s.queue=l,a=Ay.bind(null,me,l,u,a),l.dispatch=a,s.memoizedState=t,[n,a,!1]}function Vg(t){var n=dn();return kg(n,Ke,t)}function kg(t,n,a){if(n=Wf(t,n,Hg)[0],t=_c(ba)[0],typeof n=="object"&&n!==null&&typeof n.then=="function")try{var s=qo(n)}catch(g){throw g===Sr?lc:g}else s=n;n=dn();var l=n.queue,u=l.dispatch;return a!==n.memoizedState&&(me.flags|=2048,br(9,{destroy:void 0},Ry.bind(null,l,a),null)),[s,u,t]}function Ry(t,n){t.action=n}function Xg(t){var n=dn(),a=Ke;if(a!==null)return kg(n,a,t);dn(),n=n.memoizedState,a=dn();var s=a.queue.dispatch;return a.memoizedState=t,[n,s,!1]}function br(t,n,a,s){return t={tag:t,create:a,deps:s,inst:n,next:null},n=me.updateQueue,n===null&&(n=gc(),me.updateQueue=n),a=n.lastEffect,a===null?n.lastEffect=t.next=t:(s=a.next,a.next=t,t.next=s,n.lastEffect=t),t}function Wg(){return dn().memoizedState}function xc(t,n,a,s){var l=Vn();me.flags|=t,l.memoizedState=br(1|n,{destroy:void 0},a,s===void 0?null:s)}function Sc(t,n,a,s){var l=dn();s=s===void 0?null:s;var u=l.memoizedState.inst;Ke!==null&&s!==null&&Ff(s,Ke.memoizedState.deps)?l.memoizedState=br(n,u,a,s):(me.flags|=t,l.memoizedState=br(1|n,u,a,s))}function qg(t,n){xc(8390656,8,t,n)}function Zf(t,n){Sc(2048,8,t,n)}function wy(t){me.flags|=4;var n=me.updateQueue;if(n===null)n=gc(),me.updateQueue=n,n.events=[t];else{var a=n.events;a===null?n.events=[t]:a.push(t)}}function Yg(t){var n=dn().memoizedState;return wy({ref:n,nextImpl:t}),function(){if((Ge&2)!==0)throw Error(r(440));return n.impl.apply(void 0,arguments)}}function jg(t,n){return Sc(4,2,t,n)}function Zg(t,n){return Sc(4,4,t,n)}function Kg(t,n){if(typeof n=="function"){t=t();var a=n(t);return function(){typeof a=="function"?a():n(null)}}if(n!=null)return t=t(),n.current=t,function(){n.current=null}}function Qg(t,n,a){a=a!=null?a.concat([t]):null,Sc(4,4,Kg.bind(null,n,t),a)}function Kf(){}function Jg(t,n){var a=dn();n=n===void 0?null:n;var s=a.memoizedState;return n!==null&&Ff(n,s[1])?s[0]:(a.memoizedState=[t,n],t)}function $g(t,n){var a=dn();n=n===void 0?null:n;var s=a.memoizedState;if(n!==null&&Ff(n,s[1]))return s[0];if(s=t(),Fs){Le(!0);try{t()}finally{Le(!1)}}return a.memoizedState=[s,n],s}function Qf(t,n,a){return a===void 0||(Ea&1073741824)!==0&&(Ce&261930)===0?t.memoizedState=n:(t.memoizedState=a,t=cv(),me.lanes|=t,as|=t,a)}function t0(t,n,a,s){return si(a,n)?a:Ja.current!==null?(t=Qf(t,a,s),si(t,n)||(mn=!0),t):(Ea&106)===0||(Ea&1073741824)!==0&&(Ce&261930)===0?(mn=!0,t.memoizedState=a):(t=cv(),me.lanes|=t,as|=t,n)}function e0(t,n,a,s,l){var u=Ct.p;Ct.p=u!==0&&8>u?u:8;var g=xt.T,A={};A.types=g!==null?g.types:null,xt.T=A,td(t,!1,n,a);try{var z=l(),it=xt.S;if(it!==null&&it(A,z),z!==null&&typeof z=="object"&&typeof z.then=="function"){var ht=Ey(z,s);Yo(t,n,ht,ui(t))}else Yo(t,n,s,ui(t))}catch(Et){Yo(t,n,{then:function(){},status:"rejected",reason:Et},ui())}finally{Ct.p=u,g!==null&&A.types!==null&&(g.types=A.types),xt.T=g}}function Cy(){}function Jf(t,n,a,s){if(t.tag!==5)throw Error(r(476));var l=n0(t).queue;e0(t,l,n,Fe,a===null?Cy:function(){return i0(t),a(s)})}function n0(t){var n=t.memoizedState;if(n!==null)return n;n={memoizedState:Fe,baseState:Fe,baseQueue:null,queue:{pending:null,lanes:0,dispatch:null,lastRenderedReducer:ba,lastRenderedState:Fe},next:null};var a={};return n.next={memoizedState:a,baseState:a,baseQueue:null,queue:{pending:null,lanes:0,dispatch:null,lastRenderedReducer:ba,lastRenderedState:a},next:null},t.memoizedState=n,t=t.alternate,t!==null&&(t.memoizedState=n),n}function i0(t){var n=n0(t);n.next===null&&(n=t.alternate.memoizedState),Yo(t,n.next.queue,{},ui())}function $f(){return wn(kr)}function a0(){return dn().memoizedState}function s0(){return dn().memoizedState}function Ny(t){for(var n=t.return;n!==null;){switch(n.tag){case 24:case 3:var a=ui();t=Ka(a);var s=Qa(n,t,a);s!==null&&(Jn(s,n,a),Go(s,n,a)),n={cache:Rf()},t.payload=n;return}n=n.return}}function Dy(t,n,a){var s=ui();a={lane:s,revertLane:0,gesture:null,action:a,hasEagerState:!1,eagerState:null,next:null},yc(t)?o0(n,a):(a=_f(t,n,a,s),a!==null&&(Jn(a,t,s),l0(a,n,s)))}function r0(t,n,a){var s=ui();Yo(t,n,a,s)}function Yo(t,n,a,s){var l={lane:s,revertLane:0,gesture:null,action:a,hasEagerState:!1,eagerState:null,next:null};if(yc(t))o0(n,l);else{var u=t.alternate;if(t.lanes===0&&(u===null||u.lanes===0)&&(u=n.lastRenderedReducer,u!==null))try{var g=n.lastRenderedState,A=u(g,a);if(l.hasEagerState=!0,l.eagerState=A,si(A,g))return Jl(t,n,l,0),Qe===null&&Ql(),!1}catch{}finally{}if(a=_f(t,n,l,s),a!==null)return Jn(a,t,s),l0(a,n,s),!0}return!1}function td(t,n,a,s){if(s={lane:2,revertLane:Xd(),gesture:null,action:s,hasEagerState:!1,eagerState:null,next:null},yc(t)){if(n)throw Error(r(479))}else n=_f(t,a,s,2),n!==null&&Jn(n,t,2)}function yc(t){var n=t.alternate;return t===me||n!==null&&n===me}function o0(t,n){Mr=pc=!0;var a=t.pending;a===null?n.next=n:(n.next=a.next,a.next=n),t.pending=n}function l0(t,n,a){if((a&4194048)!==0){var s=n.lanes;s&=t.pendingLanes,a|=s,n.lanes=a,Mo(t,a)}}var Mc={readContext:wn,use:vc,useCallback:un,useContext:un,useEffect:un,useImperativeHandle:un,useLayoutEffect:un,useInsertionEffect:un,useMemo:un,useReducer:un,useRef:un,useState:un,useDebugValue:un,useDeferredValue:un,useTransition:un,useSyncExternalStore:un,useId:un,useHostTransitionStatus:un,useFormState:un,useActionState:un,useOptimistic:un,useMemoCache:un,useCacheRefresh:un,useEffectEvent:un},c0={readContext:wn,use:vc,useCallback:function(t,n){return Vn().memoizedState=[t,n===void 0?null:n],t},useContext:wn,useEffect:qg,useImperativeHandle:function(t,n,a){a=a!=null?a.concat([t]):null,xc(4194308,4,Kg.bind(null,n,t),a)},useLayoutEffect:function(t,n){return xc(4194308,4,t,n)},useInsertionEffect:function(t,n){xc(4,2,t,n)},useMemo:function(t,n){var a=Vn();n=n===void 0?null:n;var s=t();if(Fs){Le(!0);try{t()}finally{Le(!1)}}return a.memoizedState=[s,n],s},useReducer:function(t,n,a){var s=Vn();if(a!==void 0){var l=a(n);if(Fs){Le(!0);try{a(n)}finally{Le(!1)}}}else l=n;return s.memoizedState=s.baseState=l,t={pending:null,lanes:0,dispatch:null,lastRenderedReducer:t,lastRenderedState:l},s.queue=t,t=t.dispatch=Dy.bind(null,me,t),[s.memoizedState,t]},useRef:function(t){var n=Vn();return t={current:t},n.memoizedState=t},useState:function(t){t=Yf(t);var n=t.queue,a=r0.bind(null,me,n);return n.dispatch=a,[t.memoizedState,a]},useDebugValue:Kf,useDeferredValue:function(t,n){var a=Vn();return Qf(a,t,n)},useTransition:function(){var t=Yf(!1);return t=e0.bind(null,me,t.queue,!0,!1),Vn().memoizedState=t,[!1,t]},useSyncExternalStore:function(t,n,a){var s=me,l=Vn();if(ye){if(a===void 0)throw Error(r(407));a=a()}else{if(a=n(),Qe===null)throw Error(r(349));(Ce&127)!==0||Ng(s,n,a)}l.memoizedState=a;var u={value:a,getSnapshot:n};return l.queue=u,qg(Ug.bind(null,s,u,t),[t]),s.flags|=2048,br(9,{destroy:void 0},Dg.bind(null,s,u,a,n),null),a},useId:function(){var t=Vn(),n=Qe.identifierPrefix;if(ye){var a=Zi,s=ji;a=(s&~(1<<32-ue(s)-1)).toString(32)+a,n="_"+n+"R_"+a,a=mc++,0<a&&(n+="H"+a.toString(32)),n+="_"}else a=by++,n="_"+n+"r_"+a.toString(32)+"_";return t.memoizedState=n},useHostTransitionStatus:$f,useFormState:Gg,useActionState:Gg,useOptimistic:function(t){var n=Vn();n.memoizedState=n.baseState=t;var a={pending:null,lanes:0,dispatch:null,lastRenderedReducer:null,lastRenderedState:null};return n.queue=a,n=td.bind(null,me,!0,a),a.dispatch=n,[t,n]},useMemoCache:Xf,useCacheRefresh:function(){return Vn().memoizedState=Ny.bind(null,me)},useEffectEvent:function(t){var n=Vn(),a={impl:t};return n.memoizedState=a,function(){if((Ge&2)!==0)throw Error(r(440));return a.impl.apply(void 0,arguments)}}},u0={readContext:wn,use:vc,useCallback:Jg,useContext:wn,useEffect:Zf,useImperativeHandle:Qg,useInsertionEffect:jg,useLayoutEffect:Zg,useMemo:$g,useReducer:_c,useRef:Wg,useState:function(){return _c(ba)},useDebugValue:Kf,useDeferredValue:function(t,n){var a=dn();return t0(a,Ke.memoizedState,t,n)},useTransition:function(){var t=_c(ba)[0],n=dn().memoizedState;return[typeof t=="boolean"?t:qo(t),n]},useSyncExternalStore:Cg,useId:a0,useHostTransitionStatus:$f,useFormState:Vg,useActionState:Vg,useOptimistic:function(t,n){var a=dn();return Pg(a,Ke,t,n)},useMemoCache:Xf,useCacheRefresh:s0,useEffectEvent:Yg},Uy={readContext:wn,use:vc,useCallback:Jg,useContext:wn,useEffect:Zf,useImperativeHandle:Qg,useInsertionEffect:jg,useLayoutEffect:Zg,useMemo:$g,useReducer:qf,useRef:Wg,useState:function(){return qf(ba)},useDebugValue:Kf,useDeferredValue:function(t,n){var a=dn();return Ke===null?Qf(a,t,n):t0(a,Ke.memoizedState,t,n)},useTransition:function(){var t=qf(ba)[0],n=dn().memoizedState;return[typeof t=="boolean"?t:qo(t),n]},useSyncExternalStore:Cg,useId:a0,useHostTransitionStatus:$f,useFormState:Xg,useActionState:Xg,useOptimistic:function(t,n){var a=dn();return Ke!==null?Pg(a,Ke,t,n):(a.baseState=t,[t,a.queue.dispatch])},useMemoCache:Xf,useCacheRefresh:s0,useEffectEvent:Yg};function ed(t,n,a,s){n=t.memoizedState,a=a(s,n),a=a==null?n:B({},n,a),t.memoizedState=a,t.lanes===0&&(t.updateQueue.baseState=a)}var nd={enqueueSetState:function(t,n,a){t=t._reactInternals;var s=ui(),l=Ka(s);l.payload=n,a!=null&&(l.callback=a),n=Qa(t,l,s),n!==null&&(Jn(n,t,s),Go(n,t,s))},enqueueReplaceState:function(t,n,a){t=t._reactInternals;var s=ui(),l=Ka(s);l.tag=1,l.payload=n,a!=null&&(l.callback=a),n=Qa(t,l,s),n!==null&&(Jn(n,t,s),Go(n,t,s))},enqueueForceUpdate:function(t,n){t=t._reactInternals;var a=ui(),s=Ka(a);s.tag=2,n!=null&&(s.callback=n),n=Qa(t,s,a),n!==null&&(Jn(n,t,a),Go(n,t,a))}};function f0(t,n,a,s,l,u,g){return t=t.stateNode,typeof t.shouldComponentUpdate=="function"?t.shouldComponentUpdate(s,u,g):n.prototype&&n.prototype.isPureReactComponent?!Lo(a,s)||!Lo(l,u):!0}function d0(t,n,a,s){t=n.state,typeof n.componentWillReceiveProps=="function"&&n.componentWillReceiveProps(a,s),typeof n.UNSAFE_componentWillReceiveProps=="function"&&n.UNSAFE_componentWillReceiveProps(a,s),n.state!==t&&nd.enqueueReplaceState(n,n.state,null)}function Hs(t,n){var a=n;if("ref"in n){a={};for(var s in n)s!=="ref"&&(a[s]=n[s])}if(t=t.defaultProps){a===n&&(a=B({},a));for(var l in t)a[l]===void 0&&(a[l]=t[l])}return a}function h0(t){Kl(t)}function p0(t){console.error(t)}function m0(t){Kl(t)}function Ec(t,n){try{var a=t.onUncaughtError;a(n.value,{componentStack:n.stack})}catch(s){setTimeout(function(){throw s})}}function g0(t,n,a){try{var s=t.onCaughtError;s(a.value,{componentStack:a.stack,errorBoundary:n.tag===1?n.stateNode:null})}catch(l){setTimeout(function(){throw l})}}function id(t,n,a){return a=Ka(a),a.tag=3,a.payload={element:null},a.callback=function(){Ec(t,n)},a}function v0(t){return t=Ka(t),t.tag=3,t}function _0(t,n,a,s){var l=a.type.getDerivedStateFromError;if(typeof l=="function"){var u=s.value;t.payload=function(){return l(u)},t.callback=function(){g0(n,a,s)}}var g=a.stateNode;g!==null&&typeof g.componentDidCatch=="function"&&(t.callback=function(){g0(n,a,s),typeof l!="function"&&(ss===null?ss=new Set([this]):ss.add(this));var A=s.stack;this.componentDidCatch(s.value,{componentStack:A!==null?A:""})})}function Ly(t,n,a,s,l){if(a.flags|=32768,s!==null&&typeof s=="object"&&typeof s.then=="function"){if(n=a.alternate,n!==null&&Us(n,a,l,!0),a=Cn.current,a!==null){switch(a.tag){case 31:case 13:case 19:return Pn===null?Xc():a.alternate===null&&fn===0&&(fn=3),a.flags&=-257,a.flags|=65536,a.lanes=l,s===cc?a.flags|=16384:(n=a.updateQueue,n===null?a.updateQueue=new Set([s]):n.add(s),Gd(t,s,l)),!1;case 22:return a.flags|=65536,s===cc?a.flags|=16384:(n=a.updateQueue,n===null?(n={transitions:null,markerInstances:null,retryQueue:new Set([s])},a.updateQueue=n):(a=n.retryQueue,a===null?n.retryQueue=new Set([s]):a.add(s)),Gd(t,s,l)),!1}throw Error(r(435,a.tag))}return Gd(t,s,l),Xc(),!1}if(ye)return n=Cn.current,n!==null?((n.flags&65536)===0&&(n.flags|=256),n.flags|=65536,n.lanes=l,s!==Ef&&(t=Error(r(422),{cause:s}),Io(xi(t,a)))):(s!==Ef&&(n=Error(r(423),{cause:s}),Io(xi(n,a))),t=t.current.alternate,t.flags|=65536,l&=-l,t.lanes|=l,s=xi(s,a),l=id(t.stateNode,s,l),Lf(t,l),fn!==4&&(fn=2)),!1;var u=Error(r(520),{cause:s});if(u=xi(u,a),el===null?el=[u]:el.push(u),fn!==4&&(fn=2),n===null)return!0;s=xi(s,a),a=n;do{switch(a.tag){case 3:return a.flags|=65536,t=l&-l,a.lanes|=t,t=id(a.stateNode,s,t),Lf(a,t),!1;case 1:if(n=a.type,u=a.stateNode,(a.flags&128)===0&&(typeof n.getDerivedStateFromError=="function"||u!==null&&typeof u.componentDidCatch=="function"&&(ss===null||!ss.has(u))))return a.flags|=65536,l&=-l,a.lanes|=l,l=v0(l),_0(l,t,a,s),Lf(a,l),!1;break;case 22:if(a.memoizedState!==null)return a.flags|=65536,!1}a=a.return}while(a!==null);return!1}var ad=Error(r(461)),mn=!1;function Sn(t,n,a,s){n.child=t===null?Mg(n,null,a,s):zs(n,t.child,a,s)}function x0(t,n,a,s,l){a=a.render;var u=n.ref;if("ref"in s){var g={};for(var A in s)A!=="ref"&&(g[A]=s[A])}else g=s;return Ls(n),s=Hf(t,n,a,g,u,l),A=Gf(),t!==null&&!mn?(Vf(t,n,l),Ta(t,n,l)):(ye&&A&&nc(n),n.flags|=1,Sn(t,n,s,l),n.child)}function S0(t,n,a,s,l){if(t===null){var u=a.type;return typeof u=="function"&&!xf(u)&&u.defaultProps===void 0&&a.compare===null?(n.tag=15,n.type=u,y0(t,n,u,s,l)):(t=tc(a.type,null,s,n,n.mode,l),t.ref=n.ref,t.return=n,n.child=t)}if(u=t.child,!dd(t,l)){var g=u.memoizedProps;if(a=a.compare,a=a!==null?a:Lo,a(g,s)&&t.ref===n.ref)return Ta(t,n,l)}return n.flags|=1,t=xa(u,s),t.ref=n.ref,t.return=n,n.child=t}function y0(t,n,a,s,l){if(t!==null){var u=t.memoizedProps;if(Lo(u,s)&&t.ref===n.ref)if(mn=!1,n.pendingProps=s=u,dd(t,l))(t.flags&131072)!==0&&(mn=!0);else return n.lanes=t.lanes,Ta(t,n,l)}return sd(t,n,a,s,l)}function M0(t,n,a,s){var l=s.children,u=t!==null?t.memoizedState:null;if(t===null&&n.stateNode===null&&(n.stateNode={_visibility:1,_pendingMarkers:null,_retryCache:null,_transitions:null}),s.mode==="hidden"){if((n.flags&128)!==0){if(u=u!==null?u.baseLanes|a:a,t!==null){for(s=n.child=t.child,l=0;s!==null;)l=l|s.lanes|s.childLanes,s=s.sibling;s=l&~u}else s=0,n.child=null;return E0(t,n,u,a,s)}if((a&536870912)!==0)n.memoizedState={baseLanes:0,cachePool:null},t!==null&&oc(n,u!==null?u.cachePool:null),u!==null?Tg(n,u):Pf(),Ag(n);else return s=n.lanes=536870912,E0(t,n,u!==null?u.baseLanes|a:a,a,s)}else u!==null?(oc(n,u.cachePool),Tg(n,u),ts(),n.memoizedState=null):(t!==null&&oc(n,null),Pf(),ts());return Sn(t,n,l,a),n.child}function jo(t,n){return t!==null&&t.tag===22||n.stateNode!==null||(n.stateNode={_visibility:1,_pendingMarkers:null,_retryCache:null,_transitions:null}),n.sibling}function E0(t,n,a,s,l){var u=Cf();return u=u===null?null:{parent:hn._currentValue,pool:u},n.memoizedState={baseLanes:a,cachePool:u},t!==null&&oc(n,null),Pf(),Ag(n),t!==null&&Us(t,n,s,!0),n.childLanes=l,null}function bc(t,n){return n=Tc({mode:n.mode,children:n.children},t.mode),n.ref=t.ref,t.child=n,n.return=t,n}function b0(t,n,a){return zs(n,t.child,null,a),t=bc(n,n.pendingProps),t.flags|=2,ri(n),n.memoizedState=null,t}function Oy(t,n,a){var s=n.pendingProps,l=(n.flags&128)!==0;if(n.flags&=-129,t===null){if(ye){if(s.mode==="hidden")return t=bc(n,s),n.lanes=536870912,t.memoizedState={baseLanes:0,cachePool:null},jo(null,t);if(Bf(n),(t=tn)?(t=Kv(t,Mi),t=t!==null&&t.data==="&"?t:null,t!==null&&(n.memoizedState={dehydrated:t,treeContext:Xa!==null?{id:ji,overflow:Zi}:null,retryLane:536870912,hydrationErrors:null},a=og(t),a.return=n,n.child=a,En=n,tn=null)):t=null,t===null)throw qa(n);return n.lanes=536870912,null}return bc(n,s)}var u=t.memoizedState;if(u!==null){var g=u.dehydrated;if(Bf(n),l)if(n.flags&256)n.flags&=-257,n=b0(t,n,a);else if(n.memoizedState!==null)n.child=t.child,n.flags|=128,n=null;else throw Error(r(558));else if(mn||Us(t,n,a,!1),l=(a&t.childLanes)!==0,mn||l){if(Ja.current===null){if(s=Qe,s!==null&&(g=Eo(s,a),g!==0&&g!==u.retryLane))throw u.retryLane=g,ws(t,g),Jn(s,t,g),ad;Xc()}n=b0(t,n,a)}else t=u.treeContext,tn=bi(g.nextSibling),En=n,ye=!0,Wa=null,Mi=!1,t!==null&&ug(n,t),n=bc(n,s),n.flags|=134221824;return n}return t=xa(t.child,{mode:s.mode,children:s.children}),t.ref=n.ref,n.child=t,t.return=n,t}function Tr(t,n){var a=n.ref;if(a===null)t!==null&&t.ref!==null&&(n.flags|=4194816);else{if(typeof a!="function"&&typeof a!="object")throw Error(r(284));(t===null||t.ref!==a)&&(n.flags|=4194816)}}function sd(t,n,a,s,l){return Ls(n),a=Hf(t,n,a,s,void 0,l),s=Gf(),t!==null&&!mn?(Vf(t,n,l),Ta(t,n,l)):(ye&&s&&nc(n),n.flags|=1,Sn(t,n,a,l),n.child)}function T0(t,n,a,s,l,u){return Ls(n),n.updateQueue=null,a=wg(n,s,a,l),Rg(t),s=Gf(),t!==null&&!mn?(Vf(t,n,u),Ta(t,n,u)):(ye&&s&&nc(n),n.flags|=1,Sn(t,n,a,u),n.child)}function A0(t,n,a,s,l){if(Ls(n),n.stateNode===null){var u=mr,g=a.contextType;typeof g=="object"&&g!==null&&(u=wn(g)),u=new a(s,u),n.memoizedState=u.state!==null&&u.state!==void 0?u.state:null,u.updater=nd,n.stateNode=u,u._reactInternals=n,u=n.stateNode,u.props=s,u.state=n.memoizedState,u.refs={},Df(n),g=a.contextType,u.context=typeof g=="object"&&g!==null?wn(g):mr,u.state=n.memoizedState,g=a.getDerivedStateFromProps,typeof g=="function"&&(ed(n,a,g,s),u.state=n.memoizedState),typeof a.getDerivedStateFromProps=="function"||typeof u.getSnapshotBeforeUpdate=="function"||typeof u.UNSAFE_componentWillMount!="function"&&typeof u.componentWillMount!="function"||(g=u.state,typeof u.componentWillMount=="function"&&u.componentWillMount(),typeof u.UNSAFE_componentWillMount=="function"&&u.UNSAFE_componentWillMount(),g!==u.state&&nd.enqueueReplaceState(u,u.state,null),ko(n,s,u,l),Vo(),u.state=n.memoizedState),typeof u.componentDidMount=="function"&&(n.flags|=4194308),s=!0}else if(t===null){u=n.stateNode;var A=n.memoizedProps,z=Hs(a,A);u.props=z;var it=u.context,ht=a.contextType;g=mr,typeof ht=="object"&&ht!==null&&(g=wn(ht));var Et=a.getDerivedStateFromProps;ht=typeof Et=="function"||typeof u.getSnapshotBeforeUpdate=="function",A=n.pendingProps!==A,ht||typeof u.UNSAFE_componentWillReceiveProps!="function"&&typeof u.componentWillReceiveProps!="function"||(A||it!==g)&&d0(n,u,s,g),Za=!1;var tt=n.memoizedState;u.state=tt,ko(n,s,u,l),Vo(),it=n.memoizedState,A||tt!==it||Za?(typeof Et=="function"&&(ed(n,a,Et,s),it=n.memoizedState),(z=Za||f0(n,a,z,s,tt,it,g))?(ht||typeof u.UNSAFE_componentWillMount!="function"&&typeof u.componentWillMount!="function"||(typeof u.componentWillMount=="function"&&u.componentWillMount(),typeof u.UNSAFE_componentWillMount=="function"&&u.UNSAFE_componentWillMount()),typeof u.componentDidMount=="function"&&(n.flags|=4194308)):(typeof u.componentDidMount=="function"&&(n.flags|=4194308),n.memoizedProps=s,n.memoizedState=it),u.props=s,u.state=it,u.context=g,s=z):(typeof u.componentDidMount=="function"&&(n.flags|=4194308),s=!1)}else{u=n.stateNode,Uf(t,n),g=n.memoizedProps,ht=Hs(a,g),u.props=ht,Et=n.pendingProps,tt=u.context,it=a.contextType,z=mr,typeof it=="object"&&it!==null&&(z=wn(it)),A=a.getDerivedStateFromProps,(it=typeof A=="function"||typeof u.getSnapshotBeforeUpdate=="function")||typeof u.UNSAFE_componentWillReceiveProps!="function"&&typeof u.componentWillReceiveProps!="function"||(g!==Et||tt!==z)&&d0(n,u,s,z),Za=!1,tt=n.memoizedState,u.state=tt,ko(n,s,u,l),Vo();var ft=n.memoizedState;g!==Et||tt!==ft||Za||t!==null&&t.dependencies!==null&&sc(t.dependencies)?(typeof A=="function"&&(ed(n,a,A,s),ft=n.memoizedState),(ht=Za||f0(n,a,ht,s,tt,ft,z)||t!==null&&t.dependencies!==null&&sc(t.dependencies))?(it||typeof u.UNSAFE_componentWillUpdate!="function"&&typeof u.componentWillUpdate!="function"||(typeof u.componentWillUpdate=="function"&&u.componentWillUpdate(s,ft,z),typeof u.UNSAFE_componentWillUpdate=="function"&&u.UNSAFE_componentWillUpdate(s,ft,z)),typeof u.componentDidUpdate=="function"&&(n.flags|=4),typeof u.getSnapshotBeforeUpdate=="function"&&(n.flags|=1024)):(typeof u.componentDidUpdate!="function"||g===t.memoizedProps&&tt===t.memoizedState||(n.flags|=4),typeof u.getSnapshotBeforeUpdate!="function"||g===t.memoizedProps&&tt===t.memoizedState||(n.flags|=1024),n.memoizedProps=s,n.memoizedState=ft),u.props=s,u.state=ft,u.context=z,s=ht):(typeof u.componentDidUpdate!="function"||g===t.memoizedProps&&tt===t.memoizedState||(n.flags|=4),typeof u.getSnapshotBeforeUpdate!="function"||g===t.memoizedProps&&tt===t.memoizedState||(n.flags|=1024),s=!1)}return u=s,Tr(t,n),s=(n.flags&128)!==0,u||s?(u=n.stateNode,a=s&&typeof a.getDerivedStateFromError!="function"?null:u.render(),n.flags|=1,t!==null&&s?(n.child=zs(n,t.child,null,l),n.child=zs(n,null,a,l)):Sn(t,n,a,l),n.memoizedState=u.state,t=n.child):t=Ta(t,n,l),t}function R0(t,n,a,s){return Ns(),n.flags|=256,Sn(t,n,a,s),n.child}var rd={dehydrated:null,treeContext:null,retryLane:0,hydrationErrors:null};function od(t){return{baseLanes:t,cachePool:gg()}}function ld(t,n,a){return t=t!==null?t.childLanes&~a:0,n&&(t|=ci),t}function w0(t,n,a){var s=n.pendingProps,l=!1,u=(n.flags&128)!==0,g;if((g=u)||(g=t!==null&&t.memoizedState===null?!1:(Nn.current&2)!==0),g&&(l=!0,n.flags&=-129),g=(n.flags&32)!==0,n.flags&=-33,t===null){if(ye){if(l?$a(n):ts(),(t=tn)?(t=Kv(t,Mi),t=t!==null&&t.data!=="&"?t:null,t!==null&&(n.memoizedState={dehydrated:t,treeContext:Xa!==null?{id:ji,overflow:Zi}:null,retryLane:536870912,hydrationErrors:null},a=og(t),a.return=n,n.child=a,En=n,tn=null)):t=null,t===null)throw qa(n);return oh(t)?n.lanes=32:n.lanes=536870912,null}return u=s.children,s=s.fallback,l?(ts(),l=n.mode,u=Tc({mode:"hidden",children:u},l),s=Cs(s,l,a,null),u.return=n,s.return=n,u.sibling=s,n.child=u,s=n.child,s.memoizedState=od(a),s.childLanes=ld(t,g,a),n.memoizedState=rd,jo(null,s)):($a(n),cd(n,u))}var A=t.memoizedState;if(A!==null){var z=A.dehydrated;if(z!==null)return Py(t,n,u,g,s,z,A,a)}return l?(ts(),l=s.fallback,u=n.mode,A=t.child,z=A.sibling,s=xa(A,{mode:"hidden",children:s.children}),s.subtreeFlags=A.subtreeFlags&1206910976,z!==null?l=xa(z,l):(l=Cs(l,u,a,null),l.flags|=2),l.return=n,s.return=n,s.sibling=l,n.child=s,jo(null,s),s=n.child,l=t.child.memoizedState,l===null?l=od(a):(u=l.cachePool,u!==null?(A=hn._currentValue,u=u.parent!==A?{parent:A,pool:A}:u):u=gg(),l={baseLanes:l.baseLanes|a,cachePool:u}),s.memoizedState=l,s.childLanes=ld(t,g,a),n.memoizedState=rd,jo(t.child,s)):($a(n),a=t.child,t=a.sibling,a=xa(a,{mode:"visible",children:s.children}),a.return=n,a.sibling=null,t!==null&&(g=n.deletions,g===null?(n.deletions=[t],n.flags|=16):g.push(t)),n.child=a,n.memoizedState=null,a)}function cd(t,n){return n=Tc({mode:"visible",children:n},t.mode),n.return=t,t.child=n}function Tc(t,n){return t=jn(22,t,null,n),t.lanes=0,t}function Ac(t,n,a){return zs(n,t.child,null,a),t=cd(n,n.pendingProps.children),t.flags|=2,n.memoizedState=null,t}function Py(t,n,a,s,l,u,g,A){if(a)return n.flags&256?($a(n),n.flags&=-257,Ac(t,n,A)):n.memoizedState!==null?(ts(),n.child=t.child,n.flags|=128,null):(ts(),u=l.fallback,g=n.mode,l=Tc({mode:"visible",children:l.children},g),u=Cs(u,g,A,null),u.flags|=2,l.return=n,u.return=n,l.sibling=u,n.child=l,zs(n,t.child,null,A),l=n.child,l.memoizedState=od(A),l.childLanes=ld(t,s,A),n.memoizedState=rd,jo(null,l));if($a(n),oh(u)){if(s=u.nextSibling&&u.nextSibling.dataset,s)var z=s.dgst;return s=z,s!==""&&(l=Error(r(419)),l.stack="",l.digest=s,Io({value:l,source:null,stack:null})),Ac(t,n,A)}if(mn||Us(t,n,A,!1),s=(A&t.childLanes)!==0,mn||s){if(Ja.current!==null)return Ac(t,n,A);if(s=Qe,s!==null&&(l=Eo(s,A),l!==0&&l!==g.retryLane))throw g.retryLane=l,ws(t,l),Jn(s,t,l),ad;return rh(u)||Xc(),Ac(t,n,A)}return rh(u)?(n.flags|=192,n.child=t.child,null):(t=g.treeContext,tn=bi(u.nextSibling),En=n,ye=!0,Wa=null,Mi=!1,t!==null&&ug(n,t),n=cd(n,l.children),n.flags|=134221824,n)}function C0(t,n,a){t.lanes|=n;var s=t.alternate;s!==null&&(s.lanes|=n),ac(t.return,n,a)}function N0(t){for(var n=null;t!==null;){var a=t.alternate;a!==null&&hc(a)===null&&(n=t),t=t.sibling}return n}function Rc(t,n,a,s,l,u){var g=t.memoizedState;g===null?t.memoizedState={isBackwards:n,rendering:null,renderingStartTime:0,last:s,tail:a,tailMode:l,treeForkCount:u}:(g.isBackwards=n,g.rendering=null,g.renderingStartTime=0,g.last=s,g.tail=a,g.tailMode=l,g.treeForkCount=u)}function ud(t){var n=t.child;for(t.child=null;n!==null;){var a=n.sibling;n.sibling=t.child,t.child=n,n=a}}function fd(t,n,a){var s=n.pendingProps,l=s.revealOrder,u=s.tail;s=s.children;var g=Nn.current;if(n.flags&128)return Xo(n,g),null;var A=(g&2)!==0;if(A?(g=g&1|2,n.flags|=128):g&=1,Xo(n,g),l==="backwards"&&t!==null?(ud(t),Sn(t,n,s,a),ud(t)):Sn(t,n,s,a),s=ye?Po:0,!A&&t!==null&&(t.flags&128)!==0)t:for(t=n.child;t!==null;){if(t.tag===13)t.memoizedState!==null&&C0(t,a,n);else if(t.tag===19)C0(t,a,n);else if(t.child!==null){t.child.return=t,t=t.child;continue}if(t===n)break t;for(;t.sibling===null;){if(t.return===null||t.return===n)break t;t=t.return}t.sibling.return=t.return,t=t.sibling}switch(l){case"backwards":a=N0(n.child),a===null?(l=n.child,n.child=null):(l=a.sibling,a.sibling=null,ud(n)),Rc(n,!0,l,null,u,s);break;case"unstable_legacy-backwards":for(a=null,l=n.child,n.child=null;l!==null;){if(t=l.alternate,t!==null&&hc(t)===null){n.child=l;break}t=l.sibling,l.sibling=a,a=l,l=t}Rc(n,!0,a,null,u,s);break;case"together":Rc(n,!1,null,null,void 0,s);break;case"independent":n.memoizedState=null;break;default:a=N0(n.child),a===null?(l=n.child,n.child=null):(l=a.sibling,a.sibling=null),Rc(n,!1,l,a,u,s)}return n.child}function D0(t,n,a){var s=n.pendingProps;return Ya(n,n.type,s.value),Sn(t,n,s.children,a),n.child}function Ta(t,n,a){if(t!==null&&(n.dependencies=t.dependencies),as|=n.lanes,(a&n.childLanes)===0)if(t!==null){if(Us(t,n,a,!1),(a&n.childLanes)===0)return null}else return null;if(t!==null&&n.child!==t.child)throw Error(r(153));if(n.child!==null){for(t=n.child,a=xa(t,t.pendingProps),n.child=a,a.return=n;t.sibling!==null;)t=t.sibling,a=a.sibling=xa(t,t.pendingProps),a.return=n;a.sibling=null}return n.child}function dd(t,n){return(t.lanes&n)!==0?!0:(t=t.dependencies,!!(t!==null&&sc(t)))}function Iy(t,n,a){switch(n.tag){case 3:Z(n,n.stateNode.containerInfo),Ya(n,hn,t.memoizedState.cache),Ns();break;case 27:case 5:Me(n);break;case 4:Z(n,n.stateNode.containerInfo);break;case 10:Ya(n,n.type,n.memoizedProps.value);break;case 31:if(n.memoizedState!==null)return n.flags|=128,Bf(n),null;break;case 13:var s=n.memoizedState;if(s!==null){if(s.dehydrated!==null)return $a(n),n.flags|=128,null;s=Us(t,n,a,!1);var l=n.child.childLanes;return s||(a&l)!==0?w0(t,n,a):($a(n),t=Ta(t,n,a),t!==null?t.sibling:null)}$a(n);break;case 19:if(n.flags&128)return fd(t,n,a);if(l=(t.flags&128)!==0,s=(a&n.childLanes)!==0,s||(Us(t,n,a,!1),s=(a&n.childLanes)!==0),l){if(s)return fd(t,n,a);n.flags|=128}if(l=n.memoizedState,l!==null&&(l.rendering=null,l.tail=null,l.lastEffect=null),Xo(n,Nn.current),s)break;return null;case 22:return n.lanes=0,M0(t,n,a,n.pendingProps);case 24:Ya(n,hn,t.memoizedState.cache)}return Ta(t,n,a)}function U0(t,n,a){if(t!==null)if(t.memoizedProps!==n.pendingProps)mn=!0;else{if(!dd(t,a)&&(n.flags&128)===0)return mn=!1,Iy(t,n,a);mn=(t.flags&131072)!==0}else mn=!1,ye&&(n.flags&1048576)!==0&&cg(n,Po,n.index);switch(n.lanes=0,n.tag){case 16:t:{var s=n.pendingProps;if(t=Is(n.elementType),n.type=t,typeof t=="function")xf(t)?(s=Hs(t,s),n.tag=1,n=A0(null,n,t,s,a)):(n.tag=0,n=sd(null,n,t,s,a));else{if(t!=null){var l=t.$$typeof;if(l===Y){n.tag=11,n=x0(null,n,t,s,a);break t}else if(l===ot){n.tag=14,n=S0(null,n,t,s,a);break t}else if(l===$){n.tag=10,n.type=t,n=D0(null,n,a);break t}}throw n=At(t)||t,Error(r(306,n,""))}}return n;case 0:return sd(t,n,n.type,n.pendingProps,a);case 1:return s=n.type,l=Hs(s,n.pendingProps),A0(t,n,s,l,a);case 3:t:{if(Z(n,n.stateNode.containerInfo),t===null)throw Error(r(387));s=n.pendingProps;var u=n.memoizedState;l=u.element,Uf(t,n),ko(n,s,null,a);var g=n.memoizedState;if(s=g.cache,Ya(n,hn,s),s!==u.cache&&Af(n,[hn],a,!0),Vo(),s=g.element,u.isDehydrated)if(u={element:s,isDehydrated:!1,cache:g.cache},n.updateQueue.baseState=u,n.memoizedState=u,n.flags&256){n=R0(t,n,s,a);break t}else if(s!==l){l=xi(Error(r(424)),n),Io(l),n=R0(t,n,s,a);break t}else{switch(t=n.stateNode.containerInfo,t.nodeType){case 9:t=t.body;break;default:t=t.nodeName==="HTML"?t.ownerDocument.body:t}for(tn=bi(t.firstChild),En=n,ye=!0,Wa=null,Mi=!0,a=Mg(n,null,s,a),n.child=a;a;)a.flags=a.flags&-3|134221824,a=a.sibling}else{if(Ns(),s===l){n=Ta(t,n,a);break t}Sn(t,n,s,a)}n=n.child}return n;case 26:return Tr(t,n),t===null?(a=i_(n.type,null,n.pendingProps,null))?n.memoizedState=a:ye||(n.stateNode=Iv(n.type,n.pendingProps,Ue.current,n)):n.memoizedState=i_(n.type,t.memoizedProps,n.pendingProps,t.memoizedState),null;case 27:return Me(n),t===null&&ye&&(s=n.stateNode=$v(n.type,n.pendingProps,Ue.current),En=n,Mi=!0,l=tn,ls(n.type)?(lh=l,tn=bi(s.firstChild)):tn=l),Sn(t,n,n.pendingProps.children,a),Tr(t,n),t===null&&(n.flags|=4194304),n.child;case 5:return t===null&&ye&&((l=s=tn)&&(s=NM(s,n.type,n.pendingProps,Mi),s!==null?(n.stateNode=s,En=n,tn=bi(s.firstChild),Mi=!1,l=!0):l=!1),l||qa(n)),Me(n),l=n.type,u=n.pendingProps,g=t!==null?t.memoizedProps:null,s=u.children,$d(l,u)?s=null:g!==null&&$d(l,g)&&(n.flags|=32),n.memoizedState!==null&&(l=Hf(t,n,Ty,null,null,a),kr._currentValue=l),Tr(t,n),Sn(t,n,s,a),n.child;case 6:return t===null&&ye&&((t=a=tn)&&(a=DM(a,n.pendingProps,Mi),a!==null?(n.stateNode=a,En=n,tn=null,t=!0):t=!1),t||qa(n)),null;case 13:return w0(t,n,a);case 4:return Z(n,n.stateNode.containerInfo),s=n.pendingProps,t===null?n.child=zs(n,null,s,a):Sn(t,n,s,a),n.child;case 11:return x0(t,n,n.type,n.pendingProps,a);case 7:return s=n.pendingProps,Tr(t,n),Sn(t,n,s,a),n.child;case 8:return Sn(t,n,n.pendingProps.children,a),n.child;case 12:return Sn(t,n,n.pendingProps.children,a),n.child;case 10:return D0(t,n,a);case 9:return l=n.type._context,s=n.pendingProps.children,Ls(n),l=wn(l),s=s(l),n.flags|=1,Sn(t,n,s,a),n.child;case 14:return S0(t,n,n.type,n.pendingProps,a);case 15:return y0(t,n,n.type,n.pendingProps,a);case 19:return fd(t,n,a);case 31:return Oy(t,n,a);case 22:return M0(t,n,a,n.pendingProps);case 24:return Ls(n),s=wn(hn),t===null?(l=Cf(),l===null&&(l=Qe,u=Rf(),l.pooledCache=u,u.refCount++,u!==null&&(l.pooledCacheLanes|=a),l=u),n.memoizedState={parent:s,cache:l},Df(n),Ya(n,hn,l)):((t.lanes&a)!==0&&(Uf(t,n),ko(n,null,null,a),Vo()),l=t.memoizedState,u=n.memoizedState,l.parent!==s?(l={parent:s,cache:s},n.memoizedState=l,n.lanes===0&&(n.memoizedState=n.updateQueue.baseState=l),Ya(n,hn,s)):(s=u.cache,Ya(n,hn,s),s!==l.cache&&Af(n,[hn],a,!0))),Sn(t,n,n.pendingProps.children,a),n.child;case 30:return n.stateNode===null&&(n.stateNode={autoName:null,paired:null,clones:null,ref:null}),s=n.pendingProps,s.name!=null&&s.name!=="auto"?n.flags|=t===null?18882560:18874368:ye&&nc(n),t!==null&&t.memoizedProps.name!==s.name?n.flags|=4194816:Tr(t,n),Sn(t,n,s.children,a),n.child;case 29:throw n.pendingProps}throw Error(r(156,n.tag))}function Aa(t){t.flags|=4}function hd(t,n,a,s,l){var u;if((u=(t.mode&32)!==0)&&(u=a===null?o_(n,s):o_(n,s)&&(s.src!==a.src||s.srcSet!==a.srcSet)),u){if(t.flags|=16777216,(l&335544128)===l)if(t.stateNode.complete)t.flags|=8192;else if(hv())t.flags|=8192;else throw Bs=cc,Nf}else t.flags&=-16777217}function L0(t,n){if(n.type!=="stylesheet"||(n.state.loading&4)!==0)t.flags&=-16777217;else if(t.flags|=16777216,!l_(n))if(hv())t.flags|=8192;else throw Bs=cc,Nf}function wc(t,n){n!==null&&(t.flags|=4),t.flags&16384&&(n=t.tag!==22?yo():536870912,t.lanes|=n,Nr|=n)}function Zo(t,n){if(!ye)switch(t.tailMode){case"visible":break;case"collapsed":for(var a=t.tail,s=null;a!==null;)a.alternate!==null&&(s=a),a=a.sibling;s===null?n||t.tail===null?t.tail=null:t.tail.sibling=null:s.sibling=null;break;default:for(n=t.tail,a=null;n!==null;)n.alternate!==null&&(a=n),n=n.sibling;a===null?t.tail=null:a.sibling=null}}function en(t){var n=t.alternate!==null&&t.alternate.child===t.child,a=0,s=0;if(n)for(var l=t.child;l!==null;)a|=l.lanes|l.childLanes,s|=l.subtreeFlags&1206910976,s|=l.flags&1206910976,l.return=t,l=l.sibling;else for(l=t.child;l!==null;)a|=l.lanes|l.childLanes,s|=l.subtreeFlags,s|=l.flags,l.return=t,l=l.sibling;return t.subtreeFlags|=s,t.childLanes=a,n}function By(t,n,a){var s=n.pendingProps;switch(Mf(n),n.tag){case 16:case 15:case 0:case 11:case 7:case 8:case 12:case 9:case 14:return en(n),null;case 1:return en(n),null;case 3:return a=n.stateNode,s=null,t!==null&&(s=t.memoizedState.cache),n.memoizedState.cache!==s&&(n.flags|=2048),Ma(hn),Je(),a.pendingContext&&(a.context=a.pendingContext,a.pendingContext=null),(t===null||t.child===null)&&(_r(n)?Aa(n):t===null||t.memoizedState.isDehydrated&&(n.flags&256)===0||(n.flags|=1024,bf())),en(n),null;case 26:var l=n.type,u=n.memoizedState;return t===null?(Aa(n),u!==null?(en(n),L0(n,u)):(en(n),hd(n,l,null,s,a))):u?u!==t.memoizedState?(Aa(n),en(n),L0(n,u)):(en(n),n.flags&=-16777217):(t=t.memoizedProps,t!==s&&Aa(n),en(n),hd(n,l,t,s,a)),null;case 27:if(I(n),a=Ue.current,l=n.type,t!==null&&n.stateNode!=null)t.memoizedProps!==s&&Aa(n);else{if(!s){if(n.stateNode===null)throw Error(r(166));return en(n),n.subtreeFlags&=-33554433,null}t=Be.current,_r(n)?fg(n):(t=$v(l,s,a),n.stateNode=t,Aa(n))}return en(n),n.subtreeFlags&=-33554433,null;case 5:if(I(n),l=n.type,t!==null&&n.stateNode!=null)t.memoizedProps!==s&&Aa(n);else{if(!s){if(n.stateNode===null)throw Error(r(166));return en(n),n.subtreeFlags&=-33554433,null}if(u=Be.current,_r(n))fg(n);else{var g=rl(Ue.current);switch(u){case 1:u=g.createElementNS("http://www.w3.org/2000/svg",l);break;case 2:u=g.createElementNS("http://www.w3.org/1998/Math/MathML",l);break;default:switch(l){case"svg":u=g.createElementNS("http://www.w3.org/2000/svg",l);break;case"math":u=g.createElementNS("http://www.w3.org/1998/Math/MathML",l);break;case"script":u=g.createElement("div"),u.innerHTML="<script><\/script>",u=u.removeChild(u.firstChild);break;case"select":u=typeof s.is=="string"?g.createElement("select",{is:s.is}):g.createElement("select"),s.multiple?u.multiple=!0:s.size&&(u.size=s.size);break;default:u=typeof s.is=="string"?g.createElement(l,{is:s.is}):g.createElement(l)}}u[T]=n,u[V]=s;t:for(g=n.child;g!==null;){if(g.tag===5||g.tag===6)u.appendChild(g.stateNode);else if(g.tag!==4&&g.tag!==27&&g.child!==null){g.child.return=g,g=g.child;continue}if(g===n)break t;for(;g.sibling===null;){if(g.return===null||g.return===n)break t;g=g.return}g.sibling.return=g.return,g=g.sibling}n.stateNode=u;t:switch(Un(u,l,s),l){case"button":case"input":case"select":case"textarea":s=!!s.autoFocus;break t;case"img":s=!0;break t;default:s=!1}s&&Aa(n)}}return en(n),n.subtreeFlags&=-33554433,hd(n,n.type,t===null?null:t.memoizedProps,n.pendingProps,a),null;case 6:if(t&&n.stateNode!=null)t.memoizedProps!==s&&Aa(n);else{if(typeof s!="string"&&n.stateNode===null)throw Error(r(166));if(t=Ue.current,_r(n)){if(t=n.stateNode,a=n.memoizedProps,s=null,l=En,l!==null)switch(l.tag){case 27:case 5:s=l.memoizedProps}t[T]=n,t=!!(t.nodeValue===a||s!==null&&s.suppressHydrationWarning===!0||Uv(t.nodeValue,a)),t||qa(n,!0)}else t=rl(t).createTextNode(s),t[T]=n,n.stateNode=t}return en(n),null;case 31:if(a=n.memoizedState,t===null||t.memoizedState!==null){if(s=_r(n),a!==null){if(t===null){if(!s)throw Error(r(318));if(t=n.memoizedState,t=t!==null?t.dehydrated:null,!t)throw Error(r(557));t[T]=n}else Ns(),(n.flags&128)===0&&(n.memoizedState=null),n.flags|=4;en(n),t=!1}else a=bf(),t!==null&&t.memoizedState!==null&&(t.memoizedState.hydrationErrors=a),t=!0;if(!t)return n.flags&256?(ri(n),n):(ri(n),null);if((n.flags&128)!==0)throw Error(r(558))}return en(n),null;case 13:if(s=n.memoizedState,t===null||t.memoizedState!==null&&t.memoizedState.dehydrated!==null){if(l=_r(n),s!==null&&s.dehydrated!==null){if(t===null){if(!l)throw Error(r(318));if(l=n.memoizedState,l=l!==null?l.dehydrated:null,!l)throw Error(r(317));l[T]=n}else Ns(),(n.flags&128)===0&&(n.memoizedState=null),n.flags|=4;en(n),l=!1}else l=bf(),t!==null&&t.memoizedState!==null&&(t.memoizedState.hydrationErrors=l),l=!0;if(!l)return n.flags&256?(ri(n),n):(ri(n),null)}return ri(n),(n.flags&128)!==0?(n.lanes=a,n):(a=s!==null,t=t!==null&&t.memoizedState!==null,a&&(s=n.child,l=null,s.alternate!==null&&s.alternate.memoizedState!==null&&s.alternate.memoizedState.cachePool!==null&&(l=s.alternate.memoizedState.cachePool.pool),u=null,s.memoizedState!==null&&s.memoizedState.cachePool!==null&&(u=s.memoizedState.cachePool.pool),u!==l&&(s.flags|=2048)),a!==t&&a&&(n.child.flags|=8192),wc(n,n.updateQueue),en(n),null);case 4:return Je(),t===null&&jd(n.stateNode.containerInfo),n.flags|=67108864,en(n),null;case 10:return Ma(n.type),en(n),null;case 19:if(zf(n),s=n.memoizedState,s===null)return en(n),null;if(l=(n.flags&128)!==0,u=s.rendering,u===null)if(l)Zo(s,!1);else{if(fn!==0||t!==null&&(t.flags&128)!==0)for(t=n.child;t!==null;){if(u=hc(t),u!==null){for(n.flags|=128,Zo(s,!1),t=u.updateQueue,n.updateQueue=t,wc(n,t),n.subtreeFlags=0,t=a,a=n.child;a!==null;)rg(a,t),a=a.sibling;return Xo(n,Nn.current&1|2),ye&&Sa(n,s.treeForkCount),n.child}t=t.sibling}s.tail!==null&&kt()>Hc&&(n.flags|=128,l=!0,Zo(s,!1),n.lanes=4194304)}else{if(!l)if(t=hc(u),t!==null){if(n.flags|=128,l=!0,t=t.updateQueue,n.updateQueue=t,wc(n,t),Zo(s,!0),s.tail===null&&s.tailMode!=="collapsed"&&s.tailMode!=="visible"&&!u.alternate&&!ye)return en(n),null}else 2*kt()-s.renderingStartTime>Hc&&a!==536870912&&(n.flags|=128,l=!0,Zo(s,!1),n.lanes=4194304);s.isBackwards?(u.sibling=n.child,n.child=u):(t=s.last,t!==null?t.sibling=u:n.child=u,s.last=u)}if(s.tail!==null){t=s.tail;t:{for(a=t;a!==null;){if(a.alternate!==null){a=!1;break t}a=a.sibling}a=!0}return s.rendering=t,s.tail=t.sibling,s.renderingStartTime=kt(),t.sibling=null,u=Nn.current,u=l?u&1|2:u&1,s.tailMode==="visible"||s.tailMode==="collapsed"||!a||ye?Xo(n,u):(a=u,ne(Cn,n),ne(Nn,a),Pn===null&&(Pn=n)),ye&&Sa(n,s.treeForkCount),t}return en(n),null;case 22:case 23:return ri(n),If(),s=n.memoizedState!==null,t!==null?t.memoizedState!==null!==s&&(n.flags|=8192):s&&(n.flags|=8192),s?(a&536870912)!==0&&(n.flags&128)===0&&(en(n),n.subtreeFlags&6&&(n.flags|=8192)):en(n),a=n.updateQueue,a!==null&&wc(n,a.retryQueue),a=null,t!==null&&t.memoizedState!==null&&t.memoizedState.cachePool!==null&&(a=t.memoizedState.cachePool.pool),s=null,n.memoizedState!==null&&n.memoizedState.cachePool!==null&&(s=n.memoizedState.cachePool.pool),s!==a&&(n.flags|=2048),t!==null&&te(Ps),null;case 24:return a=null,t!==null&&(a=t.memoizedState.cache),n.memoizedState.cache!==a&&(n.flags|=2048),Ma(hn),en(n),null;case 25:return null;case 30:return n.flags|=33554432,en(n),null}throw Error(r(156,n.tag))}function zy(t,n){switch(Mf(n),n.tag){case 1:return t=n.flags,t&65536?(n.flags=t&-65537|128,n):null;case 3:return Ma(hn),Je(),t=n.flags,(t&65536)!==0&&(t&128)===0?(n.flags=t&-65537|128,n):null;case 26:case 27:case 5:return I(n),null;case 31:if(n.memoizedState!==null){if(ri(n),n.alternate===null)throw Error(r(340));Ns()}return t=n.flags,t&65536?(n.flags=t&-65537|128,n):null;case 13:if(ri(n),t=n.memoizedState,t!==null&&t.dehydrated!==null){if(n.alternate===null)throw Error(r(340));Ns()}return t=n.flags,t&65536?(n.flags=t&-65537|128,n):null;case 19:return zf(n),t=n.flags,t&65536?(n.flags=t&-65537|128,t=n.memoizedState,t!==null&&(t.rendering=null,t.tail=null),n.flags|=4,n):null;case 4:return Je(),null;case 10:return Ma(n.type),null;case 22:case 23:return ri(n),If(),t!==null&&te(Ps),t=n.flags,t&65536?(n.flags=t&-65537|128,n):null;case 24:return Ma(hn),null;case 25:return null;default:return null}}function O0(t,n){switch(Mf(n),n.tag){case 3:Ma(hn),Je();break;case 26:case 27:case 5:I(n);break;case 4:Je();break;case 31:n.memoizedState!==null&&ri(n);break;case 13:ri(n);break;case 19:zf(n);break;case 10:Ma(n.type);break;case 22:case 23:ri(n),If(),t!==null&&te(Ps);break;case 24:Ma(hn)}}function Ko(t,n){try{var a=n.updateQueue,s=a!==null?a.lastEffect:null;if(s!==null){var l=s.next;a=l;do{if((a.tag&t)===t){s=void 0;var u=a.create,g=a.inst;s=u(),g.destroy=s}a=a.next}while(a!==l)}}catch(A){We(n,n.return,A)}}function es(t,n,a){try{var s=n.updateQueue,l=s!==null?s.lastEffect:null;if(l!==null){var u=l.next;s=u;do{if((s.tag&t)===t){var g=s.inst,A=g.destroy;if(A!==void 0){g.destroy=void 0,l=n;var z=a,it=A;try{it()}catch(ht){We(l,z,ht)}}}s=s.next}while(s!==u)}}catch(ht){We(n,n.return,ht)}}function P0(t){var n=t.updateQueue;if(n!==null){var a=t.stateNode;try{bg(n,a)}catch(s){We(t,t.return,s)}}}function I0(t,n,a){a.props=Hs(t.type,t.memoizedProps),a.state=t.memoizedState;try{a.componentWillUnmount()}catch(s){We(t,n,s)}}function Ki(t,n){try{var a=t.ref;if(a!==null){switch(t.tag){case 26:case 27:case 5:var s=t.stateNode;break;case 30:var l=t.stateNode,u=va(t.memoizedProps,l);(l.ref===null||l.ref.name!==u)&&(l.ref=kv(u)),s=l.ref;break;case 7:if(t.stateNode===null){var g=new fi(t);v(t.child,!1,wM,g,void 0,void 0),t.stateNode=g}s=t.stateNode;break;default:s=t.stateNode}typeof a=="function"?t.refCleanup=a(s):a.current=s}}catch(A){We(t,n,A)}}function Dn(t,n){var a=t.ref,s=t.refCleanup;if(a!==null)if(typeof s=="function")try{s()}catch(l){We(t,n,l)}finally{t.refCleanup=null,t=t.alternate,t!=null&&(t.refCleanup=null)}else if(typeof a=="function")try{a(null)}catch(l){We(t,n,l)}else a.current=null}function Cc(t,n){if((t.tag===5||t.tag===27||t.tag===6)&&t.alternate===null&&n!==null)for(var a=0;a<n.length;a++)Zv(t.stateNode,n[a])}function B0(t){for(var n=t.return;n!==null&&(md(n)&&Zv(t.stateNode,n.stateNode),!pd(n));)n=n.return}function Qo(t){for(var n=t.return;n!==null&&(md(n)&&CM(t.stateNode,n.stateNode),!pd(n));)n=n.return}function pd(t){return t.tag===5||t.tag===3||t.tag===27}function md(t){return t&&t.tag===7&&t.stateNode!==null}function gd(t){var n=t.type,a=t.memoizedProps,s=t.stateNode;try{t:switch(n){case"button":case"input":case"select":case"textarea":a.autoFocus&&s.focus();break t;case"img":a.src?s.src=a.src:a.srcSet&&(s.srcset=a.srcSet)}}catch(l){We(t,t.return,l)}}function vd(t,n,a){try{var s=t.stateNode;uM(s,t.type,a,n),s[V]=n}catch(l){We(t,t.return,l)}}function z0(t){return t.tag===5||t.tag===3||t.tag===26||t.tag===27&&ls(t.type)||t.tag===4}function _d(t){t:for(;;){for(;t.sibling===null;){if(t.return===null||z0(t.return))return null;t=t.return}for(t.sibling.return=t.return,t=t.sibling;t.tag!==5&&t.tag!==6&&t.tag!==18;){if(t.tag===27&&ls(t.type)||t.flags&2||t.child===null||t.tag===4)continue t;t.child.return=t,t=t.child}if(!(t.flags&2))return t.stateNode}}function xd(t,n,a,s){var l=t.tag;if(l===5||l===6)l=t.stateNode,n?(a.nodeType===9?a.body:a.nodeName==="HTML"?a.ownerDocument.body:a).insertBefore(l,n):(n=a.nodeType===9?a.body:a.nodeName==="HTML"?a.ownerDocument.body:a,n.appendChild(l),a=a._reactRootContainer,a!=null||n.onclick!==null||(n.onclick=Yi)),Cc(t,s),Se=!0;else if(l!==4&&(l===27&&(Cc(t,s),s=null,ls(t.type)&&(a=t.stateNode,n=null)),t=t.child,t!==null))for(xd(t,n,a,s),t=t.sibling;t!==null;)xd(t,n,a,s),t=t.sibling}function Nc(t,n,a,s){var l=t.tag;if(l===5||l===6)l=t.stateNode,n?a.insertBefore(l,n):a.appendChild(l),Cc(t,s),Se=!0;else if(l!==4&&(l===27&&(Cc(t,s),s=null,ls(t.type)&&(a=t.stateNode)),t=t.child,t!==null))for(Nc(t,n,a,s),t=t.sibling;t!==null;)Nc(t,n,a,s),t=t.sibling}function F0(t){var n=t.stateNode,a=t.memoizedProps;try{for(var s=t.type,l=n.attributes;l.length;)n.removeAttributeNode(l[0]);Un(n,s,a),n[T]=t,n[V]=a}catch(u){We(t,t.return,u)}}var Dc=!1,oi=null;function H0(t){(t.tag===30||(t.subtreeFlags&33554432)!==0)&&(Dc=!0)}var Qi=null;function G0(){var t=Qi;return Qi=null,t}var Zn=0;function Ar(t,n,a,s,l){return Zn=0,V0(t.child,n,a,s,l)}function V0(t,n,a,s,l){for(var u=!1;t!==null;){if(t.tag===5){var g=t.stateNode;if(s!==null){var A=nh(g);s.push(A),A.view&&(u=!0)}else u||nh(g).view&&(u=!0);Dc=!0,Gv(g,Zn===0?n:n+"_"+Zn,a),Zn++}else(t.tag!==22||t.memoizedState===null)&&(t.tag===30&&l||V0(t.child,n,a,s,l)&&(u=!0));t=t.sibling}return u}function Ji(t,n){for(;t!==null;)t.tag===5?Vv(t.stateNode,t.memoizedProps):(t.tag!==22||t.memoizedState===null)&&(t.tag===30&&n||Ji(t.child,n)),t=t.sibling}function Uc(t){if((t.subtreeFlags&18874368)!==0)for(t=t.child;t!==null;){if((t.tag!==22||t.memoizedState===null)&&(Uc(t),t.tag===30&&(t.flags&18874368)!==0&&t.stateNode.paired)){var n=t.memoizedProps;if(n.name==null||n.name==="auto")throw Error(r(544));var a=n.name;n=_a(n.default,n.share),n!=="none"&&(Ar(t,a,n,null,!1)||Ji(t.child,!1))}t=t.sibling}}function Sd(t,n){if(t.tag===30){var a=t.stateNode,s=t.memoizedProps,l=va(s,a),u=_a(s.default,a.paired?s.share:s.enter);u!=="none"?Ar(t,l,u,null,!1)?(Uc(t),a.paired||n||Or(t,s.onEnter)):Ji(t.child,!1):Uc(t)}else if((t.subtreeFlags&33554432)!==0)for(t=t.child;t!==null;)Sd(t,n),t=t.sibling;else Uc(t)}function yd(t){if(oi!==null&&oi.size!==0){var n=oi;if((t.subtreeFlags&18874368)!==0)for(t=t.child;t!==null;){if(t.tag!==22||t.memoizedState===null){if(t.tag===30&&(t.flags&18874368)!==0){var a=t.memoizedProps,s=a.name;if(s!=null&&s!=="auto"){var l=n.get(s);if(l!==void 0){var u=_a(a.default,a.share);if(u!=="none"&&(Ar(t,s,u,null,!1)?(u=t.stateNode,l.paired=u,u.paired=l,Or(t,a.onShare)):Ji(t.child,!1)),n.delete(s),n.size===0)break}}}yd(t)}t=t.sibling}}}function Md(t){if(t.tag===30){var n=t.memoizedProps,a=va(n,t.stateNode),s=oi!==null?oi.get(a):void 0,l=_a(n.default,s!==void 0?n.share:n.exit);l!=="none"&&(Ar(t,a,l,null,!1)?s!==void 0?(l=t.stateNode,s.paired=l,l.paired=s,oi.delete(a),Or(t,n.onShare)):Or(t,n.onExit):Ji(t.child,!1)),oi!==null&&yd(t)}else if((t.subtreeFlags&33554432)!==0)for(t=t.child;t!==null;)Md(t),t=t.sibling;else oi!==null&&yd(t)}function k0(t){for(t=t.child;t!==null;){if(t.tag===30){var n=t.memoizedProps,a=va(n,t.stateNode);n=_a(n.default,n.update),t.flags&=-5,n!=="none"&&Ar(t,a,n,t.memoizedState=[],!1)}else(t.subtreeFlags&33554432)!==0&&k0(t);t=t.sibling}}function Ed(t){if((t.subtreeFlags&18874368)!==0)for(t=t.child;t!==null;){if(t.tag!==22||t.memoizedState===null){if(t.tag===30&&(t.flags&18874368)!==0){var n=t.stateNode;n.paired!==null&&(n.paired=null,Ji(t.child,!1))}Ed(t)}t=t.sibling}}function Lc(t){if(t.tag===30)t.stateNode.paired=null,Ji(t.child,!1),Ed(t);else if((t.subtreeFlags&33554432)!==0)for(t=t.child;t!==null;)Lc(t),t=t.sibling;else Ed(t)}function X0(t){for(t=t.child;t!==null;)t.tag===30?Ji(t.child,!1):(t.subtreeFlags&33554432)!==0&&X0(t),t=t.sibling}function bd(t,n,a,s,l,u,g){for(var A=!1;n!==null;){if(n.tag===5){var z=n.stateNode;if(u!==null&&Zn<u.length){var it=u[Zn],ht=nh(z);(it.view||ht.view)&&(A=!0);var Et;if(Et=(t.flags&4)===0)if(ht.clip)Et=!0;else{Et=it.rect;var tt=ht.rect;Et=Et.y!==tt.y||Et.x!==tt.x||Et.height!==tt.height||Et.width!==tt.width}Et&&(t.flags|=4),ht.abs?ht=!it.abs:(it=it.rect,ht=ht.rect,ht=it.height!==ht.height||it.width!==ht.width),ht&&(t.flags|=32)}else t.flags|=32;(t.flags&4)!==0&&Gv(z,Zn===0?a:a+"_"+Zn,l),A&&(t.flags&4)!==0||(Qi===null&&(Qi=[]),Qi.push(z,Zn===0?s:s+"_"+Zn,n.memoizedProps)),Zn++}else(n.tag!==22||n.memoizedState===null)&&(n.tag===30&&g?t.flags|=n.flags&32:bd(t,n.child,a,s,l,u,g)&&(A=!0));n=n.sibling}return A}function W0(t,n){for(t=t.child;t!==null;){if(t.tag===30){var a=t.memoizedProps,s=t.stateNode,l=va(a,s),u=_a(a.default,a.update),g;g=t.memoizedState,t.memoizedState=null,s=t;var A=t.child;Zn=0,l=bd(s,A,l,l,u,g,!1),(t.flags&4)!==0&&l&&Or(t,a.onUpdate)}else(t.subtreeFlags&33554432)!==0&&W0(t);t=t.sibling}}var bn=!1,ke=!1,$i=!1,Td=!1,q0=typeof WeakSet=="function"?WeakSet:Set,Tn=null,ta=!1,Jo=!1,Oc=!1,Ad=!1;function Fy(t,n,a){if(t=t.containerInfo,Qd=Xr,t=Km(t),df(t)){if("selectionStart"in t)var s={start:t.selectionStart,end:t.selectionEnd};else t:{s=(s=t.ownerDocument)&&s.defaultView||window;var l=s.getSelection&&s.getSelection();if(l&&l.rangeCount!==0){s=l.anchorNode;var u=l.anchorOffset,g=l.focusNode;l=l.focusOffset;try{s.nodeType,g.nodeType}catch{s=null;break t}var A=0,z=-1,it=-1,ht=0,Et=0,tt=t,ft=null;e:for(;;){for(var Pt;tt!==s||u!==0&&tt.nodeType!==3||(z=A+u),tt!==g||l!==0&&tt.nodeType!==3||(it=A+l),tt.nodeType===3&&(A+=tt.nodeValue.length),(Pt=tt.firstChild)!==null;)ft=tt,tt=Pt;for(;;){if(tt===t)break e;if(ft===s&&++ht===u&&(z=A),ft===g&&++Et===l&&(it=A),(Pt=tt.nextSibling)!==null)break;tt=ft,ft=tt.parentNode}tt=Pt}s=z===-1||it===-1?null:{start:z,end:it}}else s=null}s=s||{start:0,end:0}}else s=null;for(Jd={focusedElem:t,selectionRange:s},Xr=!1,a=(a&335544064)===a,Tn=n,n=a?9270:1024;Tn!==null;){if(t=Tn,a&&(s=t.deletions,s!==null))for(u=0;u<s.length;u++)a&&Md(s[u]);if(t.alternate===null&&(t.flags&2)!==0)a&&H0(t),Pc(a);else{if(t.tag===22){if(s=t.alternate,t.memoizedState!==null){s!==null&&s.memoizedState===null&&a&&Md(s),Pc(a);continue}else if(s!==null&&s.memoizedState!==null){a&&H0(t),Pc(a);continue}}s=t.child,(t.subtreeFlags&n)!==0&&s!==null?(s.return=t,Tn=s):(a&&k0(t),Pc(a))}}oi=null}function Pc(t){for(;Tn!==null;){var n=Tn,a=t,s=n.alternate,l=n.flags;switch(n.tag){case 0:case 11:case 15:break;case 1:if((l&1024)!==0&&s!==null){a=void 0,l=s.memoizedProps,s=s.memoizedState;var u=n.stateNode;try{var g=Hs(n.type,l);a=u.getSnapshotBeforeUpdate(g,s),u.__reactInternalSnapshotBeforeUpdate=a}catch(A){We(n,n.return,A)}}break;case 3:if((l&1024)!==0){if(s=n.stateNode.containerInfo,a=s.nodeType,a===9)sh(s);else if(a===1)switch(s.nodeName){case"HEAD":case"HTML":case"BODY":sh(s);break;default:s.textContent=""}}break;case 5:case 26:case 27:case 6:case 4:case 17:break;case 30:a&&s!==null&&(a=va(s.memoizedProps,s.stateNode),l=n.memoizedProps,l=_a(l.default,l.update),l!=="none"&&Ar(s,a,l,s.memoizedState=[],!0));break;default:if((l&1024)!==0)throw Error(r(163))}if(s=n.sibling,s!==null){s.return=n.return,Tn=s;break}Tn=n.return}}function Y0(t,n,a){var s=a.flags;switch(a.tag){case 0:case 11:case 15:ea(t,a),s&4&&Ko(5,a);break;case 1:if(ea(t,a),s&4)if(t=a.stateNode,n===null)try{t.componentDidMount()}catch(g){We(a,a.return,g)}else{var l=Hs(a.type,n.memoizedProps);n=n.memoizedState;try{t.componentDidUpdate(l,n,t.__reactInternalSnapshotBeforeUpdate)}catch(g){We(a,a.return,g)}}s&64&&P0(a),s&512&&Ki(a,a.return);break;case 3:if(ea(t,a),s&64&&(t=a.updateQueue,t!==null)){if(n=null,a.child!==null)switch(a.child.tag){case 27:case 5:n=a.child.stateNode;break;case 1:n=a.child.stateNode}try{bg(t,n)}catch(g){We(a,a.return,g)}}break;case 27:n===null&&s&4&&F0(a);case 26:case 5:ea(t,a),n===null&&s&4&&gd(a),s&512&&Ki(a,a.return);break;case 12:ea(t,a);break;case 31:ea(t,a),s&4&&Q0(t,a);break;case 13:ea(t,a),s&4&&J0(t,a),s&64&&(t=a.memoizedState,t!==null&&(t=t.dehydrated,t!==null&&(a=Qy.bind(null,a),UM(t,a))));break;case 22:if(s=a.memoizedState!==null||bn,!s){var u=n!==null&&n.memoizedState!==null||ke;n=bn,l=ke,bn=s,(ke=u)&&!l?(s=2,(a.subtreeFlags&8772)!==0&&(s|=1),Di(t,a,s)):ea(t,a),bn=n,ke=l}break;case 30:ea(t,a),s&512&&Ki(a,a.return);break;case 7:s&512&&Ki(a,a.return);default:ea(t,a)}}function Rd(t,n){for(t=t.child;t!==null;)j0(t,n),t=t.sibling}function j0(t,n){switch(t.tag){case 5:case 26:try{var a=t.stateNode;if(n){var s=a.style;typeof s.setProperty=="function"?s.setProperty("display","none","important"):s.display="none"}else{var l=t.stateNode,u=t.memoizedProps.style,g=u!=null&&u.hasOwnProperty("display")?u.display:null;l.style.display=g==null||typeof g=="boolean"?"":(""+g).trim()}}catch(z){We(t,t.return,z)}wd(t,n);break;case 6:try{t.stateNode.nodeValue=n?"":t.memoizedProps,Se=!0}catch(z){We(t,t.return,z)}break;case 18:try{var A=t.stateNode;n?Hv(A,!0):Hv(t.stateNode,!1)}catch(z){We(t,t.return,z)}break;case 22:case 23:t.memoizedState===null&&Rd(t,n);break;default:Rd(t,n)}}function wd(t,n){if(t.subtreeFlags&67108864)for(t=t.child;t!==null;){t:{var a=t,s=n;switch(a.tag){case 4:j0(a,s);break t;case 22:a.memoizedState===null&&wd(a,s);break t;default:wd(a,s)}}t=t.sibling}}function Z0(t){var n=t.alternate;n!==null&&(t.alternate=null,Z0(n)),t.child=null,t.deletions=null,t.sibling=null,t.tag===5&&(n=t.stateNode,n!==null&&Kt(n)),t.stateNode=null,t.return=null,t.dependencies=null,t.memoizedProps=null,t.memoizedState=null,t.pendingProps=null,t.stateNode=null,t.updateQueue=null}var an=null,Kn=!1;function Ci(t,n,a){for(a=a.child;a!==null;)K0(t,n,a),a=a.sibling}function K0(t,n,a){if(Gt&&typeof Gt.onCommitFiberUnmount=="function")try{Gt.onCommitFiberUnmount($t,a)}catch{}switch(a.tag){case 26:ke||Dn(a,n),Ci(t,n,a),a.memoizedState?a.memoizedState.count--:a.stateNode&&!ke&&(a=a.stateNode,a.parentNode.removeChild(a));break;case 27:ke||Dn(a,n),Qo(a);var s=an,l=Kn;ls(a.type)&&(an=a.stateNode,Kn=!1),Ci(t,n,a),t_(a.stateNode,a.type,a.memoizedProps),an=s,Kn=l;break;case 5:ke||Dn(a,n),Qo(a);case 6:if(a.tag===6&&Qo(a),s=an,l=Kn,an=null,Ci(t,n,a),an=s,Kn=l,an!==null)if(Kn)try{(an.nodeType===9?an.body:an.nodeName==="HTML"?an.ownerDocument.body:an).removeChild(a.stateNode),Se=!0}catch(u){We(a,n,u)}else try{an.removeChild(a.stateNode),Se=!0}catch(u){We(a,n,u)}break;case 18:an!==null&&(Kn?(t=an,Fv(t.nodeType===9?t.body:t.nodeName==="HTML"?t.ownerDocument.body:t,a.stateNode),Wr(t)):Fv(an,a.stateNode));break;case 4:s=an,l=Kn,an=a.stateNode.containerInfo,Kn=!0,Ci(t,n,a),an=s,Kn=l;break;case 0:case 11:case 14:case 15:es(2,a,n),ke||es(4,a,n),Ci(t,n,a);break;case 1:ke||(Dn(a,n),s=a.stateNode,typeof s.componentWillUnmount=="function"&&I0(a,n,s)),Ci(t,n,a);break;case 21:Ci(t,n,a);break;case 22:ke=(s=ke)||a.memoizedState!==null,Ci(t,n,a),ke=s;break;case 30:Dn(a,n),Ci(t,n,a);break;case 7:ke||Dn(a,n),Ci(t,n,a);break;default:Ci(t,n,a)}}function Q0(t,n){if(n.memoizedState===null&&(t=n.alternate,t!==null&&(t=t.memoizedState,t!==null))){t=t.dehydrated;try{Wr(t)}catch(a){We(n,n.return,a)}}}function J0(t,n){if(n.memoizedState===null&&(t=n.alternate,t!==null&&(t=t.memoizedState,t!==null&&(t=t.dehydrated,t!==null))))try{Wr(t)}catch(a){We(n,n.return,a)}}function Hy(t){switch(t.tag){case 31:case 13:case 19:var n=t.stateNode;return n===null&&(n=t.stateNode=new q0),n;case 22:return t=t.stateNode,n=t._retryCache,n===null&&(n=t._retryCache=new q0),n;default:throw Error(r(435,t.tag))}}function Ic(t,n){var a=Hy(t);n.forEach(function(s){if(!a.has(s)){a.add(s);var l=Jy.bind(null,t,s);s.then(l,l)}})}function kn(t,n,a){var s=n.deletions;if(s!==null)for(var l=0;l<s.length;l++){var u=s[l],g=t,A=n,z=A;t:for(;z!==null;){switch(z.tag){case 27:if(ls(z.type)){an=z.stateNode,Kn=!1;break t}break;case 5:an=z.stateNode,Kn=!1;break t;case 3:case 4:an=z.stateNode.containerInfo,Kn=!0;break t}z=z.return}if(an===null)throw Error(r(160));K0(g,A,u),an=null,Kn=!1,g=u.alternate,g!==null&&(g.return=null),u.return=null}if(n.subtreeFlags&13886)for(n=n.child;n!==null;)$0(n,t,a),n=n.sibling}var Ni=null;function $0(t,n,a){var s=t.alternate,l=t.flags;switch(t.tag){case 0:case 11:case 14:case 15:if(l&4&&(s=t.updateQueue,s=s!==null?s.events:null,s!==null))for(var u=0;u<s.length;u++){var g=s[u];g.ref.impl=g.nextImpl}kn(n,t,a),Xn(t),l&4&&(es(3,t,t.return),Ko(3,t),es(5,t,t.return));break;case 1:kn(n,t,a),Xn(t),l&512&&(ke||s===null||Dn(s,s.return)),l&64&&bn&&(t=t.updateQueue,t!==null&&(n=t.callbacks,n!==null&&(a=t.shared.hiddenCallbacks,t.shared.hiddenCallbacks=a===null?n:a.concat(n))));break;case 26:if(u=Ni,kn(n,t,a),Xn(t),l&512&&(ke||s===null||Dn(s,s.return)),l&4)if(l=s!==null?s.memoizedState:null,a=t.memoizedState,s===null)if(a===null)if(t.stateNode===null)if(bn)t.stateNode=Iv(t.type,t.memoizedProps,n.containerInfo,t);else{t:{n=t.type,a=t.memoizedProps,l=u.ownerDocument||u;e:switch(n){case"title":s=l.getElementsByTagName("title")[0],(!s||s[Ot]||s[T]||s.namespaceURI==="http://www.w3.org/2000/svg"||s.hasAttribute("itemprop"))&&(s=l.createElement(n),l.head.insertBefore(s,l.querySelector("head > title"))),Un(s,n,a),s[T]=t,xe(s),n=s;break t;case"link":if(u=r_("link","href",l).get(n+(a.href||""))){for(g=0;g<u.length;g++)if(s=u[g],s.getAttribute("href")===(a.href==null||a.href===""?null:a.href)&&s.getAttribute("rel")===(a.rel==null?null:a.rel)&&s.getAttribute("title")===(a.title==null?null:a.title)&&s.getAttribute("crossorigin")===(a.crossOrigin==null?null:a.crossOrigin)){u.splice(g,1);break e}}s=l.createElement(n),Un(s,n,a),l.head.appendChild(s);break;case"meta":if(u=r_("meta","content",l).get(n+(a.content||""))){for(g=0;g<u.length;g++)if(s=u[g],s.getAttribute("content")===(a.content==null?null:""+a.content)&&s.getAttribute("name")===(a.name==null?null:a.name)&&s.getAttribute("property")===(a.property==null?null:a.property)&&s.getAttribute("http-equiv")===(a.httpEquiv==null?null:a.httpEquiv)&&s.getAttribute("charset")===(a.charSet==null?null:a.charSet)){u.splice(g,1);break e}}s=l.createElement(n),Un(s,n,a),l.head.appendChild(s);break;default:throw Error(r(468,n))}s[T]=t,xe(s),n=s}t.stateNode=n}else bn||dh(u,t.type,t.stateNode);else t.stateNode=s_(u,a,t.memoizedProps);else l!==a?(l===null?(n=s.stateNode,n===null||ke||n.parentNode.removeChild(n)):l.count--,a===null?bn||dh(u,t.type,t.stateNode):s_(u,a,t.memoizedProps)):a===null&&t.stateNode!==null&&vd(t,t.memoizedProps,s.memoizedProps);break;case 27:kn(n,t,a),Xn(t),l&512&&(ke||s===null||Dn(s,s.return)),s!==null&&l&4&&vd(t,t.memoizedProps,s.memoizedProps);break;case 5:if(u=$i,$i=!1,kn(n,t,a),$i=u,Xn(t),l&512&&(ke||s===null||Dn(s,s.return)),t.flags&32){n=t.stateNode;try{lr(n,""),Se=!0}catch(ht){We(t,t.return,ht)}}l&4&&t.stateNode!=null&&(n=t.memoizedProps,vd(t,n,s!==null?s.memoizedProps:n)),l&1024&&(Td=!0);break;case 6:if(kn(n,t,a),Xn(t),l&4){if(t.stateNode===null)throw Error(r(162));n=t.memoizedProps,a=t.stateNode;try{a.nodeValue=n,Se=!0}catch(ht){We(t,t.return,ht)}}break;case 3:if(Se=!1,Qc=null,u=Ni,Ni=ol(n.containerInfo),kn(n,t,a),Ni=u,Xn(t),l&4&&s!==null&&s.memoizedState.isDehydrated)try{Wr(n.containerInfo)}catch(ht){We(t,t.return,ht)}Td&&(Td=!1,tv(t)),Se=!1;break;case 4:l=$i,$i=bn,s=He(),u=Ni,Ni=ol(t.stateNode.containerInfo),kn(n,t,a),Xn(t),Ni=u,Se&&Jo&&(Oc=!0),Se=s,$i=l;break;case 12:kn(n,t,a),Xn(t);break;case 31:kn(n,t,a),Xn(t),l&4&&(n=t.updateQueue,n!==null&&(t.updateQueue=null,Ic(t,n)));break;case 13:kn(n,t,a),Xn(t),t.child.flags&8192&&t.memoizedState!==null!=(s!==null&&s.memoizedState!==null)&&(Fc=kt()),l&4&&(n=t.updateQueue,n!==null&&(t.updateQueue=null,Ic(t,n)));break;case 22:u=t.memoizedState!==null,g=s!==null&&s.memoizedState!==null;var A=bn,z=ke,it=$i;bn=A||u,$i=it||u,ke=z||g,kn(n,t,a),ke=z,$i=it,bn=A,Xn(t),l&8192&&(n=t.stateNode,n._visibility=u?n._visibility&-2:n._visibility|1,!u||s===null||g||bn||ke||(n=g||ke,a=bn,s=ke,bn=u||bn,ke=n,ns(t,2),bn=a,ke=s),!u&&$i||Rd(t,u)),l&4&&(n=t.updateQueue,n!==null&&(a=n.retryQueue,a!==null&&(n.retryQueue=null,Ic(t,a))));break;case 19:kn(n,t,a),Xn(t),l&4&&(n=t.updateQueue,n!==null&&(t.updateQueue=null,Ic(t,n)));break;case 30:l&512&&(ke||s===null||Dn(s,s.return)),l=He(),u=Jo,g=(a&335544064)===a,A=t.memoizedProps,Jo=g&&_a(A.default,A.update)!=="none",kn(n,t,a),Xn(t),g&&s!==null&&Se&&(t.flags|=4),Jo=u,Se=l;break;case 21:break;case 7:l&512&&(ke||s===null||Dn(s,s.return)),s&&s.stateNode!==null&&(s.stateNode._fragmentFiber=t);default:kn(n,t,a),Xn(t)}}function Xn(t){var n=t.flags;if(n&2){try{for(var a,s=t.return;s!==null;){if(z0(s)){a=s;break}s=s.return}s=null;for(var l=t.return;l!==null;){if(md(l)){var u=l.stateNode;s===null?s=[u]:s.push(u)}if(pd(l))break;l=l.return}var g=s;if(a==null)throw Error(r(160));switch(a.tag){case 27:var A=a.stateNode,z=_d(t);Nc(t,z,A,g);break;case 5:var it=a.stateNode;a.flags&32&&(lr(it,""),a.flags&=-33);var ht=_d(t);Nc(t,ht,it,g);break;case 3:case 4:var Et=a.stateNode.containerInfo,tt=_d(t);xd(t,tt,Et,g);break;default:throw Error(r(161))}}catch(ft){We(t,t.return,ft)}t.flags&=-3}n&4096&&(t.flags&=-4097)}function tv(t){if(t.subtreeFlags&1024)for(t=t.child;t!==null;){var n=t;tv(n),n.tag===5&&n.flags&1024&&(n=n.stateNode,Xr=!0,n.reset(),Xr=!1),t=t.sibling}}function Rr(t,n){if(n.subtreeFlags&9270)for(n=n.child;n!==null;)ev(n,t),n=n.sibling;else W0(n)}function ev(t,n){var a=t.alternate;if(a===null)Sd(t,!1);else switch(t.tag){case 3:if(Ad=ta=!1,G0(),Rr(n,t),!ta&&!Oc){if(t=Qi,t!==null)for(var s=0;s<t.length;s+=3){a=t[s];var l=t[s+1];Vv(a,t[s+2]),a=a.ownerDocument.documentElement,a!==null&&a.animate({opacity:[0,0],pointerEvents:["none","none"]},{duration:0,fill:"forwards",pseudoElement:"::view-transition-group("+l+")"})}t=n.containerInfo,t=t.nodeType===9?t.documentElement:t.ownerDocument.documentElement,t!==null&&t.style.viewTransitionName===""&&(t.style.viewTransitionName="none",t.animate({opacity:[0,0],pointerEvents:["none","none"]},{duration:0,fill:"forwards",pseudoElement:"::view-transition-group(root)"}),t.animate({width:[0,0],height:[0,0]},{duration:0,fill:"forwards",pseudoElement:"::view-transition"})),Ad=!0}Qi=null;break;case 5:Rr(n,t);break;case 4:s=ta,ta=!1,Rr(n,t),ta&&(Oc=!0),ta=s;break;case 22:t.memoizedState===null&&(a.memoizedState!==null?Sd(t,!1):Rr(n,t));break;case 30:s=ta,l=G0(),ta=!1,Rr(n,t),ta&&(t.flags|=4);var u=t.memoizedProps,g=t.stateNode;n=va(u,g),g=va(a.memoizedProps,g);var A=_a(u.default,u.update);A==="none"?n=!1:(u=a.memoizedState,a.memoizedState=null,a=t.child,Zn=0,n=bd(t,a,n,g,A,u,!0),Zn!==(u===null?0:u.length)&&(t.flags|=32)),(t.flags&4)!==0&&n?(Or(t,t.memoizedProps.onUpdate),Qi=l):l!==null&&(l.push.apply(l,Qi),Qi=l),ta=(t.flags&32)!==0?!0:s;break;default:Rr(n,t)}}function ea(t,n){if(n.subtreeFlags&8772)for(n=n.child;n!==null;)Y0(t,n.alternate,n),n=n.sibling}function ns(t,n){for(t=t.child;t!==null;){var a=t,s=n;switch(a.tag){case 0:case 11:case 14:case 15:es(4,a,a.return),ns(a,s);break;case 1:Dn(a,a.return);var l=a.stateNode;typeof l.componentWillUnmount=="function"&&I0(a,a.return,l),ns(a,s);break;case 27:(s&2)!==0&&t_(a.stateNode,a.type,a.memoizedProps);case 5:Dn(a,a.return),a.tag!==5&&a.tag!==27||Qo(a),ns(a,s);break;case 6:Qo(a);break;case 26:Dn(a,a.return),l=a.stateNode,a.memoizedState!==null||l===null||ke||l.parentNode.removeChild(l),ns(a,s);break;case 22:a.memoizedState===null&&ns(a,s);break;case 30:Dn(a,a.return),ns(a,s);break;case 7:Dn(a,a.return);default:ns(a,s)}t=t.sibling}}function Di(t,n,a){for(a=(n.subtreeFlags&8772)!==0?a:a&-2,n=n.child;n!==null;){var s=n.alternate,l=t,u=n,g=u.flags,A=(a&1)!==0;switch(u.tag){case 0:case 11:case 15:Di(l,u,a),Ko(4,u);break;case 1:if(Di(l,u,a),s=u,l=s.stateNode,typeof l.componentDidMount=="function")try{l.componentDidMount()}catch(ht){We(s,s.return,ht)}if(s=u,l=s.updateQueue,l!==null){var z=s.stateNode;try{var it=l.shared.hiddenCallbacks;if(it!==null)for(l.shared.hiddenCallbacks=null,l=0;l<it.length;l++)Eg(it[l],z)}catch(ht){We(s,s.return,ht)}}A&&g&64&&P0(u),Ki(u,u.return);break;case 27:(a&2)!==0&&F0(u);case 5:u.tag!==5&&u.tag!==27||B0(u),Di(l,u,a),A&&s===null&&g&4&&gd(u),Ki(u,u.return);break;case 6:B0(u);break;case 26:z=u.stateNode,u.memoizedState!==null||z===null||bn||dh(ol(z.ownerDocument),u.type,z),Di(l,u,a),A&&s===null&&g&4&&gd(u),Ki(u,u.return);break;case 12:Di(l,u,a);break;case 31:Di(l,u,a),A&&g&4&&Q0(l,u);break;case 13:Di(l,u,a),A&&g&4&&J0(l,u);break;case 22:u.memoizedState===null&&Di(l,u,a),Ki(u,u.return);break;case 30:Di(l,u,a),Ki(u,u.return);break;case 7:Ki(u,u.return);default:Di(l,u,a)}n=n.sibling}}function Cd(t,n){var a=null;t!==null&&t.memoizedState!==null&&t.memoizedState.cachePool!==null&&(a=t.memoizedState.cachePool.pool),t=null,n.memoizedState!==null&&n.memoizedState.cachePool!==null&&(t=n.memoizedState.cachePool.pool),t!==a&&(t!=null&&t.refCount++,a!=null&&Bo(a))}function Nd(t,n){t=null,n.alternate!==null&&(t=n.alternate.memoizedState.cache),n=n.memoizedState.cache,n!==t&&(n.refCount++,t!=null&&Bo(t))}function Ei(t,n,a,s){var l=(a&335544064)===a;if(n.subtreeFlags&(l?10262:10256))for(n=n.child;n!==null;)nv(t,n,a,s),n=n.sibling;else l&&X0(n)}function nv(t,n,a,s){var l=(a&335544064)===a;l&&n.alternate===null&&n.return!==null&&n.return.alternate!==null&&Lc(n);var u=n.flags;switch(n.tag){case 0:case 11:case 15:Ei(t,n,a,s),u&2048&&Ko(9,n);break;case 1:Ei(t,n,a,s);break;case 3:Ei(t,n,a,s),l&&Ad&&(t=t.containerInfo,t=t.nodeType===9?t.body:t.nodeName==="HTML"?t.ownerDocument.body:t,t.style.viewTransitionName==="root"&&(t.style.viewTransitionName=""),t=t.ownerDocument.documentElement,t!==null&&t.style.viewTransitionName==="none"&&(t.style.viewTransitionName="")),u&2048&&(u=null,n.alternate!==null&&(u=n.alternate.memoizedState.cache),n=n.memoizedState.cache,n!==u&&(n.refCount++,u!=null&&Bo(u)));break;case 12:if(u&2048){Ei(t,n,a,s),u=n.stateNode;try{var g=n.memoizedProps,A=g.id,z=g.onPostCommit;typeof z=="function"&&z(A,n.alternate===null?"mount":"update",u.passiveEffectDuration,-0)}catch(it){We(n,n.return,it)}}else Ei(t,n,a,s);break;case 31:Ei(t,n,a,s);break;case 13:Ei(t,n,a,s);break;case 23:break;case 22:g=n.stateNode,A=n.alternate,n.memoizedState!==null?(l&&A!==null&&A.memoizedState===null&&Lc(A),g._visibility&2?Ei(t,n,a,s):$o(t,n)):(l&&A!==null&&A.memoizedState!==null&&Lc(n),g._visibility&2?Ei(t,n,a,s):(g._visibility|=2,wr(t,n,a,s,(n.subtreeFlags&10256)!==0||!1))),u&2048&&Cd(A,n);break;case 24:Ei(t,n,a,s),u&2048&&Nd(n.alternate,n);break;case 30:l&&(u=n.alternate,u!==null&&(Ji(u.child,!0),Ji(n.child,!0))),Ei(t,n,a,s);break;default:Ei(t,n,a,s)}}function wr(t,n,a,s,l){for(l=l&&((n.subtreeFlags&10256)!==0||!1),n=n.child;n!==null;){var u=t,g=n,A=a,z=s,it=g.flags;switch(g.tag){case 0:case 11:case 15:wr(u,g,A,z,l),Ko(8,g);break;case 23:break;case 22:var ht=g.stateNode;g.memoizedState!==null?ht._visibility&2?wr(u,g,A,z,l):$o(u,g):(ht._visibility|=2,wr(u,g,A,z,l)),l&&it&2048&&Cd(g.alternate,g);break;case 24:wr(u,g,A,z,l),l&&it&2048&&Nd(g.alternate,g);break;default:wr(u,g,A,z,l)}n=n.sibling}}function $o(t,n){if(n.subtreeFlags&10256)for(n=n.child;n!==null;){var a=t,s=n,l=s.flags;switch(s.tag){case 22:$o(a,s),l&2048&&Cd(s.alternate,s);break;case 24:$o(a,s),l&2048&&Nd(s.alternate,s);break;default:$o(a,s)}n=n.sibling}}var Gs=8192;function Vs(t,n,a){if(t.subtreeFlags&Gs)for(t=t.child;t!==null;)iv(t,n,a),t=t.sibling}function iv(t,n,a){switch(t.tag){case 26:Vs(t,n,a),t.flags&Gs&&(t.memoizedState!==null?qM(a,Ni,t.memoizedState,t.memoizedProps):(t=t.stateNode,(n&335544128)===n&&u_(a,t)));break;case 5:Vs(t,n,a),t.flags&Gs&&(t=t.stateNode,(n&335544128)===n&&u_(a,t));break;case 3:case 4:var s=Ni;Ni=ol(t.stateNode.containerInfo),Vs(t,n,a),Ni=s;break;case 22:t.memoizedState===null&&(s=t.alternate,s!==null&&s.memoizedState!==null?(s=Gs,Gs=16777216,Vs(t,n,a),Gs=s):Vs(t,n,a));break;case 30:if((t.flags&Gs)!==0&&(s=t.memoizedProps.name,s!=null&&s!=="auto")){var l=t.stateNode;l.paired=null,oi===null&&(oi=new Map),oi.set(s,l)}Vs(t,n,a);break;default:Vs(t,n,a)}}function av(t){var n=t.alternate;if(n!==null&&(t=n.child,t!==null)){n.child=null;do n=t.sibling,t.sibling=null,t=n;while(t!==null)}}function tl(t){var n=t.deletions;if((t.flags&16)!==0){if(n!==null)for(var a=0;a<n.length;a++){var s=n[a];Tn=s,rv(s,t)}av(t)}if(t.subtreeFlags&10256)for(t=t.child;t!==null;)sv(t),t=t.sibling}function sv(t){switch(t.tag){case 0:case 11:case 15:tl(t),t.flags&2048&&es(9,t,t.return);break;case 3:tl(t);break;case 12:tl(t);break;case 22:var n=t.stateNode;t.memoizedState!==null&&n._visibility&2&&(t.return===null||t.return.tag!==13)?(n._visibility&=-3,Bc(t)):tl(t);break;default:tl(t)}}function Bc(t){var n=t.deletions;if((t.flags&16)!==0){if(n!==null)for(var a=0;a<n.length;a++){var s=n[a];Tn=s,rv(s,t)}av(t)}for(t=t.child;t!==null;){switch(n=t,n.tag){case 0:case 11:case 15:es(8,n,n.return),Bc(n);break;case 22:a=n.stateNode,a._visibility&2&&(a._visibility&=-3,Bc(n));break;default:Bc(n)}t=t.sibling}}function rv(t,n){for(;Tn!==null;){var a=Tn;switch(a.tag){case 0:case 11:case 15:es(8,a,n);break;case 23:case 22:if(a.memoizedState!==null&&a.memoizedState.cachePool!==null){var s=a.memoizedState.cachePool.pool;s!=null&&s.refCount++}break;case 24:Bo(a.memoizedState.cache)}if(s=a.child,s!==null)s.return=a,Tn=s;else t:for(a=t;Tn!==null;){s=Tn;var l=s.sibling,u=s.return;if(Z0(s),s===a){Tn=null;break t}if(l!==null){l.return=u,Tn=l;break t}Tn=u}}}var Gy={getCacheForType:function(t){var n=wn(hn),a=n.data.get(t);return a===void 0&&(a=t(),n.data.set(t,a)),a},cacheSignal:function(){return wn(hn).controller.signal}},Vy=typeof WeakMap=="function"?WeakMap:Map,Ge=0,Qe=null,be=null,Ce=0,Xe=0,li=null,is=!1,Cr=!1,Dd=!1,Ra=0,fn=0,as=0,ks=0,zc=0,ci=0,Nr=0,el=null,Qn=null,Ud=!1,Fc=0,ov=0,Hc=1/0,Gc=null,ss=null,rn=0,Ui=null,Xs=null,na=0,Ld=0,Od=null,lv=null,Dr=null,Ur=null,Lr=null,nl=0,Vc=null;function ui(){return(Ge&2)!==0&&Ce!==0?Ce&-Ce:xt.T!==null?Xd():Hl()}function cv(){if(ci===0)if((Ce&536870912)===0||ye){var t=Es;Es<<=1,(Es&3932160)===0&&(Es=262144),ci=t}else ci=536870912;return t=Cn.current,t!==null&&(t.flags|=32),ci}function Or(t,n){if(n!=null){var a=t.stateNode,s=a.ref;s===null&&(s=a.ref=kv(va(t.memoizedProps,a))),Ur===null&&(Ur=[]),Ur.push(n.bind(null,s))}}function Jn(t,n,a){(t===Qe&&(Xe===2||Xe===9)||t.cancelPendingCommit!==null)&&(Pr(t,0),rs(t,Ce,ci,!1)),Wi(t,a),((Ge&2)===0||t!==Qe)&&(t===Qe&&((Ge&2)===0&&(ks|=a),fn===4&&rs(t,Ce,ci,!1)),ia(t))}function uv(t,n,a){if((Ge&6)!==0)throw Error(r(327));var s=!a&&(n&127)===0&&(n&t.expiredLanes)===0||Ga(t,n),l=s?Wy(t,n):Id(t,n,!0),u=s;do{if(l===0){Cr&&!s&&rs(t,n,0,!1);break}else{if(a=t.current.alternate,u&&!ky(a)){l=Id(t,n,!1),u=!1;continue}if(l===2){if(u=n,t.errorRecoveryDisabledLanes&u)var g=0;else g=t.pendingLanes&-536870913,g=g!==0?g:g&536870912?536870912:0;if(g!==0){n=g;t:{var A=t;l=el;var z=A.current.memoizedState.isDehydrated;if(z&&(Pr(A,g).flags|=256),g=Id(A,g,!1),g!==2&&g!==6){if(Dd&&!z){A.errorRecoveryDisabledLanes|=u,ks|=u,l=4;break t}u=Qn,Qn=l,u!==null&&(Qn===null?Qn=u:Qn.push.apply(Qn,u))}l=g}if(u=!1,l!==2)continue}}if(l===1){Pr(t,0),rs(t,n,0,!0);break}t:{switch(s=t,u=l,u){case 0:case 1:throw Error(r(345));case 4:if((n&4194048)!==n&&(n&62914560)!==n)break;case 6:rs(s,n,ci,!is);break t;case 2:Qn=null;break;case 3:case 5:break;default:throw Error(r(329))}if((n&62914560)===n&&(l=Fc+300-kt(),10<l)){if(rs(s,n,ci,!is),bs(s,0,!0)!==0)break t;na=n,s.timeoutHandle=eh(fv.bind(null,s,a,Qn,Gc,Ud,n,ci,ks,Nr,is,u,"Throttled",-0,0),l);break t}fv(s,a,Qn,Gc,Ud,n,ci,ks,Nr,is,u,null,-0,0)}}break}while(!0);ia(t)}function fv(t,n,a,s,l,u,g,A,z,it,ht,Et,tt,ft){t.timeoutHandle=-1;var Pt=n.subtreeFlags,Qt=(u&335544064)===u;if(Et=null,(Qt||Pt&8192||(Pt&16785408)===16785408)&&(Et={stylesheets:null,count:0,imgCount:0,imgBytes:0,suspenseyImages:[],waitingForImages:!0,waitingForViewTransition:!1,unsuspend:Yi},oi=null,iv(n,u,Et),Qt&&(Pt=Et,Qt=t.containerInfo,Qt=(Qt.nodeType===9?Qt:Qt.ownerDocument).__reactViewTransition,Qt!=null&&(Pt.count++,Pt.waitingForViewTransition=!0,Pt=ul.bind(Pt),Qt.finished.then(Pt,Pt))),Pt=(u&62914560)===u?Fc-kt():(u&4194048)===u?ov-kt():0,Pt=YM(Et,Pt),Pt!==null)){na=u,t.cancelPendingCommit=Pt(xv.bind(null,t,n,u,a,s,l,g,A,z,it,ht,Et,null,tt,ft)),rs(t,u,g,!it);return}xv(t,n,u,a,s,l,g,A,z,it,ht,Et)}function ky(t){for(var n=t;;){var a=n.tag;if((a===0||a===11||a===15)&&n.flags&16384&&(a=n.updateQueue,a!==null&&(a=a.stores,a!==null)))for(var s=0;s<a.length;s++){var l=a[s],u=l.getSnapshot;l=l.value;try{if(!si(u(),l))return!1}catch{return!1}}if(a=n.child,n.subtreeFlags&16384&&a!==null)a.return=n,n=a;else{if(n===t)break;for(;n.sibling===null;){if(n.return===null||n.return===t)return!0;n=n.return}n.sibling.return=n.return,n=n.sibling}}return!0}function rs(t,n,a,s){n=Xi(t,n),n&=~zc,n&=~ks,t.suspendedLanes|=n,t.pingedLanes&=~n,s&&(t.warmLanes|=n),s=t.expirationTimes;for(var l=n;0<l;){var u=31-ue(l),g=1<<u;s[u]=-1,l&=~g}a!==0&&Ts(t,a,n)}function kc(){return(Ge&6)===0?(il(0),!1):!0}function Pd(){if(be!==null){if(Xe===0)var t=be.return;else t=be,ya=Ds=null,kf(t),yr=null,Ho=0,t=be;for(;t!==null;)O0(t.alternate,t),t=t.return;be=null}}function Pr(t,n){var a=t.timeoutHandle;return a!==-1&&(t.timeoutHandle=-1,hM(a)),a=t.cancelPendingCommit,a!==null&&(t.cancelPendingCommit=null,a()),na=0,Pd(),Qe=t,be=a=xa(t.current,null),Ce=n,Xe=0,li=null,is=!1,Cr=Ga(t,n),Dd=!1,Nr=ci=zc=ks=as=fn=0,Qn=el=null,Ud=!1,Ra=Xi(t,n),Ql(),a}function dv(t,n){me=null,xt.H=Mc,n===Sr||n===lc?(n=xg(),Xe=3):n===Nf?(n=xg(),Xe=4):Xe=n===ad?8:n!==null&&typeof n=="object"&&typeof n.then=="function"?6:1,li=n,be===null&&(fn=1,Ec(t,xi(n,t.current)))}function hv(){var t=Cn.current;return t===null?!0:(Ce&4194048)===Ce?Pn===null:(Ce&62914560)===Ce||(Ce&536870912)!==0?t===Pn:!1}function pv(){var t=xt.H;return xt.H=Mc,t===null?Mc:t}function mv(){var t=xt.A;return xt.A=Gy,t}function Xc(){fn=4,is||(Ce&4194048)!==Ce&&Cn.current!==null||(Cr=!0),(as&134217727)===0&&(ks&134217727)===0||Qe===null||rs(Qe,Ce,ci,!1)}function Id(t,n,a){var s=Ge;Ge|=2;var l=pv(),u=mv();(Qe!==t||Ce!==n)&&(Gc=null,Pr(t,n)),n=!1;var g=fn;t:do try{if(Xe!==0&&be!==null){var A=be,z=li;switch(Xe){case 8:Pd(),g=6;break t;case 3:case 2:case 9:case 6:Cn.current===null&&(n=!0);var it=Xe;if(Xe=0,li=null,Ir(t,A,z,it),a&&Cr){g=0;break t}break;default:it=Xe,Xe=0,li=null,Ir(t,A,z,it)}}Xy(),g=fn;break}catch(ht){dv(t,ht)}while(!0);return n&&t.shellSuspendCounter++,ya=Ds=null,Ge=s,xt.H=l,xt.A=u,be===null&&(Qe=null,Ce=0,Ql()),g}function Xy(){for(;be!==null;)gv(be)}function Wy(t,n){var a=Ge;Ge|=2;var s=pv(),l=mv();Qe!==t||Ce!==n?(Gc=null,Hc=kt()+500,Pr(t,n)):Cr=Ga(t,n);t:do try{if(Xe!==0&&be!==null){n=be;var u=li;e:switch(Xe){case 1:Xe=0,li=null,Ir(t,n,u,1);break;case 2:case 9:if(vg(u)){Xe=0,li=null,vv(n);break}n=function(){Xe!==2&&Xe!==9||Qe!==t||(Xe=7),ia(t)},u.then(n,n);break t;case 3:Xe=7;break t;case 4:Xe=5;break t;case 7:vg(u)?(Xe=0,li=null,vv(n)):(Xe=0,li=null,Ir(t,n,u,7));break;case 5:var g=null;switch(be.tag){case 26:g=be.memoizedState;case 5:case 27:var A=be;if(g?l_(g):A.stateNode.complete){Xe=0,li=null;var z=A.sibling;if(z!==null)be=z;else{var it=A.return;it!==null?(be=it,Wc(it)):be=null}break e}}Xe=0,li=null,Ir(t,n,u,5);break;case 6:Xe=0,li=null,Ir(t,n,u,6);break;case 8:Pd(),fn=6;break t;default:throw Error(r(462))}}qy();break}catch(ht){dv(t,ht)}while(!0);return ya=Ds=null,xt.H=s,xt.A=l,Ge=a,be!==null?0:(Qe=null,Ce=0,Ql(),fn)}function qy(){for(;be!==null&&!Bt();)gv(be)}function gv(t){var n=U0(t.alternate,t,Ra);t.memoizedProps=t.pendingProps,n===null?Wc(t):be=n}function vv(t){var n=t,a=n.alternate;switch(n.tag){case 15:case 0:n=T0(a,n,n.pendingProps,n.type,void 0,Ce);break;case 11:n=T0(a,n,n.pendingProps,n.type.render,n.ref,Ce);break;case 5:kf(n);var s=n;s===En&&(ye?(ic(s),s.tag===5&&s.stateNode!=null&&(tn=s.stateNode)):(ic(s),ye=!0));default:O0(a,n),n=be=rg(n,Ra),n=U0(a,n,Ra)}t.memoizedProps=t.pendingProps,n===null?Wc(t):be=n}function Ir(t,n,a,s){ya=Ds=null,kf(n),yr=null,Ho=0;var l=n.return;try{if(Ly(t,l,n,a,Ce)){fn=1,Ec(t,xi(a,t.current)),be=null;return}}catch(u){if(l!==null)throw be=l,u;fn=1,Ec(t,xi(a,t.current)),be=null;return}n.flags&32768?(ye||s===1?t=!0:Cr||(Ce&536870912)!==0?t=!1:(is=t=!0,(s===2||s===9||s===3||s===6)&&(s=Cn.current,s!==null&&s.tag===13&&(s.flags|=16384))),_v(n,t)):Wc(n)}function Wc(t){var n=t;do{if((n.flags&32768)!==0){_v(n,is);return}t=n.return;var a=By(n.alternate,n,Ra);if(a!==null){be=a;return}if(n=n.sibling,n!==null){be=n;return}be=n=t}while(n!==null);fn===0&&(fn=5)}function _v(t,n){do{var a=zy(t.alternate,t);if(a!==null){a.flags&=32767,be=a;return}if(a=t.return,a!==null&&(a.flags|=32768,a.subtreeFlags=0,a.deletions=null),!n&&(t=t.sibling,t!==null)){be=t;return}be=t=a}while(t!==null);fn=6,be=null}function xv(t,n,a,s,l,u,g,A,z,it,ht,Et){t.cancelPendingCommit=null;do qc();while(rn!==0);if((Ge&6)!==0)throw Error(r(327));if(n!==null){if(n===t.current)throw Error(r(177));t===Qe&&(be=Qe=null,Ce=0),Xs=n,Ui=t,na=a,Od=l,lv=s,Yy(t,n,a,g,A,z,Et)}}function Yy(t,n,a,s,l,u,g){var A=n.lanes|n.childLanes;if(Ld=A,A|=vf,Fl(t,a,A,s,l,u),Ur=null,(a&335544064)===a?(Lr=yy(t),s=10262):(Lr=null,s=10256),(n.subtreeFlags&s)!==0||(n.flags&s)!==0?(t.callbackNode=null,t.callbackPriority=0,$y(Ut,function(){return Hd(),null})):(t.callbackNode=null,t.callbackPriority=0),Dc=!1,s=(n.flags&13878)!==0,(n.subtreeFlags&13878)!==0||s){s=xt.T,xt.T=null,l=Ct.p,Ct.p=2,u=Ge,Ge|=4;try{Fy(t,n,a)}finally{Ge=u,Ct.p=l,xt.T=s}}rn=1,Dc?Dr=xM(g,t.containerInfo,Lr,Bd,zd,Zy,Fd,Hd,jy):(Bd(),zd(),Fd())}function jy(t){if(rn!==0){var n=Ui.onRecoverableError;n(t,{componentStack:null})}}function Zy(){rn===3&&(rn=0,ev(Xs,Ui),rn=4)}function Bd(){if(rn===1){rn=0;var t=Ui,n=Xs,a=na,s=(n.flags&13878)!==0;if((n.subtreeFlags&13878)!==0||s){s=xt.T,xt.T=null;var l=Ct.p;Ct.p=2;var u=Ge;Ge|=4;try{Jo=Oc=!1,$0(n,t,a),a=Jd;var g=Km(t.containerInfo),A=a.focusedElem,z=a.selectionRange;if(g!==A&&A&&A.ownerDocument&&Zm(A.ownerDocument.documentElement,A)){if(z!==null&&df(A)){var it=z.start,ht=z.end;if(ht===void 0&&(ht=it),"selectionStart"in A)A.selectionStart=it,A.selectionEnd=Math.min(ht,A.value.length);else{var Et=A.ownerDocument||document,tt=Et&&Et.defaultView||window;if(tt.getSelection){var ft=tt.getSelection(),Pt=A.textContent.length,Qt=Math.min(z.start,Pt),ge=z.end===void 0?Qt:Math.min(z.end,Pt);!ft.extend&&Qt>ge&&(g=ge,ge=Qt,Qt=g);var nt=jm(A,Qt),W=jm(A,ge);if(nt&&W&&(ft.rangeCount!==1||ft.anchorNode!==nt.node||ft.anchorOffset!==nt.offset||ft.focusNode!==W.node||ft.focusOffset!==W.offset)){var rt=Et.createRange();rt.setStart(nt.node,nt.offset),ft.removeAllRanges(),Qt>ge?(ft.addRange(rt),ft.extend(W.node,W.offset)):(rt.setEnd(W.node,W.offset),ft.addRange(rt))}}}}for(Et=[],ft=A;ft=ft.parentNode;)ft.nodeType===1&&Et.push({element:ft,left:ft.scrollLeft,top:ft.scrollTop});for(typeof A.focus=="function"&&A.focus(),A=0;A<Et.length;A++){var Mt=Et[A];Mt.element.scrollLeft=Mt.left,Mt.element.scrollTop=Mt.top}}Xr=!!Qd,Jd=Qd=null}finally{Ge=u,Ct.p=l,xt.T=s}}t.current=n,rn=2}}function zd(){if(rn===2){rn=0;var t=Ui,n=Xs,a=(n.flags&8772)!==0;if((n.subtreeFlags&8772)!==0||a){a=xt.T,xt.T=null;var s=Ct.p;Ct.p=2;var l=Ge;Ge|=4;try{Y0(t,n.alternate,n)}finally{Ge=l,Ct.p=s,xt.T=a}}rn=3}}function Fd(){if(rn===4||rn===3){rn=0;var t=Dr;Dr=null,It();var n=Ui,a=Xs,s=na,l=lv,u=(s&335544064)===s?10262:10256;if((a.subtreeFlags&u)!==0||(a.flags&u)!==0?rn=5:(rn=0,Xs=Ui=null,Sv(n,n.pendingLanes)),u=n.pendingLanes,u===0&&(ss=null),To(s),a=a.stateNode,Gt&&typeof Gt.onCommitFiberRoot=="function")try{Gt.onCommitFiberRoot($t,a,void 0,(a.current.flags&128)===128)}catch{}if(l!==null){a=xt.T,u=Ct.p,Ct.p=2,xt.T=null;try{for(var g=n.onRecoverableError,A=0;A<l.length;A++){var z=l[A];g(z.value,{componentStack:z.stack})}}finally{xt.T=a,Ct.p=u}}if(l=Ur,g=Lr,Lr=null,l!==null&&(Ur=null,g===null&&(g=[]),t!==null))for(z=0;z<l.length;z++)a=(0,l[z])(g),a!==void 0&&t.finished.finally(a);(na&3)!==0&&qc(),ia(n),u=n.pendingLanes,(s&261930)!==0&&(u&42)!==0?n===Vc?nl++:(nl=0,Vc=n):(nl=0,Vc=null),il(0)}}function Sv(t,n){(t.pooledCacheLanes&=n)===0&&(n=t.pooledCache,n!=null&&(t.pooledCache=null,Bo(n)))}function qc(){return Dr!==null&&(Dr.skipTransition(),Dr=null),Bd(),zd(),Fd(),Hd()}function Hd(){if(rn!==5)return!1;var t=Ui,n=Ld;Ld=0;var a=To(na),s=xt.T,l=Ct.p;try{Ct.p=32>a?32:a,xt.T=null,a=Od,Od=null;var u=Ui,g=na;if(rn=0,Xs=Ui=null,na=0,(Ge&6)!==0)throw Error(r(331));var A=Ge;if(Ge|=4,sv(u.current),nv(u,u.current,g,a),Ge=A,il(0,!1),Gt&&typeof Gt.onPostCommitFiberRoot=="function")try{Gt.onPostCommitFiberRoot($t,u)}catch{}return!0}finally{Ct.p=l,xt.T=s,Sv(t,n)}}function yv(t,n,a){n=xi(a,n),n=id(t.stateNode,n,2),t=Qa(t,n,2),t!==null&&(Wi(t,2),ia(t))}function We(t,n,a){if(t.tag===3)yv(t,t,a);else for(;n!==null;){if(n.tag===3){yv(n,t,a);break}else if(n.tag===1){var s=n.stateNode;if(typeof n.type.getDerivedStateFromError=="function"||typeof s.componentDidCatch=="function"&&(ss===null||!ss.has(s))){t=xi(a,t),a=v0(2),s=Qa(n,a,2),s!==null&&(_0(a,s,n,t),Wi(s,2),ia(s));break}}n=n.return}}function Gd(t,n,a){var s=t.pingCache;if(s===null){s=t.pingCache=new Vy;var l=new Set;s.set(n,l)}else l=s.get(n),l===void 0&&(l=new Set,s.set(n,l));l.has(a)||(Dd=!0,l.add(a),t=Ky.bind(null,t,n,a),n.then(t,t))}function Ky(t,n,a){var s=t.pingCache;s!==null&&s.delete(n),t.pingedLanes|=t.suspendedLanes&a,t.warmLanes&=~a,Qe===t&&(Ce&a)===a&&((fn===4||fn===3&&(Ce&62914560)===Ce&&300>kt()-Fc)&&(Ge&2)===0?Pr(t,0):zc|=a,Nr===Ce&&(Nr=0)),ia(t)}function Mv(t,n){n===0&&(n=yo()),t=ws(t,n),t!==null&&(Wi(t,n),ia(t))}function Qy(t){var n=t.memoizedState,a=0;n!==null&&(a=n.retryLane),Mv(t,a)}function Jy(t,n){var a=0;switch(t.tag){case 31:case 13:var s=t.stateNode,l=t.memoizedState;l!==null&&(a=l.retryLane);break;case 19:s=t.stateNode;break;case 22:s=t.stateNode._retryCache;break;default:throw Error(r(314))}s!==null&&s.delete(n),Mv(t,a)}function $y(t,n){return Nt(t,n)}var Br=null,zr=null,Vd=!1,Yc=!1,kd=!1,os=0;function ia(t){t!==zr&&t.next===null&&(zr===null?Br=zr=t:zr=zr.next=t),Yc=!0,Vd||(Vd=!0,eM())}function il(t,n){if(!kd&&Yc){kd=!0;do for(var a=!1,s=Br;s!==null;){if(t!==0){var l=s.pendingLanes;if(l===0)var u=0;else{var g=s.suspendedLanes,A=s.pingedLanes;u=(1<<31-ue(42|t)+1)-1,u&=l&~(g&~A),u=u&201326741?u&201326741|1:u?u|2:0}u!==0&&(a=!0,Av(s,u))}else u=Ce,u=bs(s,s===Qe?u:0,s.cancelPendingCommit!==null||s.timeoutHandle!==-1),(u&3)===0||Ga(s,u)||(a=!0,Av(s,u));s=s.next}while(a);kd=!1}}function tM(){Ev()}function Ev(){Yc=Vd=!1;var t=0;os!==0&&dM()&&(t=os);for(var n=kt(),a=null,s=Br;s!==null;){var l=s.next,u=bv(s,n);u===0?(s.next=null,a===null?Br=l:a.next=l,l===null&&(zr=a)):(a=s,(t!==0||(u&3)!==0)&&(Yc=!0)),s=l}rn!==0&&rn!==5||il(t),os!==0&&(os=0)}function bv(t,n){for(var a=t.suspendedLanes,s=t.pingedLanes,l=t.expirationTimes,u=t.pendingLanes&-62914561;0<u;){var g=31-ue(u),A=1<<g,z=l[g];z===-1?((A&a)===0||(A&s)!==0)&&(l[g]=So(A,n)):z<=n&&(t.expiredLanes|=A),u&=~A}if(n=Qe,a=Ce,a=bs(t,t===n?a:0,t.cancelPendingCommit!==null||t.timeoutHandle!==-1),s=t.callbackNode,a===0||t===n&&(Xe===2||Xe===9)||t.cancelPendingCommit!==null)return s!==null&&s!==null&&Jt(s),t.callbackNode=null,t.callbackPriority=0;if((a&3)===0||Ga(t,a)){if(n=a&-a,n===t.callbackPriority)return n;switch(s!==null&&Jt(s),To(a)){case 2:case 8:a=K;break;case 32:a=Ut;break;case 268435456:a=Lt;break;default:a=Ut}return s=Tv.bind(null,t),a=Nt(a,s),t.callbackPriority=n,t.callbackNode=a,n}return s!==null&&s!==null&&Jt(s),t.callbackPriority=2,t.callbackNode=null,2}function Tv(t,n){if(rn!==0&&rn!==5)return t.callbackNode=null,t.callbackPriority=0,null;var a=t.callbackNode;if(qc()&&t.callbackNode!==a)return null;var s=Ce;return s=bs(t,t===Qe?s:0,t.cancelPendingCommit!==null||t.timeoutHandle!==-1),s===0?null:(uv(t,s,n),bv(t,kt()),t.callbackNode!=null&&t.callbackNode===a?Tv.bind(null,t):null)}function Av(t,n){if(qc())return null;uv(t,n,!0)}function eM(){pM(function(){(Ge&6)!==0?Nt(ce,tM):Ev()})}function Xd(){if(os===0){var t=Os;t===0&&(t=sr,sr<<=1,(sr&261888)===0&&(sr=256)),os=t}return os}function Rv(t){return t==null||typeof t=="symbol"||typeof t=="boolean"?null:typeof t=="function"?t:kl(t)}function nM(t,n,a,s,l){if(n==="submit"&&a&&a.stateNode===l){var u=Rv((l[V]||null).action),g=s.submitter;g&&(n=(n=g[V]||null)?Rv(n.formAction):g.getAttribute("formAction"),n!==null&&(u=n,g=null));var A=new Yl("action","action",null,s,l);t.push({event:A,listeners:[{instance:null,listener:function(){if(s.defaultPrevented){if(os!==0){var z=new FormData(l,g);Jf(a,{pending:!0,data:z,method:l.method,action:u},null,z)}}else typeof u=="function"&&(A.preventDefault(),z=new FormData(l,g),Jf(a,{pending:!0,data:z,method:l.method,action:u},u,z))},currentTarget:l}]})}}for(var Wd=0;Wd<gf.length;Wd++){var qd=gf[Wd],iM=qd.toLowerCase(),aM=qd[0].toUpperCase()+qd.slice(1);wi(iM,"on"+aM)}wi($m,"onAnimationEnd"),wi(tg,"onAnimationIteration"),wi(eg,"onAnimationStart"),wi("dblclick","onDoubleClick"),wi("focusin","onFocus"),wi("focusout","onBlur"),wi(hy,"onTransitionRun"),wi(py,"onTransitionStart"),wi(my,"onTransitionCancel"),wi(ng,"onTransitionEnd"),ln("onMouseEnter",["mouseout","mouseover"]),ln("onMouseLeave",["mouseout","mouseover"]),ln("onPointerEnter",["pointerout","pointerover"]),ln("onPointerLeave",["pointerout","pointerover"]),Ft("onChange","change click focusin focusout input keydown keyup selectionchange".split(" ")),Ft("onSelect","focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange".split(" ")),Ft("onBeforeInput",["compositionend","keypress","textInput","paste"]),Ft("onCompositionEnd","compositionend focusout keydown keypress keyup mousedown".split(" ")),Ft("onCompositionStart","compositionstart focusout keydown keypress keyup mousedown".split(" ")),Ft("onCompositionUpdate","compositionupdate focusout keydown keypress keyup mousedown".split(" "));var al="abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting".split(" "),sM=new Set("beforetoggle cancel close invalid load scroll scrollend toggle".split(" ").concat(al));function wv(t,n){n=(n&4)!==0;for(var a=0;a<t.length;a++){var s=t[a],l=s.event;s=s.listeners;t:{var u=void 0;if(n)for(var g=s.length-1;0<=g;g--){var A=s[g],z=A.instance,it=A.currentTarget;if(A=A.listener,z!==u&&l.isPropagationStopped())break t;u=A,l.currentTarget=it;try{u(l)}catch(ht){Kl(ht)}l.currentTarget=null,u=z}else for(g=0;g<s.length;g++){if(A=s[g],z=A.instance,it=A.currentTarget,A=A.listener,z!==u&&l.isPropagationStopped())break t;u=A,l.currentTarget=it;try{u(l)}catch(ht){Kl(ht)}l.currentTarget=null,u=z}}}}function Te(t,n){var a=n[ct];a===void 0&&(a=n[ct]=new Set);var s=t+"__bubble";a.has(s)||(Cv(n,t,2,!1),a.add(s))}function Yd(t,n,a){var s=0;n&&(s|=4),Cv(a,t,s,n)}var jc="_reactListening"+Math.random().toString(36).slice(2);function jd(t){if(!t[jc]){t[jc]=!0,Ve.forEach(function(a){a!=="selectionchange"&&(sM.has(a)||Yd(a,!1,t),Yd(a,!0,t))});var n=t.nodeType===9?t:t.ownerDocument;n===null||n[jc]||(n[jc]=!0,Yd("selectionchange",!1,n))}}function Cv(t,n,a,s){switch(__(n)){case 2:var l=QM;break;case 8:l=JM;break;default:l=ph}a=l.bind(null,n,a,t),l=void 0,!ef||n!=="touchstart"&&n!=="touchmove"&&n!=="wheel"||(l=!0),s?l!==void 0?t.addEventListener(n,a,{capture:!0,passive:l}):t.addEventListener(n,a,!0):l!==void 0?t.addEventListener(n,a,{passive:l}):t.addEventListener(n,a,!1)}function Zd(t,n,a,s,l){var u=s;if((n&1)===0&&(n&2)===0&&s!==null)t:for(;;){if(s===null)return;var g=s.tag;if(g===3||g===4){var A=s.stateNode.containerInfo;if(A===l)break;if(g===4)for(g=s.return;g!==null;){var z=g.tag;if((z===3||z===4)&&g.stateNode.containerInfo===l)return;g=g.return}for(;A!==null;){if(g=se(A),g===null)return;if(z=g.tag,z===5||z===6||z===26||z===27){s=u=g;continue t}A=A.parentNode}}s=s.return}Cm(function(){var it=u,ht=$u(a),Et=[];t:{var tt=ig.get(t);if(tt!==void 0){var ft=Yl,Pt=t;switch(t){case"keypress":if(Wl(a)===0)break t;case"keydown":case"keyup":ft=kS;break;case"focusin":Pt="focus",ft=rf;break;case"focusout":Pt="blur",ft=rf;break;case"beforeblur":case"afterblur":ft=rf;break;case"click":if(a.button===2)break t;case"auxclick":case"dblclick":case"mousedown":case"mousemove":case"mouseup":case"mouseout":case"mouseover":case"contextmenu":ft=Um;break;case"drag":case"dragend":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"dragstart":case"drop":ft=DS;break;case"touchcancel":case"touchend":case"touchmove":case"touchstart":ft=jS;break;case $m:case tg:case eg:ft=OS;break;case ng:ft=KS;break;case"scroll":case"scrollend":ft=CS;break;case"wheel":ft=JS;break;case"copy":case"cut":case"paste":ft=IS;break;case"gotpointercapture":case"lostpointercapture":case"pointercancel":case"pointerdown":case"pointermove":case"pointerout":case"pointerover":case"pointerup":ft=Om;break;case"submit":ft=qS;break;case"toggle":case"beforetoggle":ft=ty}var Qt=(n&4)!==0,ge=!Qt&&(t==="scroll"||t==="scrollend"),nt=Qt?tt!==null?tt+"Capture":null:tt;Qt=[];for(var W=it,rt;W!==null;){var Mt=W;if(rt=Mt.stateNode,Mt=Mt.tag,Mt!==5&&Mt!==26&&Mt!==27||rt===null||nt===null||(Mt=Ao(W,nt),Mt!=null&&Qt.push(sl(W,Mt,rt))),ge)break;W=W.return}0<Qt.length&&(tt=new ft(tt,Pt,null,a,ht),Et.push({event:tt,listeners:Qt}))}}if((n&7)===0){t:{if(ft=t==="mouseover"||t==="pointerover",tt=t==="mouseout"||t==="pointerout",ft&&a!==Ju&&(Pt=a.relatedTarget||a.fromElement)&&(se(Pt)||Pt[mt]))break t;(tt||ft)&&(Pt=ht.window===ht?ht:(ft=ht.ownerDocument)?ft.defaultView||ft.parentWindow:window,tt?(ft=a.relatedTarget||a.toElement,tt=it,ft=ft?se(ft):null,ft!==null&&(ge=f(ft),Qt=ft.tag,ft!==ge||Qt!==5&&Qt!==27&&Qt!==6)&&(ft=null)):(tt=null,ft=it),tt!==ft&&(Qt=Um,Mt="onMouseLeave",nt="onMouseEnter",W="mouse",(t==="pointerout"||t==="pointerover")&&(Qt=Om,Mt="onPointerLeave",nt="onPointerEnter",W="pointer"),ge=tt==null?Pt:Wt(tt),rt=ft==null?Pt:Wt(ft),Pt=new Qt(Mt,W+"leave",tt,a,ht),Pt.target=ge,Pt.relatedTarget=rt,Mt=null,se(ht)===it&&(Qt=new Qt(nt,W+"enter",ft,a,ht),Qt.target=rt,Qt.relatedTarget=ge,Mt=Qt),ge=Mt,Qt=tt&&ft?U(tt,ft,rM):null,tt!==null&&Nv(Et,Pt,tt,Qt,!1),ft!==null&&ge!==null&&Nv(Et,ge,ft,Qt,!0)))}t:{if(tt=it?Wt(it):window,ft=tt.nodeName&&tt.nodeName.toLowerCase(),ft==="select"||ft==="input"&&tt.type==="file")var qt=Vm;else if(Hm(tt))if(km)qt=uy;else{qt=ly;var Ne=oy}else ft=tt.nodeName,!ft||ft.toLowerCase()!=="input"||tt.type!=="checkbox"&&tt.type!=="radio"?it&&Qu(it.elementType)&&(qt=Vm):qt=cy;if(qt&&(qt=qt(t,it))){Gm(Et,qt,a,ht);break t}Ne&&Ne(t,tt,it)}switch(Ne=it?Wt(it):window,t){case"focusin":(Hm(Ne)||Ne.contentEditable==="true")&&(dr=Ne,hf=it,Oo=null);break;case"focusout":Oo=hf=dr=null;break;case"mousedown":pf=!0;break;case"contextmenu":case"mouseup":case"dragend":pf=!1,Qm(Et,a,ht);break;case"selectionchange":if(dy)break;case"keydown":case"keyup":Qm(Et,a,ht)}var ee;if(lf)t:{switch(t){case"compositionstart":var ae="onCompositionStart";break t;case"compositionend":ae="onCompositionEnd";break t;case"compositionupdate":ae="onCompositionUpdate";break t}ae=void 0}else fr?zm(t,a)&&(ae="onCompositionEnd"):t==="keydown"&&a.keyCode===229&&(ae="onCompositionStart");ae&&(Pm&&a.locale!=="ko"&&(fr||ae!=="onCompositionStart"?ae==="onCompositionEnd"&&fr&&(ee=Nm()):(Va=ht,nf="value"in Va?Va.value:Va.textContent,fr=!0)),Ne=Zc(it,ae),0<Ne.length&&(ae=new Lm(ae,t,null,a,ht),Et.push({event:ae,listeners:Ne}),ee?ae.data=ee:(ee=Fm(a),ee!==null&&(ae.data=ee)))),(ee=ny?iy(t,a):ay(t,a))&&(ae=Zc(it,"onBeforeInput"),0<ae.length&&(Ne=new Lm("onBeforeInput","beforeinput",null,a,ht),Et.push({event:Ne,listeners:ae}),Ne.data=ee)),nM(Et,t,it,a,ht)}wv(Et,n)})}function sl(t,n,a){return{instance:t,listener:n,currentTarget:a}}function Zc(t,n){for(var a=n+"Capture",s=[];t!==null;){var l=t,u=l.stateNode;if(l=l.tag,l!==5&&l!==26&&l!==27||u===null||(l=Ao(t,a),l!=null&&s.unshift(sl(t,l,u)),l=Ao(t,n),l!=null&&s.push(sl(t,l,u))),t.tag===3)return s;t=t.return}return[]}function rM(t){if(t===null)return null;do t=t.return;while(t&&t.tag!==5&&t.tag!==27);return t||null}function Nv(t,n,a,s,l){for(var u=n._reactName,g=[];a!==null&&a!==s;){var A=a,z=A.alternate,it=A.stateNode;if(A=A.tag,z!==null&&z===s)break;A!==5&&A!==26&&A!==27||it===null||(z=it,l?(it=Ao(a,u),it!=null&&g.unshift(sl(a,it,z))):l||(it=Ao(a,u),it!=null&&g.push(sl(a,it,z)))),a=a.return}g.length!==0&&t.push({event:n,listeners:g})}var oM=/\r\n?/g,lM=/\u0000|\uFFFD/g;function Dv(t){return(typeof t=="string"?t:""+t).replace(oM,`
`).replace(lM,"")}function Uv(t,n){return n=Dv(n),Dv(t)===n}function qe(t,n,a,s,l,u){switch(a){case"children":if(typeof s=="string")n==="body"||n==="textarea"&&s===""||lr(t,s);else if(typeof s=="number"||typeof s=="bigint")n!=="body"&&lr(t,""+s);else return;break;case"className":ai(t,"class",s);break;case"tabIndex":ai(t,"tabindex",s);break;case"dir":case"role":case"viewBox":case"width":case"height":ai(t,a,s);break;case"style":Rm(t,s,u);return;case"data":if(n!=="object"){ai(t,"data",s);break}case"src":case"href":if(s===""&&(n!=="a"||a!=="href")){t.removeAttribute(a);break}if(s==null||typeof s=="function"||typeof s=="symbol"||typeof s=="boolean"){t.removeAttribute(a);break}s=kl(s),t.setAttribute(a,s);break;case"action":case"formAction":if(typeof s=="function"){t.setAttribute(a,"javascript:throw new Error('A React form was unexpectedly submitted. If you called form.submit() manually, consider using form.requestSubmit() instead. If you\\'re trying to use event.stopPropagation() in a submit event handler, consider also calling event.preventDefault().')");break}else typeof u=="function"&&(a==="formAction"?(n!=="input"&&qe(t,n,"name",l.name,l,null),qe(t,n,"formEncType",l.formEncType,l,null),qe(t,n,"formMethod",l.formMethod,l,null),qe(t,n,"formTarget",l.formTarget,l,null)):(qe(t,n,"encType",l.encType,l,null),qe(t,n,"method",l.method,l,null),qe(t,n,"target",l.target,l,null)));if(s==null||typeof s=="symbol"||typeof s=="boolean"){t.removeAttribute(a);break}s=kl(s),t.setAttribute(a,s);break;case"onClick":s!=null&&(t.onclick=Yi);return;case"onScroll":s!=null&&Te("scroll",t);return;case"onScrollEnd":s!=null&&Te("scrollend",t);return;case"dangerouslySetInnerHTML":if(s!=null){if(typeof s!="object"||!("__html"in s))throw Error(r(61));if(a=s.__html,a!=null){if(l.children!=null)throw Error(r(60));(u!=null?u.__html:void 0)!==a&&(t.innerHTML=a)}}break;case"multiple":t.multiple=s&&typeof s!="function"&&typeof s!="symbol";break;case"muted":t.muted=s&&typeof s!="function"&&typeof s!="symbol";break;case"suppressContentEditableWarning":case"suppressHydrationWarning":case"defaultValue":case"defaultChecked":case"innerHTML":case"ref":break;case"autoFocus":break;case"xlinkHref":if(s==null||typeof s=="function"||typeof s=="boolean"||typeof s=="symbol"){t.removeAttribute("xlink:href");break}a=kl(s),t.setAttributeNS("http://www.w3.org/1999/xlink","xlink:href",a);break;case"contentEditable":case"spellCheck":case"draggable":case"value":case"autoReverse":case"externalResourcesRequired":case"focusable":case"preserveAlpha":s!=null&&typeof s!="function"&&typeof s!="symbol"?t.setAttribute(a,s):t.removeAttribute(a);break;case"inert":case"allowFullScreen":case"async":case"autoPlay":case"controls":case"credentialless":case"default":case"defer":case"disabled":case"disablePictureInPicture":case"disableRemotePlayback":case"formNoValidate":case"hidden":case"loop":case"noModule":case"noValidate":case"open":case"playsInline":case"readOnly":case"required":case"reversed":case"scoped":case"seamless":case"itemScope":s&&typeof s!="function"&&typeof s!="symbol"?t.setAttribute(a,""):t.removeAttribute(a);break;case"capture":case"download":s===!0?t.setAttribute(a,""):s!==!1&&s!=null&&typeof s!="function"&&typeof s!="symbol"?t.setAttribute(a,s):t.removeAttribute(a);break;case"cols":case"rows":case"size":case"span":s!=null&&typeof s!="function"&&typeof s!="symbol"&&!isNaN(s)&&1<=s?t.setAttribute(a,s):t.removeAttribute(a);break;case"rowSpan":case"start":s==null||typeof s=="function"||typeof s=="symbol"||isNaN(s)?t.removeAttribute(a):t.setAttribute(a,s);break;case"popover":Te("beforetoggle",t),Te("toggle",t),$e(t,"popover",s);break;case"xlinkActuate":we(t,"http://www.w3.org/1999/xlink","xlink:actuate",s);break;case"xlinkArcrole":we(t,"http://www.w3.org/1999/xlink","xlink:arcrole",s);break;case"xlinkRole":we(t,"http://www.w3.org/1999/xlink","xlink:role",s);break;case"xlinkShow":we(t,"http://www.w3.org/1999/xlink","xlink:show",s);break;case"xlinkTitle":we(t,"http://www.w3.org/1999/xlink","xlink:title",s);break;case"xlinkType":we(t,"http://www.w3.org/1999/xlink","xlink:type",s);break;case"xmlBase":we(t,"http://www.w3.org/XML/1998/namespace","xml:base",s);break;case"xmlLang":we(t,"http://www.w3.org/XML/1998/namespace","xml:lang",s);break;case"xmlSpace":we(t,"http://www.w3.org/XML/1998/namespace","xml:space",s);break;case"is":$e(t,"is",s);break;case"innerText":case"textContent":return;default:if(!(2<a.length)||a[0]!=="o"&&a[0]!=="O"||a[1]!=="n"&&a[1]!=="N")a=RS.get(a)||a,$e(t,a,s);else return}Se=!0}function Kd(t,n,a,s,l,u){switch(a){case"style":Rm(t,s,u);return;case"dangerouslySetInnerHTML":if(s!=null){if(typeof s!="object"||!("__html"in s))throw Error(r(61));if(a=s.__html,a!=null){if(l.children!=null)throw Error(r(60));(u!=null?u.__html:void 0)!==a&&(t.innerHTML=a)}}break;case"children":if(typeof s=="string")lr(t,s);else if(typeof s=="number"||typeof s=="bigint")lr(t,""+s);else return;break;case"onScroll":s!=null&&Te("scroll",t);return;case"onScrollEnd":s!=null&&Te("scrollend",t);return;case"onClick":s!=null&&(t.onclick=Yi);return;case"suppressContentEditableWarning":case"suppressHydrationWarning":case"innerHTML":case"ref":return;case"innerText":case"textContent":return;default:if(!xn.hasOwnProperty(a))t:{if(a[0]==="o"&&a[1]==="n"&&(l=a.endsWith("Capture"),u=a.slice(2,l?a.length-7:void 0),n=t[V]||null,n=n!=null?n[a]:null,typeof n=="function"&&t.removeEventListener(u,n,l),typeof s=="function")){typeof n!="function"&&n!==null&&(a in t?t[a]=null:t.hasAttribute(a)&&t.removeAttribute(a)),t.addEventListener(u,s,l);break t}Se=!0,a in t?t[a]=s:s===!0?t.setAttribute(a,""):$e(t,a,s)}return}Se=!0}function Un(t,n,a){switch(n){case"div":case"span":case"svg":case"path":case"a":case"g":case"p":case"li":break;case"img":Te("error",t),Te("load",t);var s=!1,l=!1,u;for(u in a)if(a.hasOwnProperty(u)){var g=a[u];if(g!=null)switch(u){case"src":s=!0;break;case"srcSet":l=!0;break;case"children":case"dangerouslySetInnerHTML":throw Error(r(137,n));default:qe(t,n,u,g,a,null)}}l&&qe(t,n,"srcSet",a.srcSet,a,null),s&&qe(t,n,"src",a.src,a,null);return;case"input":Te("invalid",t);var A=u=g=l=null,z=null,it=null;for(s in a)if(a.hasOwnProperty(s)){var ht=a[s];if(ht!=null)switch(s){case"name":l=ht;break;case"type":g=ht;break;case"checked":z=ht;break;case"defaultChecked":it=ht;break;case"value":u=ht;break;case"defaultValue":A=ht;break;case"children":case"dangerouslySetInnerHTML":if(ht!=null)throw Error(r(137,n));break;default:qe(t,n,s,ht,a,null)}}Em(t,u,A,z,it,g,l,!1);return;case"select":Te("invalid",t),s=g=u=null;for(l in a)if(a.hasOwnProperty(l)&&(A=a[l],A!=null))switch(l){case"value":u=A;break;case"defaultValue":g=A;break;case"multiple":s=A;default:qe(t,n,l,A,a,null)}n=u,a=g,t.multiple=!!s,n!=null?or(t,!!s,n,!1):a!=null&&or(t,!!s,a,!0);return;case"textarea":Te("invalid",t),u=l=s=null;for(g in a)if(a.hasOwnProperty(g)&&(A=a[g],A!=null))switch(g){case"value":s=A;break;case"defaultValue":l=A;break;case"children":u=A;break;case"dangerouslySetInnerHTML":if(A!=null)throw Error(r(91));break;default:qe(t,n,g,A,a,null)}Tm(t,s,l,u);return;case"option":for(z in a)if(a.hasOwnProperty(z)&&(s=a[z],s!=null))switch(z){case"selected":t.selected=s&&typeof s!="function"&&typeof s!="symbol";break;default:qe(t,n,z,s,a,null)}return;case"dialog":Te("beforetoggle",t),Te("toggle",t),Te("cancel",t),Te("close",t);break;case"iframe":case"object":Te("load",t);break;case"video":case"audio":for(s=0;s<al.length;s++)Te(al[s],t);break;case"image":Te("error",t),Te("load",t);break;case"details":Te("toggle",t);break;case"embed":case"source":case"link":Te("error",t),Te("load",t);case"area":case"base":case"br":case"col":case"hr":case"keygen":case"meta":case"param":case"track":case"wbr":case"menuitem":for(it in a)if(a.hasOwnProperty(it)&&(s=a[it],s!=null))switch(it){case"children":case"dangerouslySetInnerHTML":throw Error(r(137,n));default:qe(t,n,it,s,a,null)}return;default:if(Qu(n)){for(ht in a)a.hasOwnProperty(ht)&&(s=a[ht],s!==void 0&&Kd(t,n,ht,s,a,void 0));return}}for(A in a)a.hasOwnProperty(A)&&(s=a[A],s!=null&&qe(t,n,A,s,a,null))}var cM={};function uM(t,n,a,s){switch(n){case"div":case"span":case"svg":case"path":case"a":case"g":case"p":case"li":break;case"input":var l=null,u=null,g=null,A=null,z=null,it=null,ht=null;for(ft in a){var Et=a[ft];if(a.hasOwnProperty(ft)&&Et!=null)switch(ft){case"checked":break;case"value":break;case"defaultValue":z=Et;default:s.hasOwnProperty(ft)||qe(t,n,ft,null,s,Et)}}for(var tt in s){var ft=s[tt];if(Et=a[tt],s.hasOwnProperty(tt)&&(ft!=null||Et!=null))switch(tt){case"type":ft!==Et&&(Se=!0),u=ft;break;case"name":ft!==Et&&(Se=!0),l=ft;break;case"checked":ft!==Et&&(Se=!0),it=ft;break;case"defaultChecked":ft!==Et&&(Se=!0),ht=ft;break;case"value":ft!==Et&&(Se=!0),g=ft;break;case"defaultValue":ft!==Et&&(Se=!0),A=ft;break;case"children":case"dangerouslySetInnerHTML":if(ft!=null)throw Error(r(137,n));break;default:ft!==Et&&qe(t,n,tt,ft,s,Et)}}Zu(t,g,A,z,it,ht,u,l);return;case"select":ft=g=A=tt=null;for(u in a)if(z=a[u],a.hasOwnProperty(u)&&z!=null)switch(u){case"value":break;case"multiple":ft=z;default:s.hasOwnProperty(u)||qe(t,n,u,null,s,z)}for(l in s)if(u=s[l],z=a[l],s.hasOwnProperty(l)&&(u!=null||z!=null))switch(l){case"value":u!==z&&(Se=!0),tt=u;break;case"defaultValue":u!==z&&(Se=!0),A=u;break;case"multiple":u!==z&&(Se=!0),g=u;default:u!==z&&qe(t,n,l,u,s,z)}n=A,a=g,s=ft,tt!=null?or(t,!!a,tt,!1):!!s!=!!a&&(n!=null?or(t,!!a,n,!0):or(t,!!a,a?[]:"",!1));return;case"textarea":ft=tt=null;for(A in a)if(l=a[A],a.hasOwnProperty(A)&&l!=null&&!s.hasOwnProperty(A))switch(A){case"value":break;case"children":break;default:qe(t,n,A,null,s,l)}for(g in s)if(l=s[g],u=a[g],s.hasOwnProperty(g)&&(l!=null||u!=null))switch(g){case"value":l!==u&&(Se=!0),tt=l;break;case"defaultValue":l!==u&&(Se=!0),ft=l;break;case"children":break;case"dangerouslySetInnerHTML":if(l!=null)throw Error(r(91));break;default:l!==u&&qe(t,n,g,l,s,u)}bm(t,tt,ft);return;case"option":for(var Pt in a)if(tt=a[Pt],a.hasOwnProperty(Pt)&&tt!=null&&!s.hasOwnProperty(Pt))switch(Pt){case"selected":t.selected=!1;break;default:qe(t,n,Pt,null,s,tt)}for(z in s)if(tt=s[z],ft=a[z],s.hasOwnProperty(z)&&tt!==ft&&(tt!=null||ft!=null))switch(z){case"selected":tt!==ft&&(Se=!0),t.selected=tt&&typeof tt!="function"&&typeof tt!="symbol";break;default:qe(t,n,z,tt,s,ft)}return;case"img":case"link":case"area":case"base":case"br":case"col":case"embed":case"hr":case"keygen":case"meta":case"param":case"source":case"track":case"wbr":case"menuitem":for(var Qt in a)tt=a[Qt],a.hasOwnProperty(Qt)&&tt!=null&&!s.hasOwnProperty(Qt)&&qe(t,n,Qt,null,s,tt);for(it in s)if(tt=s[it],ft=a[it],s.hasOwnProperty(it)&&tt!==ft&&(tt!=null||ft!=null))switch(it){case"children":case"dangerouslySetInnerHTML":if(tt!=null)throw Error(r(137,n));break;default:qe(t,n,it,tt,s,ft)}return;default:if(Qu(n)){for(var ge in a)tt=a[ge],a.hasOwnProperty(ge)&&tt!==void 0&&!s.hasOwnProperty(ge)&&Kd(t,n,ge,void 0,s,tt);for(ht in s)tt=s[ht],ft=a[ht],!s.hasOwnProperty(ht)||tt===ft||tt===void 0&&ft===void 0||Kd(t,n,ht,tt,s,ft);return}}for(var nt in a)tt=a[nt],a.hasOwnProperty(nt)&&tt!=null&&!s.hasOwnProperty(nt)&&qe(t,n,nt,null,s,tt);for(Et in s)tt=s[Et],ft=a[Et],!s.hasOwnProperty(Et)||tt===ft||tt==null&&ft==null||qe(t,n,Et,tt,s,ft)}function Lv(t){switch(t){case"css":case"script":case"font":case"img":case"image":case"input":case"link":return!0;default:return!1}}function fM(){if(typeof performance.getEntriesByType=="function"){for(var t=0,n=0,a=performance.getEntriesByType("resource"),s=0;s<a.length;s++){var l=a[s],u=l.transferSize,g=l.initiatorType,A=l.duration;if(u&&A&&Lv(g)){for(g=0,A=l.responseEnd,s+=1;s<a.length;s++){var z=a[s],it=z.startTime;if(it>A)break;var ht=z.transferSize,Et=z.initiatorType;ht&&Lv(Et)&&(z=z.responseEnd,g+=ht*(z<A?1:(A-it)/(z-it)))}if(--s,n+=8*(u+g)/(l.duration/1e3),t++,10<t)break}}if(0<t)return n/t/1e6}return navigator.connection&&(t=navigator.connection.downlink,typeof t=="number")?t:5}var Qd=null,Jd=null;function rl(t){return t.nodeType===9?t:t.ownerDocument}function Ov(t){switch(t){case"http://www.w3.org/2000/svg":return 1;case"http://www.w3.org/1998/Math/MathML":return 2;default:return 0}}function Pv(t,n){if(t===0)switch(n){case"svg":return 1;case"math":return 2;default:return 0}return t===1&&n==="foreignObject"?0:t}function Iv(t,n,a,s){return a=rl(a).createElement(t),a[T]=s,a[V]=n,Un(a,t,n),xe(a),a}function $d(t,n){return t==="textarea"||t==="noscript"||typeof n.children=="string"||typeof n.children=="number"||typeof n.children=="bigint"||typeof n.dangerouslySetInnerHTML=="object"&&n.dangerouslySetInnerHTML!==null&&n.dangerouslySetInnerHTML.__html!=null}var th=null;function dM(){var t=window.event;return t&&t.type==="popstate"?t===th?!1:(th=t,!0):(th=null,!1)}var eh=typeof setTimeout=="function"?setTimeout:void 0,hM=typeof clearTimeout=="function"?clearTimeout:void 0,Bv=typeof Promise=="function"?Promise:void 0,zv=typeof requestAnimationFrame=="function"?requestAnimationFrame:eh,pM=typeof queueMicrotask=="function"?queueMicrotask:typeof Bv<"u"?function(t){return Bv.resolve(null).then(t).catch(mM)}:eh;function mM(t){setTimeout(function(){throw t})}function ls(t){return t==="head"}function Fv(t,n){var a=n,s=0;do{var l=a.nextSibling;if(t.removeChild(a),l&&l.nodeType===8)if(a=l.data,a==="/$"||a==="/&"){if(s===0){t.removeChild(l),Wr(n);return}s--}else if(a==="$"||a==="$?"||a==="$~"||a==="$!"||a==="&")s++;else if(a==="html")ch(t.ownerDocument.documentElement);else if(a==="head"){a=t.ownerDocument.head,ch(a);for(var u=a.firstChild;u;){var g=u.nextSibling,A=u.nodeName;u[Ot]||A==="SCRIPT"||A==="STYLE"||A==="LINK"&&u.rel.toLowerCase()==="stylesheet"||a.removeChild(u),u=g}}else a==="body"&&ch(t.ownerDocument.body);a=l}while(a);Wr(n)}function Hv(t,n){var a=t;t=0;do{var s=a.nextSibling;if(a.nodeType===1?n?(a._stashedDisplay=a.style.display,a.style.display="none"):(a.style.display=a._stashedDisplay||"",a.getAttribute("style")===""&&a.removeAttribute("style")):a.nodeType===3&&(n?(a._stashedText=a.nodeValue,a.nodeValue=""):a.nodeValue=a._stashedText||""),s&&s.nodeType===8)if(a=s.data,a==="/$"){if(t===0)break;t--}else a!=="$"&&a!=="$?"&&a!=="$~"&&a!=="$!"||t++;a=s}while(a)}function Gv(t,n,a){if(n=CSS.escape(n)!==n?"r-"+btoa(n).replace(/=/g,""):n,t.style.viewTransitionName=n,a!=null&&(t.style.viewTransitionClass=a),a=getComputedStyle(t),a.display==="inline"){if(n=t.getClientRects(),n.length===1)var s=1;else for(var l=s=0;l<n.length;l++){var u=n[l];0<u.width&&0<u.height&&s++}s===1&&(t=t.style,t.display=n.length===1?"inline-block":"block",t.marginTop="-"+a.paddingTop,t.marginBottom="-"+a.paddingBottom)}}function Vv(t,n){t=t.style,n=n.style;var a=n!=null?n.hasOwnProperty("viewTransitionName")?n.viewTransitionName:n.hasOwnProperty("view-transition-name")?n["view-transition-name"]:null:null;t.viewTransitionName=a==null||typeof a=="boolean"?"":(""+a).trim(),a=n!=null?n.hasOwnProperty("viewTransitionClass")?n.viewTransitionClass:n.hasOwnProperty("view-transition-class")?n["view-transition-class"]:null:null,t.viewTransitionClass=a==null||typeof a=="boolean"?"":(""+a).trim(),t.display==="inline-block"&&(n==null?t.display=t.margin="":(a=n.display,t.display=a==null||typeof a=="boolean"?"":a,a=n.margin,a!=null?t.margin=a:(a=n.hasOwnProperty("marginTop")?n.marginTop:n["margin-top"],t.marginTop=a==null||typeof a=="boolean"?"":a,n=n.hasOwnProperty("marginBottom")?n.marginBottom:n["margin-bottom"],t.marginBottom=n==null||typeof n=="boolean"?"":n)))}function gM(t,n,a){return a=a.ownerDocument.defaultView,{rect:t,abs:n.position==="absolute"||n.position==="fixed",clip:n.clipPath!=="none"||n.overflow!=="visible"||n.filter!=="none"||n.mask!=="none"||n.mask!=="none"||n.borderRadius!=="0px",view:0<=t.bottom&&0<=t.right&&t.top<=a.innerHeight&&t.left<=a.innerWidth}}function nh(t){var n=t.getBoundingClientRect(),a=getComputedStyle(t);return gM(n,a,t)}function vM(t){return t.documentElement.clientHeight}function _M(t){this.addEventListener("load",t),this.addEventListener("error",t)}function xM(t,n,a,s,l,u,g,A,z){var it=n.nodeType===9?n:n.ownerDocument;try{var ht=it.startViewTransition({update:function(){var tt=it.defaultView,ft=tt.navigation&&tt.navigation.transition,Pt=it.fonts.status;s();var Qt=[];if(Pt==="loaded"&&(vM(it),it.fonts.status==="loading"&&Qt.push(it.fonts.ready)),Pt=Qt.length,t!==null)for(var ge=t.suspenseyImages,nt=0,W=0;W<ge.length;W++){var rt=ge[W];if(!rt.complete){var Mt=rt.getBoundingClientRect();if(0<Mt.bottom&&0<Mt.right&&Mt.top<tt.innerHeight&&Mt.left<tt.innerWidth){if(nt+=c_(rt),nt>Jc){Qt.length=Pt;break}rt=new Promise(_M.bind(rt)),Qt.push(rt)}}}if(0<Qt.length)return tt=Promise.race([Promise.all(Qt),new Promise(function(qt){return setTimeout(qt,500)})]).then(l,l),(ft?Promise.allSettled([ft.finished,tt]):tt).then(u,u);if(l(),ft)return ft.finished.then(u,u);u()},types:a});it.__reactViewTransition=ht;var Et=[];return ht.ready.then(function(){for(var tt=it.documentElement.getAnimations({subtree:!0}),ft=0;ft<tt.length;ft++){var Pt=tt[ft],Qt=Pt.effect,ge=Qt.pseudoElement;if(ge!=null&&ge.startsWith("::view-transition")){Et.push(Pt),Pt=Qt.getKeyframes();for(var nt=ge=void 0,W=!0,rt=0;rt<Pt.length;rt++){var Mt=Pt[rt],qt=Mt.width;if(ge===void 0)ge=qt;else if(ge!==qt){W=!1;break}if(qt=Mt.height,nt===void 0)nt=qt;else if(nt!==qt){W=!1;break}delete Mt.width,delete Mt.height,Mt.transform==="none"&&delete Mt.transform}W&&ge!==void 0&&nt!==void 0&&(Qt.setKeyframes(Pt),W=getComputedStyle(Qt.target,Qt.pseudoElement),W.width!==ge||W.height!==nt)&&(W=Pt[0],W.width=ge,W.height=nt,W=Pt[Pt.length-1],W.width=ge,W.height=nt,Qt.setKeyframes(Pt))}}g()},function(tt){it.__reactViewTransition===ht&&(it.__reactViewTransition=null);try{if(typeof tt=="object"&&tt!==null)switch(tt.name){case"InvalidStateError":(tt.message==="View transition was skipped because document visibility state is hidden."||tt.message==="Skipping view transition because document visibility state has become hidden."||tt.message==="Skipping view transition because viewport size changed."||tt.message==="Transition was aborted because of invalid state")&&(tt=null)}tt!==null&&z(tt)}finally{s(),l(),g()}}),ht.finished.finally(function(){for(var tt=0;tt<Et.length;tt++)Et[tt].cancel();it.__reactViewTransition===ht&&(it.__reactViewTransition=null),A()}),ht}catch{return s(),l(),g(),null}}function Ws(t,n){this._scope=document.documentElement,this._selector="::view-transition-"+t+"("+n+")"}Ws.prototype.animate=function(t,n){return n=typeof n=="number"?{duration:n}:B({},n),n.pseudoElement=this._selector,this._scope.animate(t,n)},Ws.prototype.getAnimations=function(){for(var t=this._scope,n=this._selector,a=t.getAnimations({subtree:!0}),s=[],l=0;l<a.length;l++){var u=a[l].effect;u!==null&&u.target===t&&u.pseudoElement===n&&s.push(a[l])}return s},Ws.prototype.getComputedStyle=function(){return getComputedStyle(this._scope,this._selector)};function kv(t){return{name:t,group:new Ws("group",t),imagePair:new Ws("image-pair",t),old:new Ws("old",t),new:new Ws("new",t)}}function fi(t){this._fragmentFiber=t,this._observers=this._eventListeners=null}fi.prototype.addEventListener=function(t,n,a){var s=null,l=null;if(!(a!=null&&typeof a!="boolean"&&(s=a.signal||null,s!==null&&s.aborted))){this._eventListeners===null&&(this._eventListeners=[]);var u=this._eventListeners;if(Wv(u,t,n,a)===-1){var g=this,A=n;a!=null&&typeof a!="boolean"&&a.once===!0&&(A=function(z){g.removeEventListener(t,n,a),typeof n=="function"?n.call(this,z):n.handleEvent(z)}),s!==null&&(l=g.removeEventListener.bind(g,t,n,a),s.addEventListener("abort",l,{once:!0}),l=s.removeEventListener.bind(s,"abort",l)),s=Fr(a),u.push({type:t,listener:n,optionsOrUseCapture:a,attachedListener:A,cleanup:l}),v(this._fragmentFiber.child,!1,SM,t,A,s)}this._eventListeners=u}};function SM(t,n,a,s){return y(t).addEventListener(n,a,s),!1}fi.prototype.removeEventListener=function(t,n,a){var s=this._eventListeners;if(s!==null&&(n=Wv(s,t,n,a),n!==-1)){var l=s[n];a=l.attachedListener;var u=l.cleanup;l=Fr(l.optionsOrUseCapture),v(this._fragmentFiber.child,!1,yM,t,a,l),s.splice(n,1),u!==null&&u()}};function yM(t,n,a,s){return y(t).removeEventListener(n,a,s),!1}function Fr(t){return t!=null&&typeof t!="boolean"&&(t.once===!0||t.signal instanceof AbortSignal)?{capture:t.capture,passive:t.passive}:t}function Xv(t){return t==null?"c=0":typeof t=="boolean"?"c="+(t?"1":"0"):"c="+(t.capture?"1":"0")}function Wv(t,n,a,s){if(t.length===0)return-1;s=Xv(s);for(var l=0;l<t.length;l++){var u=t[l];if(u.type===n&&u.listener===a&&Xv(u.optionsOrUseCapture)===s)return l}return-1}fi.prototype.dispatchEvent=function(t){var n=_(this._fragmentFiber);if(n===null)return!0;n=y(n);var a=this._eventListeners;if(a!==null&&0<a.length||!t.bubbles){var s=n.nodeType===9?n.createComment(""):document.createTextNode("");if(a)for(var l=0;l<a.length;l++){var u=a[l];s.addEventListener(u.type,u.attachedListener,Fr(u.optionsOrUseCapture))}if(n.appendChild(s),t=s.dispatchEvent(t),a)for(l=0;l<a.length;l++)u=a[l],s.removeEventListener(u.type,u.attachedListener,Fr(u.optionsOrUseCapture));return n.removeChild(s),t}return n.dispatchEvent(t)},fi.prototype.focus=function(t){v(this._fragmentFiber.child,!0,qv,t,void 0,void 0)};function qv(t,n){return t.tag===6?!1:(t=y(t),LM(t,n))}fi.prototype.focusLast=function(t){var n=[];v(this._fragmentFiber.child,!0,ih,n,void 0,void 0);for(var a=n.length-1;0<=a&&!qv(n[a],t);a--);};function ih(t,n){return n.push(t),!1}fi.prototype.blur=function(){var t=_(this._fragmentFiber);t!==null&&(t=y(t),t=rl(t).activeElement,t!==null&&v(this._fragmentFiber.child,!1,MM,t,void 0,void 0))};function MM(t,n){return t.tag===6?!1:(t=y(t),t===n||t.contains(n)?(n.blur(),!0):!1)}fi.prototype.observeUsing=function(t){this._observers===null&&(this._observers=new Set),this._observers.add(t),v(this._fragmentFiber.child,!1,EM,t,void 0,void 0)};function EM(t,n){return t.tag===6||(t=y(t),n.observe(t)),!1}fi.prototype.unobserveUsing=function(t){var n=this._observers;if(n!==null&&n.has(t)){n.delete(t),v(this._fragmentFiber.child,!1,bM,t,void 0,void 0);for(var a=n=0;a<Li.length;a++){var s=Li[a];s.fragmentInstance===this&&s.observer===t?t.unobserve(s.instance):Li[n++]=s}Li.length=n}};function bM(t,n){return t.tag===6||(t=y(t),n.unobserve(t)),!1}var Li=[],ah=!1;function TM(t,n,a){Li.push({fragmentInstance:t,observer:n,instance:a}),ah||(ah=!0,OM(function(){ah=!1;var s=Li;Li=[];for(var l=0;l<s.length;l++){var u=s[l];u.observer.unobserve(u.instance)}}))}fi.prototype.getClientRects=function(){var t=[];return v(this._fragmentFiber.child,!1,AM,t,void 0,void 0),t};function AM(t,n){if(t.tag===6){t=t.stateNode;var a=t.ownerDocument.createRange();a.selectNodeContents(t),n.push.apply(n,a.getClientRects())}else t=y(t),n.push.apply(n,t.getClientRects());return!1}fi.prototype.getRootNode=function(t){var n=_(this._fragmentFiber);return n===null?this:y(n).getRootNode(t)},fi.prototype.compareDocumentPosition=function(t){var n=_(this._fragmentFiber);if(n===null)return Node.DOCUMENT_POSITION_DISCONNECTED;var a=[];v(this._fragmentFiber.child,!1,ih,a,void 0,void 0);var s=y(n);if(a.length===0){if(a=s,M(this._fragmentFiber)){t:{for(n=this._fragmentFiber.return;n!==null;){if(n.tag===4){n=n.stateNode.containerInfo;break t}if(n.tag===3||n.tag===5||n.tag===27)break;n=n.return}n=null}n!=null&&(a=n)}n=this._fragmentFiber;var l=s=a.compareDocumentPosition(t);return a===t?l=Node.DOCUMENT_POSITION_CONTAINS:s&Node.DOCUMENT_POSITION_CONTAINED_BY&&(a=R(n)[1],a===null?l=Node.DOCUMENT_POSITION_PRECEDING:(t=y(a).compareDocumentPosition(t),l=t===0||t&Node.DOCUMENT_POSITION_FOLLOWING?Node.DOCUMENT_POSITION_FOLLOWING:Node.DOCUMENT_POSITION_PRECEDING)),l|=Node.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC}n=y(a[0]),l=y(a[a.length-1]);var u=M(this._fragmentFiber)?n.parentElement:s;if(u==null)return Node.DOCUMENT_POSITION_DISCONNECTED;s=u.compareDocumentPosition(n)&Node.DOCUMENT_POSITION_CONTAINED_BY,u=u.compareDocumentPosition(l)&Node.DOCUMENT_POSITION_CONTAINED_BY;var g=n.compareDocumentPosition(t),A=l.compareDocumentPosition(t),z=g&Node.DOCUMENT_POSITION_CONTAINED_BY||A&Node.DOCUMENT_POSITION_CONTAINED_BY;return A=s&&u&&g&Node.DOCUMENT_POSITION_FOLLOWING&&A&Node.DOCUMENT_POSITION_PRECEDING,n=s&&n===t||u&&l===t||z||A?Node.DOCUMENT_POSITION_CONTAINED_BY:!s&&n===t||!u&&l===t?Node.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC:g,n&Node.DOCUMENT_POSITION_DISCONNECTED||n&Node.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC||RM(n,this._fragmentFiber,a[0],a[a.length-1],t)?n:Node.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC};function RM(t,n,a,s,l){var u=se(l);if(t&Node.DOCUMENT_POSITION_CONTAINED_BY){if(a=!!u)t:{for(;u!==null;){if(u.tag===7&&(u===n||u.alternate===n)){a=!0;break t}u=u.return}a=!1}return a}if(t&Node.DOCUMENT_POSITION_CONTAINS){if(u===null)return u=l.ownerDocument,l===u||l===u.documentElement||l===u.body;t:{for(u=n,n=_(n);u!==null;){if(!(u.tag!==5&&u.tag!==3&&u.tag!==27||u!==n&&u.alternate!==n)){u=!0;break t}u=u.return}u=!1}return u}return t&Node.DOCUMENT_POSITION_PRECEDING?((n=!!u)&&!(n=u===a)&&(n=U(a,u,D),n===null?n=!1:(v(n,!0,k,u,a),u=S,S=null,n=u!==null)),n):t&Node.DOCUMENT_POSITION_FOLLOWING?((n=!!u)&&!(n=u===s)&&(n=U(s,u,D),n===null?n=!1:(v(n,!0,C,u,s),u=S,O=S=null,n=u!==null)),n):!1}function Yv(t,n){var a=t.ownerDocument.createRange();a.selectNodeContents(t),t=a.getBoundingClientRect(),window.scrollTo(window.scrollX+t.left,n?window.scrollY+t.top:window.scrollY+t.bottom-window.innerHeight)}fi.prototype.scrollIntoView=function(t){if(typeof t=="object")throw Error(r(566));var n=[];v(this._fragmentFiber.child,!1,ih,n,void 0,void 0);var a=t!==!1;if(n.length===0){var s=R(this._fragmentFiber);if(s=a?s[1]||s[0]||_(this._fragmentFiber):s[0]||s[1],s===null)return;if(s.tag===6){t=y(s),Yv(t,a);return}if(s=y(s),s.nodeType!==9){if(s.nodeType===11){a="host"in s?s.host:null,a!==null&&a.scrollIntoView(t);return}s.scrollIntoView(t)}}for(s=a?n.length-1:0;s!==(a?-1:n.length);){var l=n[s];l.tag===6?(l=y(l),Yv(l,a)):y(l).scrollIntoView(t),s+=a?-1:1}};function wM(t,n){return t=y(t),jv(t,n),!1}function jv(t,n){t.reactFragments==null&&(t.reactFragments=new Set),t.reactFragments.add(n)}function Zv(t,n){var a=n._eventListeners;if(a!==null)for(var s=0;s<a.length;s++){var l=a[s];t.addEventListener(l.type,l.attachedListener,Fr(l.optionsOrUseCapture))}t.nodeType!==3&&(a=n._observers,a!==null&&a.forEach(function(u){for(var g=0,A=0;A<Li.length;A++){var z=Li[A];(z.fragmentInstance!==n||z.observer!==u||z.instance!==t)&&(Li[g++]=z)}Li.length=g,u.observe(t)}),jv(t,n))}function CM(t,n){var a=n._eventListeners;if(a!==null)for(var s=0;s<a.length;s++){var l=a[s];t.removeEventListener(l.type,l.attachedListener,Fr(l.optionsOrUseCapture))}t.nodeType!==3&&(a=n._observers,a!==null&&a.forEach(function(u){typeof u.rootMargin=="string"?TM(n,u,t):u.unobserve(t)}),t.reactFragments!=null&&t.reactFragments.delete(n))}function sh(t){var n=t.firstChild;for(n&&n.nodeType===10&&(n=n.nextSibling);n;){var a=n;switch(n=n.nextSibling,a.nodeName){case"HTML":case"HEAD":case"BODY":sh(a),Kt(a);continue;case"SCRIPT":case"STYLE":continue;case"LINK":if(a.rel.toLowerCase()==="stylesheet")continue}t.removeChild(a)}}function NM(t,n,a,s){for(;t.nodeType===1;){var l=a;if(t.nodeName.toLowerCase()!==n.toLowerCase()){if(!s&&(t.nodeName!=="INPUT"||t.type!=="hidden"))break}else if(s){if(!t[Ot])switch(n){case"meta":if(!t.hasAttribute("itemprop"))break;return t;case"link":if(u=t.getAttribute("rel"),u==="stylesheet"&&t.hasAttribute("data-precedence"))break;if(u!==l.rel||t.getAttribute("href")!==(l.href==null||l.href===""?null:l.href)||t.getAttribute("crossorigin")!==(l.crossOrigin==null?null:l.crossOrigin)||t.getAttribute("title")!==(l.title==null?null:l.title))break;return t;case"style":if(t.hasAttribute("data-precedence"))break;return t;case"script":if(u=t.getAttribute("src"),(u!==(l.src==null?null:l.src)||t.getAttribute("type")!==(l.type==null?null:l.type)||t.getAttribute("crossorigin")!==(l.crossOrigin==null?null:l.crossOrigin))&&u&&t.hasAttribute("async")&&!t.hasAttribute("itemprop"))break;return t;default:return t}}else if(n==="input"&&t.type==="hidden"){var u=l.name==null?null:""+l.name;if(l.type==="hidden"&&t.getAttribute("name")===u)return t}else return t;if(t=bi(t.nextSibling),t===null)break}return null}function DM(t,n,a){if(n==="")return null;for(;t.nodeType!==3;)if((t.nodeType!==1||t.nodeName!=="INPUT"||t.type!=="hidden")&&!a||(t=bi(t.nextSibling),t===null))return null;return t}function Kv(t,n){for(;t.nodeType!==8;)if((t.nodeType!==1||t.nodeName!=="INPUT"||t.type!=="hidden")&&!n||(t=bi(t.nextSibling),t===null))return null;return t}function rh(t){return t.data==="$?"||t.data==="$~"}function oh(t){return t.data==="$!"||t.data==="$?"&&t.ownerDocument.readyState!=="loading"}function UM(t,n){var a=t.ownerDocument;if(t.data==="$~")t._reactRetry=n;else if(t.data!=="$?"||a.readyState!=="loading")n();else{var s=function(){n(),a.removeEventListener("DOMContentLoaded",s)};a.addEventListener("DOMContentLoaded",s),t._reactRetry=s}}function bi(t){for(;t!=null;t=t.nextSibling){var n=t.nodeType;if(n===1||n===3)break;if(n===8){if(n=t.data,n==="$"||n==="$!"||n==="$?"||n==="$~"||n==="&"||n==="F!"||n==="F")break;if(n==="/$"||n==="/&")return null}}return t}var lh=null;function Qv(t){t=t.nextSibling;for(var n=0;t;){if(t.nodeType===8){var a=t.data;if(a==="/$"||a==="/&"){if(n===0)return bi(t.nextSibling);n--}else a!=="$"&&a!=="$!"&&a!=="$?"&&a!=="$~"&&a!=="&"||n++}t=t.nextSibling}return null}function Jv(t){t=t.previousSibling;for(var n=0;t;){if(t.nodeType===8){var a=t.data;if(a==="$"||a==="$!"||a==="$?"||a==="$~"||a==="&"){if(n===0)return t;n--}else a!=="/$"&&a!=="/&"||n++}t=t.previousSibling}return null}function LM(t,n){function a(){s=!0}if(t.ownerDocument.activeElement===t)return!0;var s=!1;try{t.ownerDocument.addEventListener("focus",a,!0),(t.focus||HTMLElement.prototype.focus).call(t,n)}finally{t.ownerDocument.removeEventListener("focus",a,!0)}return s}function OM(t){zv(function(){zv(function(n){return t(n)})})}function $v(t,n,a){switch(n=rl(a),t){case"html":if(t=n.documentElement,!t)throw Error(r(452));return t;case"head":if(t=n.head,!t)throw Error(r(453));return t;case"body":if(t=n.body,!t)throw Error(r(454));return t;default:throw Error(r(451))}}function t_(t,n,a){for(var s in a){var l=a[s];a.hasOwnProperty(s)&&l!=null&&qe(t,n,s,null,cM,l)}a.dangerouslySetInnerHTML!=null&&(t.textContent=""),t.onclick===Yi&&(t.onclick=null),Kt(t)}function ch(t){for(var n=t.attributes;n.length;)t.removeAttributeNode(n[0]);Kt(t)}var Ti=new Map,e_=new Set;function ol(t){if(typeof t.getRootNode=="function"){var n=t.getRootNode();if(n.nodeType===9||n.nodeType===11)return n}return t.nodeType===9?t:t.ownerDocument}var wa=Ct.d;Ct.d={f:PM,r:IM,D:BM,C:zM,L:FM,m:HM,X:VM,S:GM,M:kM};function PM(){var t=wa.f(),n=kc();return t||n}function IM(t){var n=fe(t);n!==null&&n.tag===5&&n.type==="form"?i0(n):wa.r(t)}var Hr=typeof document>"u"?null:document;function n_(t,n,a){var s=Hr;if(s&&typeof n=="string"&&n){var l=vi(n);l='link[rel="'+t+'"][href="'+l+'"]',typeof a=="string"&&(l+='[crossorigin="'+a+'"]'),e_.has(l)||(e_.add(l),t={rel:t,crossOrigin:a,href:n},s.querySelector(l)===null&&(n=s.createElement("link"),Un(n,"link",t),xe(n),s.head.appendChild(n)))}}function BM(t){wa.D(t),n_("dns-prefetch",t,null)}function zM(t,n){wa.C(t,n),n_("preconnect",t,n)}function FM(t,n,a){wa.L(t,n,a);var s=Hr;if(s&&t&&n){var l='link[rel="preload"][as="'+vi(n)+'"]';n==="image"&&a&&a.imageSrcSet?(l+='[imagesrcset="'+vi(a.imageSrcSet)+'"]',typeof a.imageSizes=="string"&&(l+='[imagesizes="'+vi(a.imageSizes)+'"]')):l+='[href="'+vi(t)+'"]';var u=l;switch(n){case"style":u=Gr(t);break;case"script":u=Vr(t)}if(!(Ti.has(u)||(t=B({rel:"preload",href:n==="image"&&a&&a.imageSrcSet?void 0:t,as:n},a),Ti.set(u,t),s.querySelector(l)!==null||n==="style"&&s.querySelector(ll(u))||n==="script"&&s.querySelector(cl(u))))){var g=s.createElement("link");Un(g,"link",t),n==="style"&&(g[jt]=!0,g.onload=g.onerror=function(){Ze(g)}),xe(g),s.head.appendChild(g)}}}function HM(t,n){wa.m(t,n);var a=Hr;if(a&&t){var s=n&&typeof n.as=="string"?n.as:"script",l='link[rel="modulepreload"][as="'+vi(s)+'"][href="'+vi(t)+'"]',u=l;switch(s){case"audioworklet":case"paintworklet":case"serviceworker":case"sharedworker":case"worker":case"script":u=Vr(t)}if(!Ti.has(u)&&(t=B({rel:"modulepreload",href:t},n),Ti.set(u,t),a.querySelector(l)===null)){switch(s){case"audioworklet":case"paintworklet":case"serviceworker":case"sharedworker":case"worker":case"script":if(a.querySelector(cl(u)))return}s=a.createElement("link"),Un(s,"link",t),xe(s),a.head.appendChild(s)}}}function GM(t,n,a){wa.S(t,n,a);var s=Hr;if(s&&t){var l=Ee(s).hoistableStyles,u=Gr(t);n=n||"default";var g=l.get(u);if(!g){var A={loading:0,preload:null};if(g=s.querySelector(ll(u)))A.loading=5;else{t=B({rel:"stylesheet",href:t,"data-precedence":n},a),(a=Ti.get(u))&&uh(t,a);var z=g=s.createElement("link");xe(z),Un(z,"link",t),z._p=new Promise(function(it,ht){z.onload=it,z.onerror=ht}),z.addEventListener("load",function(){A.loading|=1}),z.addEventListener("error",function(){A.loading|=2}),A.loading|=4,Kc(g,n,s)}g={type:"stylesheet",instance:g,count:1,state:A},l.set(u,g)}}}function VM(t,n){wa.X(t,n);var a=Hr;if(a&&t){var s=Ee(a).hoistableScripts,l=Vr(t),u=s.get(l);u||(u=a.querySelector(cl(l)),u||(t=B({src:t,async:!0},n),(n=Ti.get(l))&&fh(t,n),u=a.createElement("script"),xe(u),Un(u,"link",t),a.head.appendChild(u)),u={type:"script",instance:u,count:1,state:null},s.set(l,u))}}function kM(t,n){wa.M(t,n);var a=Hr;if(a&&t){var s=Ee(a).hoistableScripts,l=Vr(t),u=s.get(l);u||(u=a.querySelector(cl(l)),u||(t=B({src:t,async:!0,type:"module"},n),(n=Ti.get(l))&&fh(t,n),u=a.createElement("script"),xe(u),Un(u,"link",t),a.head.appendChild(u)),u={type:"script",instance:u,count:1,state:null},s.set(l,u))}}function i_(t,n,a,s){var l=(l=Ue.current)?ol(l):null;if(!l)throw Error(r(446));switch(t){case"meta":case"title":return null;case"style":return typeof a.precedence=="string"&&typeof a.href=="string"?(a=Gr(a.href),n=Ee(l).hoistableStyles,s=n.get(a),s||(s={type:"style",instance:null,count:0,state:null},n.set(a,s)),s):{type:"void",instance:null,count:0,state:null};case"link":if(a.rel==="stylesheet"&&typeof a.href=="string"&&typeof a.precedence=="string"){t=Gr(a.href);var u=Ee(l).hoistableStyles,g=u.get(t);if(g||(l=l.ownerDocument||l,g={type:"stylesheet",instance:null,count:0,state:{loading:0,preload:null}},u.set(t,g),(u=l.querySelector(ll(t)))?u._p||(g.instance=u,g.state.loading=5):(u=Ti.get(t),u||(u={rel:"preload",as:"style",href:a.href,crossOrigin:a.crossOrigin,integrity:a.integrity,media:a.media,hrefLang:a.hrefLang,referrerPolicy:a.referrerPolicy},Ti.set(t,u)),XM(l,t,u,g.state))),n&&s===null)throw Error(r(528,""));return g}if(n&&s!==null)throw Error(r(529,""));return null;case"script":return n=a.async,a=a.src,typeof a=="string"&&n&&typeof n!="function"&&typeof n!="symbol"?(a=Vr(a),n=Ee(l).hoistableScripts,s=n.get(a),s||(s={type:"script",instance:null,count:0,state:null},n.set(a,s)),s):{type:"void",instance:null,count:0,state:null};default:throw Error(r(444,t))}}function Gr(t){return'href="'+vi(t)+'"'}function ll(t){return'link[rel="stylesheet"]['+t+"]"}function a_(t){return B({},t,{"data-precedence":t.precedence,precedence:null})}function XM(t,n,a,s){if(n=t.querySelector('link[rel="preload"][as="style"]['+n+"]")){if(n[jt]!==!0){s.loading=1;return}}else n=t.createElement("link"),n[jt]=!0,n.onload=n.onerror=Ze.bind(null,n),Un(n,"link",a),xe(n),t.head.appendChild(n);s.preload=n,n.addEventListener("load",function(){return s.loading|=1}),n.addEventListener("error",function(){return s.loading|=2})}function Vr(t){return'[src="'+vi(t)+'"]'}function cl(t){return"script[async]"+t}function s_(t,n,a){if(n.count++,n.instance===null)switch(n.type){case"style":var s=t.querySelector('style[data-href~="'+vi(a.href)+'"]');if(s)return n.instance=s,xe(s),s;var l=B({},a,{"data-href":a.href,"data-precedence":a.precedence,href:null,precedence:null});return s=(t.ownerDocument||t).createElement("style"),xe(s),Un(s,"style",l),Kc(s,a.precedence,t),n.instance=s;case"stylesheet":l=Gr(a.href);var u=t.querySelector(ll(l));if(u)return n.state.loading|=4,n.instance=u,xe(u),u;s=a_(a),(l=Ti.get(l))&&uh(s,l),u=(t.ownerDocument||t).createElement("link"),xe(u);var g=u;return g._p=new Promise(function(A,z){g.onload=A,g.onerror=z}),Un(u,"link",s),n.state.loading|=4,Kc(u,a.precedence,t),n.instance=u;case"script":return u=Vr(a.src),(l=t.querySelector(cl(u)))?(n.instance=l,xe(l),l):(s=a,(l=Ti.get(u))&&(s=B({},a),fh(s,l)),t=t.ownerDocument||t,l=t.createElement("script"),xe(l),Un(l,"link",s),t.head.appendChild(l),n.instance=l);case"void":return null;default:throw Error(r(443,n.type))}else n.type==="stylesheet"&&(n.state.loading&4)===0&&(s=n.instance,n.state.loading|=4,Kc(s,a.precedence,t));return n.instance}function Kc(t,n,a){for(var s=a.querySelectorAll('link[rel="stylesheet"][data-precedence],style[data-precedence]'),l=s.length?s[s.length-1]:null,u=l,g=0;g<s.length;g++){var A=s[g];if(A.dataset.precedence===n)u=A;else if(u!==l)break}u?u.parentNode.insertBefore(t,u.nextSibling):(n=a.nodeType===9?a.head:a,n.insertBefore(t,n.firstChild))}function uh(t,n){t.crossOrigin==null&&(t.crossOrigin=n.crossOrigin),t.referrerPolicy==null&&(t.referrerPolicy=n.referrerPolicy),t.title==null&&(t.title=n.title)}function fh(t,n){t.crossOrigin==null&&(t.crossOrigin=n.crossOrigin),t.referrerPolicy==null&&(t.referrerPolicy=n.referrerPolicy),t.integrity==null&&(t.integrity=n.integrity)}var Qc=null;function r_(t,n,a){if(Qc===null){var s=new Map,l=Qc=new Map;l.set(a,s)}else l=Qc,s=l.get(a),s||(s=new Map,l.set(a,s));if(s.has(t))return s;for(s.set(t,null),a=a.getElementsByTagName(t),l=0;l<a.length;l++){var u=a[l];if(!(u[Ot]||u[T]||t==="link"&&u.getAttribute("rel")==="stylesheet")&&u.namespaceURI!=="http://www.w3.org/2000/svg"){var g=u.getAttribute(n)||"";g=t+g;var A=s.get(g);A?A.push(u):s.set(g,[u])}}return s}function dh(t,n,a){t=t.ownerDocument||t,t.head.insertBefore(a,n==="title"?t.querySelector("head > title"):null)}function WM(t,n,a){if(a===1||n.itemProp!=null)return!1;switch(t){case"meta":case"title":return!0;case"style":if(typeof n.precedence!="string"||typeof n.href!="string"||n.href==="")break;return!0;case"link":if(typeof n.rel!="string"||typeof n.href!="string"||n.href===""||n.onLoad||n.onError)break;switch(n.rel){case"stylesheet":return t=n.disabled,typeof n.precedence=="string"&&t==null;default:return!0}case"script":if(n.async&&typeof n.async!="function"&&typeof n.async!="symbol"&&!n.onLoad&&!n.onError&&n.src&&typeof n.src=="string")return!0}return!1}function o_(t,n){return t==="img"&&n.src!=null&&n.src!==""&&n.onLoad==null&&n.loading!=="lazy"}function l_(t){return!(t.type==="stylesheet"&&(t.state.loading&3)===0)}function c_(t){return(t.width||100)*(t.height||100)*(typeof devicePixelRatio=="number"?devicePixelRatio:1)*.25}function u_(t,n){typeof n.decode=="function"&&(t.imgCount++,n.complete||(t.imgBytes+=c_(n),t.suspenseyImages.push(n)),t=jM.bind(t),n.decode().then(t,t))}function qM(t,n,a,s){if(a.type==="stylesheet"&&(typeof s.media!="string"||matchMedia(s.media).matches!==!1)&&(a.state.loading&4)===0){if(a.instance===null){var l=Gr(s.href),u=n.querySelector(ll(l));if(u){n=u._p,n!==null&&typeof n=="object"&&typeof n.then=="function"&&(t.count++,t=ul.bind(t),n.then(t,t)),a.state.loading|=4,a.instance=u,xe(u);return}u=n.ownerDocument||n,s=a_(s),(l=Ti.get(l))&&uh(s,l),u=u.createElement("link"),xe(u);var g=u;g._p=new Promise(function(A,z){g.onload=A,g.onerror=z}),Un(u,"link",s),a.instance=u}t.stylesheets===null&&(t.stylesheets=new Map),t.stylesheets.set(a,n),(n=a.state.preload)&&(a.state.loading&3)===0&&(t.count++,a=ul.bind(t),n.addEventListener("load",a),n.addEventListener("error",a))}}var Jc=0;function YM(t,n){return t.stylesheets&&t.count===0&&tu(t,t.stylesheets),0<t.count||0<t.imgCount?function(a){var s=setTimeout(function(){if(t.stylesheets&&tu(t,t.stylesheets),t.unsuspend){var u=t.unsuspend;t.unsuspend=null,u()}},6e4+n);0<t.imgBytes&&Jc===0&&(Jc=62500*fM());var l=setTimeout(function(){if(t.waitingForImages=!1,t.count===0&&(t.stylesheets&&tu(t,t.stylesheets),t.unsuspend)){var u=t.unsuspend;t.unsuspend=null,u()}},(t.imgBytes>Jc?50:800)+n);return t.unsuspend=a,function(){t.unsuspend=null,clearTimeout(s),clearTimeout(l)}}:null}function f_(t){if(t.count===0&&(t.imgCount===0||!t.waitingForImages)){if(t.stylesheets)tu(t,t.stylesheets);else if(t.unsuspend){var n=t.unsuspend;t.unsuspend=null,n()}}}function ul(){this.count--,f_(this)}function jM(){this.imgCount--,f_(this)}var $c=null;function tu(t,n){t.stylesheets=null,t.unsuspend!==null&&(t.count++,$c=new Map,n.forEach(ZM,t),$c=null,ul.call(t))}function ZM(t,n){if(!(n.state.loading&4)){var a=$c.get(t);if(a)var s=a.get(null);else{a=new Map,$c.set(t,a);for(var l=t.querySelectorAll("link[data-precedence],style[data-precedence]"),u=0;u<l.length;u++){var g=l[u];(g.nodeName==="LINK"||g.getAttribute("media")!=="not all")&&(a.set(g.dataset.precedence,g),s=g)}s&&a.set(null,s)}l=n.instance,g=l.getAttribute("data-precedence"),u=a.get(g)||s,u===s&&a.set(null,l),a.set(g,l),this.count++,s=ul.bind(this),l.addEventListener("load",s),l.addEventListener("error",s),u?u.parentNode.insertBefore(l,u.nextSibling):(t=t.nodeType===9?t.head:t,t.insertBefore(l,t.firstChild)),n.state.loading|=4}}var kr={$$typeof:$,Provider:null,Consumer:null,_currentValue:Fe,_currentValue2:Fe,_threadCount:0};function KM(t,n,a,s,l,u,g,A,z){this.tag=1,this.containerInfo=t,this.pingCache=this.current=this.pendingChildren=null,this.timeoutHandle=-1,this.callbackNode=this.next=this.pendingContext=this.context=this.cancelPendingCommit=null,this.callbackPriority=0,this.expirationTimes=rr(-1),this.entangledLanes=this.shellSuspendCounter=this.errorRecoveryDisabledLanes=this.expiredLanes=this.warmLanes=this.pingedLanes=this.suspendedLanes=this.pendingLanes=0,this.entanglements=rr(0),this.hiddenUpdates=rr(null),this.identifierPrefix=s,this.onUncaughtError=l,this.onCaughtError=u,this.onRecoverableError=g,this.pooledCache=null,this.pooledCacheLanes=0,this.formState=z,this.transitionTypes=null,this.incompleteTransitions=new Map}function d_(t,n,a,s,l,u,g,A,z,it,ht,Et){return t=new KM(t,n,a,g,z,it,ht,Et,A),n=1,u===!0&&(n|=24),u=jn(3,null,null,n),t.current=u,u.stateNode=t,n=Rf(),n.refCount++,t.pooledCache=n,n.refCount++,u.memoizedState={element:s,isDehydrated:a,cache:n},Df(u),t}function h_(t){return t?(t=mr,t):mr}function p_(t,n,a,s,l,u){l=h_(l),s.context===null?s.context=l:s.pendingContext=l,s=Ka(n),s.payload={element:a},u=u===void 0?null:u,u!==null&&(s.callback=u),a=Qa(t,s,n),a!==null&&(Jn(a,t,n),Go(a,t,n))}function m_(t,n){if(t=t.memoizedState,t!==null&&t.dehydrated!==null){var a=t.retryLane;t.retryLane=a!==0&&a<n?a:n}}function hh(t,n){m_(t,n),(t=t.alternate)&&m_(t,n)}function g_(t){if(t.tag===13||t.tag===31){var n=ws(t,67108864);n!==null&&Jn(n,t,67108864),hh(t,67108864)}}function v_(t){if(t.tag===13||t.tag===31){var n=ui();n=bo(n);var a=ws(t,n);a!==null&&Jn(a,t,n),hh(t,n)}}var Xr=!0;function QM(t,n,a,s){var l=xt.T;xt.T=null;var u=Ct.p;try{Ct.p=2,ph(t,n,a,s)}finally{Ct.p=u,xt.T=l}}function JM(t,n,a,s){var l=xt.T;xt.T=null;var u=Ct.p;try{Ct.p=8,ph(t,n,a,s)}finally{Ct.p=u,xt.T=l}}function ph(t,n,a,s){if(Xr){var l=mh(s);if(l===null)Zd(t,n,s,eu,a),x_(t,s);else if(tE(l,t,n,a,s))s.stopPropagation();else if(x_(t,s),n&4&&-1<$M.indexOf(t)){for(;l!==null;){var u=fe(l);if(u!==null)switch(u.tag){case 3:if(u=u.stateNode,u.current.memoizedState.isDehydrated){var g=pa(u.pendingLanes);if(g!==0){var A=u;for(A.pendingLanes|=2,A.entangledLanes|=2;g;){var z=1<<31-ue(g);A.entanglements[1]|=z,g&=~z}ia(u),(Ge&6)===0&&(Hc=kt()+500,il(0))}}break;case 31:case 13:A=ws(u,2),A!==null&&Jn(A,u,2),kc(),hh(u,2)}if(u=mh(s),u===null&&Zd(t,n,s,eu,a),u===l)break;l=u}l!==null&&s.stopPropagation()}else Zd(t,n,s,null,a)}}function mh(t){return t=$u(t),gh(t)}var eu=null;function gh(t){if(eu=null,t=se(t),t!==null){var n=f(t);if(n===null)t=null;else{var a=n.tag;if(a===13){if(t=d(n),t!==null)return t;t=null}else if(a===31){if(t=h(n),t!==null)return t;t=null}else if(a===3){if(n.stateNode.current.memoizedState.isDehydrated)return n.tag===3?n.stateNode.containerInfo:null;t=null}else n!==t&&(t=null)}}return eu=t,null}function __(t){switch(t){case"beforetoggle":case"cancel":case"click":case"close":case"contextmenu":case"copy":case"cut":case"auxclick":case"dblclick":case"dragend":case"dragstart":case"drop":case"focusin":case"focusout":case"input":case"invalid":case"keydown":case"keypress":case"keyup":case"mousedown":case"mouseup":case"paste":case"pause":case"play":case"pointercancel":case"pointerdown":case"pointerup":case"ratechange":case"reset":case"seeked":case"submit":case"toggle":case"touchcancel":case"touchend":case"touchstart":case"volumechange":case"change":case"selectionchange":case"textInput":case"compositionstart":case"compositionend":case"compositionupdate":case"beforeblur":case"afterblur":case"beforeinput":case"blur":case"fullscreenchange":case"fullscreenerror":case"focus":case"hashchange":case"popstate":case"select":case"selectstart":return 2;case"drag":case"dragenter":case"dragexit":case"dragleave":case"dragover":case"mousemove":case"mouseout":case"mouseover":case"pointermove":case"pointerout":case"pointerover":case"resize":case"scroll":case"touchmove":case"wheel":case"mouseenter":case"mouseleave":case"pointerenter":case"pointerleave":return 8;case"message":switch(ie()){case ce:return 2;case K:return 8;case Ut:case bt:return 32;case Lt:return 268435456;default:return 32}default:return 32}}var vh=!1,cs=null,us=null,fs=null,fl=new Map,dl=new Map,ds=[],$M="mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset".split(" ");function x_(t,n){switch(t){case"focusin":case"focusout":cs=null;break;case"dragenter":case"dragleave":us=null;break;case"mouseover":case"mouseout":fs=null;break;case"pointerover":case"pointerout":fl.delete(n.pointerId);break;case"gotpointercapture":case"lostpointercapture":dl.delete(n.pointerId)}}function hl(t,n,a,s,l,u){return t===null||t.nativeEvent!==u?(t={blockedOn:n,domEventName:a,eventSystemFlags:s,nativeEvent:u,targetContainers:[l]},n!==null&&(n=fe(n),n!==null&&g_(n)),t):(t.eventSystemFlags|=s,n=t.targetContainers,l!==null&&n.indexOf(l)===-1&&n.push(l),t)}function tE(t,n,a,s,l){switch(n){case"focusin":return cs=hl(cs,t,n,a,s,l),!0;case"dragenter":return us=hl(us,t,n,a,s,l),!0;case"mouseover":return fs=hl(fs,t,n,a,s,l),!0;case"pointerover":var u=l.pointerId;return fl.set(u,hl(fl.get(u)||null,t,n,a,s,l)),!0;case"gotpointercapture":return u=l.pointerId,dl.set(u,hl(dl.get(u)||null,t,n,a,s,l)),!0}return!1}function S_(t){var n=se(t.target);if(n!==null){var a=f(n);if(a!==null){if(n=a.tag,n===13){if(n=d(a),n!==null){t.blockedOn=n,Gl(t.priority,function(){v_(a)});return}}else if(n===31){if(n=h(a),n!==null){t.blockedOn=n,Gl(t.priority,function(){v_(a)});return}}else if(n===3&&a.stateNode.current.memoizedState.isDehydrated){t.blockedOn=a.tag===3?a.stateNode.containerInfo:null;return}}}t.blockedOn=null}function nu(t){if(t.blockedOn!==null)return!1;for(var n=t.targetContainers;0<n.length;){var a=mh(t.nativeEvent);if(a===null){a=t.nativeEvent;var s=new a.constructor(a.type,a);Ju=s,a.target.dispatchEvent(s),Ju=null}else return n=fe(a),n!==null&&g_(n),t.blockedOn=a,!1;n.shift()}return!0}function y_(t,n,a){nu(t)&&a.delete(n)}function eE(){vh=!1,cs!==null&&nu(cs)&&(cs=null),us!==null&&nu(us)&&(us=null),fs!==null&&nu(fs)&&(fs=null),fl.forEach(y_),dl.forEach(y_)}function iu(t,n){t.blockedOn===n&&(t.blockedOn=null,vh||(vh=!0,o.unstable_scheduleCallback(o.unstable_NormalPriority,eE)))}var au=null;function M_(t){au!==t&&(au=t,o.unstable_scheduleCallback(o.unstable_NormalPriority,function(){au===t&&(au=null);for(var n=0;n<t.length;n+=3){var a=t[n],s=t[n+1],l=t[n+2];if(typeof s!="function"){if(gh(s||a)===null)continue;break}var u=fe(a);u!==null&&(t.splice(n,3),n-=3,Jf(u,{pending:!0,data:l,method:a.method,action:s},s,l))}}))}function Wr(t){function n(z){return iu(z,t)}cs!==null&&iu(cs,t),us!==null&&iu(us,t),fs!==null&&iu(fs,t),fl.forEach(n),dl.forEach(n);for(var a=0;a<ds.length;a++){var s=ds[a];s.blockedOn===t&&(s.blockedOn=null)}for(;0<ds.length&&(a=ds[0],a.blockedOn===null);)S_(a),a.blockedOn===null&&ds.shift();if(a=(t.ownerDocument||t).$$reactFormReplay,a!=null)for(s=0;s<a.length;s+=3){var l=a[s],u=a[s+1],g=l[V]||null;if(typeof u=="function")g||M_(a);else if(g){var A=null;if(u&&u.hasAttribute("formAction")){if(l=u,g=u[V]||null)A=g.formAction;else if(gh(l)!==null)continue}else A=g.action;typeof A=="function"?a[s+1]=A:(a.splice(s,3),s-=3),M_(a)}}}function E_(){function t(u){u.canIntercept&&u.info==="react-transition"&&u.intercept({handler:function(){return new Promise(function(g){return l=g})},focusReset:"manual",scroll:"manual"})}function n(){l!==null&&(l(),l=null),s||setTimeout(a,20)}function a(){if(!s&&!navigation.transition){var u=navigation.currentEntry;u&&u.url!=null&&navigation.navigate(u.url,{state:u.getState(),info:"react-transition",history:"replace"})}}if(typeof navigation=="object"){var s=!1,l=null;return navigation.addEventListener("navigate",t),navigation.addEventListener("navigatesuccess",n),navigation.addEventListener("navigateerror",n),setTimeout(a,100),function(){s=!0,navigation.removeEventListener("navigate",t),navigation.removeEventListener("navigatesuccess",n),navigation.removeEventListener("navigateerror",n),l!==null&&(l(),l=null)}}}function _h(t){this._internalRoot=t}su.prototype.render=_h.prototype.render=function(t){var n=this._internalRoot;if(n===null)throw Error(r(409));var a=n.current,s=ui();p_(a,s,t,n,null,null)},su.prototype.unmount=_h.prototype.unmount=function(){var t=this._internalRoot;if(t!==null){this._internalRoot=null;var n=t.containerInfo;p_(t.current,2,null,t,null,null),kc(),n[mt]=null}};function su(t){this._internalRoot=t}su.prototype.unstable_scheduleHydration=function(t){if(t){var n=Hl();t={blockedOn:null,target:t,priority:n};for(var a=0;a<ds.length&&n!==0&&n<ds[a].priority;a++);ds.splice(a,0,t),a===0&&S_(t)}};var b_=e.version;if(b_!=="19.3.0")throw Error(r(527,b_,"19.3.0"));Ct.findDOMNode=function(t){var n=t._reactInternals;if(n===void 0)throw typeof t.render=="function"?Error(r(188)):(t=Object.keys(t).join(","),Error(r(268,t)));return t=p(n),t=t!==null?x(t):null,t=t===null?null:t.stateNode,t};var nE={bundleType:0,version:"19.3.0",rendererPackageName:"react-dom",currentDispatcherRef:xt,reconcilerVersion:"19.3.0"};if(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__<"u"){var ru=__REACT_DEVTOOLS_GLOBAL_HOOK__;if(!ru.isDisabled&&ru.supportsFiber)try{$t=ru.inject(nE),Gt=ru}catch{}}return ml.createRoot=function(t,n){if(!c(t))throw Error(r(299));var a=!1,s="",l=h0,u=p0,g=m0;return n!=null&&(n.unstable_strictMode===!0&&(a=!0),n.identifierPrefix!==void 0&&(s=n.identifierPrefix),n.onUncaughtError!==void 0&&(l=n.onUncaughtError),n.onCaughtError!==void 0&&(u=n.onCaughtError),n.onRecoverableError!==void 0&&(g=n.onRecoverableError)),n=d_(t,1,!1,null,null,a,s,null,l,u,g,E_),t[mt]=n.current,jd(t),new _h(n)},ml.hydrateRoot=function(t,n,a){if(!c(t))throw Error(r(299));var s=!1,l="",u=h0,g=p0,A=m0,z=null;return a!=null&&(a.unstable_strictMode===!0&&(s=!0),a.identifierPrefix!==void 0&&(l=a.identifierPrefix),a.onUncaughtError!==void 0&&(u=a.onUncaughtError),a.onCaughtError!==void 0&&(g=a.onCaughtError),a.onRecoverableError!==void 0&&(A=a.onRecoverableError),a.formState!==void 0&&(z=a.formState)),n=d_(t,1,!0,n,a??null,s,l,z,u,g,A,E_),n.context=h_(null),a=n.current,s=ui(),s=bo(s),l=Ka(s),l.callback=null,Qa(a,l,s),a=s,n.current.lanes=a,Wi(n,a),ia(n),t[mt]=n.current,jd(t),new su(n)},ml.version="19.3.0",ml}var O_;function dE(){if(O_)return yh.exports;O_=1;function o(){if(!(typeof __REACT_DEVTOOLS_GLOBAL_HOOK__>"u"||typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE!="function"))try{__REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(o)}catch(e){console.error(e)}}return o(),yh.exports=fE(),yh.exports}var hE=dE();const P_=["en","zh","ja"],pE={zh:{meta:{title:"Yoyo — 本地智能体工作站",description:"一台本地的智能体工作站。在你指定的文件夹里改代码、审查改动、拍短剧。写之前先问你。新的干活方式要你亲自换成正在用的那一份。"},nav:{workstation:"工作站",video:"视频",model:"模型",trust:"信任",doors:"入口",download:"下载",menu:"菜单",close:"关闭菜单",themeToDark:"切换到深色",themeToLight:"切换到浅色"},hero:{title:`在你选的文件夹里，
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
変わるのは働き方。`,dek:"Yoyo は自前モデルを訓練しない。プロンプトを長くしても、賢くなったことにはならない。OpenAI 互換の API につなぐ。重みは元のサービスに残る。磨けるのはプロンプト、手順、スキル。結果は脇に置かれ、切り替えるのはあなた。",solver:"ソルバー",solverBody:"ターンの中で、モデルは文脈を読み、ツールを呼び、文章を書く。Stop はそのターンを止める。すでに書いたものは残る。Plan は提案だけ。書き込みと shell は閉じたまま。",shape:"形",shapeBody:"古いツール結果は短く切られる。文脈には予算がある。@file は一つのファイルをピンする。木全体はプロンプトに入らない。これはハーネスの形で、個性ではない。",evolve:"進化",evolveBody:"評価は staging へ。Evolve は canary へ。保持セットもホールドアウトも下がってはならない。安全失敗は常に止める。checkout したあとだけ、ポインタは active。",weightsStill:"重みは動かない",modelName:"gpt-4.1-mini",lineLive:"エラーは返す。握りつぶさない。",lineNext:"呼び出し元の名前も書く。",l1:"L1",l1Body:"プロンプト、playbook、スキル。Evolve は変えられる。Harbor を通す。",l2:"L2",l2Body:"署名済み WASM。未署名は active に入らない。ファイルシステムはない。",l3:"L3",l3Body:"ループと方針。変えられるのは人だけ。",l4:"L4",l4Body:"カーネル、評価器、鍵、アップデータ。エージェントも evolve も不可。",fp:"モデル指紋",fpBody:"refs/models/<指紋>/active。同じ重みでも、ハーネスはモデルごと。"},see:{title:"頼んだあと、変更は見える場所に残る。",dek:"ファイルをピンする。残すハンクを選ぶ。動画は専用フォルダ。",pin:"ファイルをピン",pinBody:"@ を打っただけでリポジトリ全体は入らない。",diff:"選んで残す",diffBody:"ハンク単位で適用。見つからなければ失敗する。",drama:"動画は会話",dramaBody:"章を貼る。話数を確認してから撮る。コードの作業場とは混ぜない。",folderAlt:"暗いオークの机の上の、閉じたクラフト紙のフォルダと真鍮のペン。"},trust:{title:"賢くなってよい。自分では昇格できない。",dek:"評価は下書きへ。評価が通ったら、あなたが checkout する。動くのはハーネスで、モデルの重みではない。鍵とアップデータには触れない。",staging:"下書き",stagingHint:"評価はここだけ",canary:"待機",canaryHint:"評価は通った。まだ checkout していない",active:"使用中",activeHint:"動かすのは人だけ",checkout:"昇格",done:"動かしたのはあなた。エージェントではない。"},doors:{title:"ひとつのループ。三つの入口。",dek:"デスクトップは窓とトレイ。ブラウザは本機。CLI は同じ home。",desktop:"デスクトップ",desktopBody:"ネイティブ窓、メニュー、トレイ、通知。インストーラはこれ。",browser:"ブラウザ",browserBody:"会話と評価は本機。トレイはない。",cli:"CLI",cliBody:"ターン、評価、進化、パック。",skills:"スキルは既定オフ",skillsBody:"パックは ~/.yoyo/packs/。ワークスペースで有効化。評価は読まない。",budget:"予算はハードストップ",budgetBody:"上限に達したら止まる。鍵はローカル保管庫。"},get:{title:"アカウントもクラウドもない。入れて走らせる。",dek:"フォルダを選ぶ。OpenAI 互換キーを貼る。タスクを打ち、@ でピンし、書き込みは承認。",premise:"キーは自分で持つ。ホストされたチャットが欲しいなら、このページは対象外。",version:"バージョン",steps:["OS のデスクトップアプリを入れる。","設定で作業フォルダを選ぶ。そこだけ触る。","設定に OpenAI 互換キーを貼る。","タスクを打つ。@ でピン。書き込みは承認。"],thisMachine:"このマシン",note:"Windows と Linux は未署名。SmartScreen は「詳細情報 → 実行」。macOS はアドホック署名で公証なし。ソースは Apache-2.0。"},pitch:{title:"実際にすること",dek:"クラウド上のもう一人ではない。このマシンでの一つのターン。フォルダもモデルも、書いてよいかも、決めるのはあなた。",cards:[{title:"選んだフォルダを直す",body:"ディレクトリを選び、タスクを打つ。読みながら直す。Plan は提案だけ。止めても、すでに書いたものは残る。"},{title:"聞いてから書き、実行する",body:"書き込みもコマンドも、スレッドで尋ねる。一度、この会話、このマシン、または拒否。"},{title:"変更は見えるところに残る",body:"キューは会話のそば。断片ごとに残す。ファイルがなければ、復元は失敗してはっきり言う。"},{title:"映像はリポジトリの外",body:"章を貼り、話数を確認してから撮る。作成、ドラマ、キャンバスは一つの殻。映像は別フォルダ。"},{title:"スキルは最初オフ",body:"パックは毎ターン入らない。このフォルダで有効化してから。評価は読まない。"},{title:"予算は停止",body:"上限に達したらターンは止まる。鍵はこのマシン。設定も実行中のアプリも、自分では替えられない。"}]},foot:{product:"製品",invariants:"不変条件",issues:"Issues",license:"Apache-2.0",stack:"Go · Wails"},demo:{agent:"Agent",review:"審査",video:"ビデオ",harbor:"Harbor",module:"モジュール",chats:"チャット",threadLive:"Parser fix",threadLiveSub:"apply_patch · 待ち",threadIdle:"Harbor smoke",newChat:"新しい会話",hello:"こんにちは。",bubble:"parser のエラーを握りつぶさず返す",read:"読んだ",patch:"apply_patch · +24 −6",needsOk:"確認が必要",needsApproval:"承認が必要",reading:"読み取り",patching:"パッチ",worked:"4s 作業した",plan:"Plan",action:"write_file",path:"internal/runtime/loop.go",allowOnce:"一度許可",allowSession:"この会話",allowAlways:"常に許可",reject:"拒否",apply:"適用",hunks:"ハンク",composer:"@file:loop.go の parser を直す",createTab:"作成",canvasTab:"キャンバス",browserBar:"localhost · preview",browserBody:"パーサーはエラーを返すようになった。",trailRead:"Read",trailPatch:"apply_patch",queue:"キュー",queueItem:"loop.go · +24 −6",statusFolder:"~/work/parser",local:"ローカル",kept:"残した",keptNote:"選んだハンクはフォルダに残る。選ばなかったものは書き込まれない。",createBody:"一本を説明する。画像、音声、動画のアダプタは自分のもの。",canvasHint:"ショットは一つの机に並ぶ。フォルダはコードの作業場とは別。",dramaPrompt:"この章を三話で撮る。計画を確認してから撮る。",ep1:"到着",ep2:"ランプ",ep3:"出発",shotCap:"02 · ランプ",createHint:"樫の机のランプ、黄昏。"}}};function mE(){try{const e=new URLSearchParams(window.location.search).get("lang");if(e&&P_.includes(e))return e;const i=localStorage.getItem("yoyo-site-lang");if(i&&P_.includes(i))return i}catch{}const o=(typeof navigator<"u"?navigator.language:"en").toLowerCase();return o.startsWith("zh")?"zh":o.startsWith("ja")?"ja":"en"}function gE(o){return o==="zh"?"zh-Hans":o}const Qp="Shenchangxin/yoyo",Th=`https://github.com/${Qp}/releases/latest`,Nl="0.4.2";function gl(o,e){return`https://github.com/${Qp}/releases/download/v${o}/${e}`}function vE(o){return o?o<1024*1024?`${Math.round(o/1024)} KB`:`${(o/(1024*1024)).toFixed(0)} MB`:""}function rp(o){return[{os:"windows",label:"Windows",name:`Yoyo-${o}-windows-x64-setup.exe`,href:gl(o,`Yoyo-${o}-windows-x64-setup.exe`),size:""},{os:"mac",label:"macOS",name:`Yoyo-${o}-darwin-universal.dmg`,href:gl(o,`Yoyo-${o}-darwin-universal.dmg`),size:""},{os:"linux",label:"Linux",name:`Yoyo-${o}-linux-amd64.AppImage`,href:gl(o,`Yoyo-${o}-linux-amd64.AppImage`),size:""},{os:"deb",label:"Debian",name:`Yoyo-${o}-linux-amd64.deb`,href:gl(o,`Yoyo-${o}-linux-amd64.deb`),size:""},{os:"rpm",label:"RPM",name:`Yoyo-${o}-linux-amd64.rpm`,href:gl(o,`Yoyo-${o}-linux-amd64.rpm`),size:""}]}function _E(){const o=navigator.userAgent;return/Mac OS X|Macintosh/.test(o)?"mac":/Linux/.test(o)&&!/Android/.test(o)?"linux":"windows"}function vl(o,e){for(const i of e){const r=o.find(c=>i.test(c.name));if(r)return r}return null}async function xE(){const o=rp(Nl);try{const e=await fetch(`https://api.github.com/repos/${Qp}/releases/latest`);if(!e.ok)throw new Error("release");const i=await e.json(),r=String(i.tag_name||Nl).replace(/^v/,""),c=i.assets||[],f=rp(r),d=vl(c,[/windows-x64-setup\.exe$/i]),h=vl(c,[/darwin-universal\.dmg$/i,/\.dmg$/i]),m=vl(c,[/linux-amd64\.AppImage$/i,/\.AppImage$/i]),p=vl(c,[/\.deb$/i]),x=vl(c,[/\.rpm$/i]),v=[d,h,m,p,x];return{version:r,assets:f.map((_,M)=>{const R=v[M];return R?{..._,name:R.name,href:R.browser_download_url,size:vE(R.size)}:_})}}catch{return{version:Nl,assets:o}}}/**
 * @license
 * Copyright 2010-2026 Three.js Authors
 * SPDX-License-Identifier: MIT
 */const Jp="186",SE=0,I_=1,yE=2,Nu=1,Vx=2,wl=3,$s=0,qn=1,Bi=2,Ia=0,Dl=1,B_=2,z_=3,F_=4,ME=5,oo=100,EE=101,bE=102,TE=103,AE=104,RE=200,wE=201,CE=202,NE=203,kx=204,Xx=205,DE=206,UE=207,LE=208,OE=209,PE=210,IE=211,BE=212,zE=213,FE=214,op=0,lp=1,cp=2,Ul=3,up=4,fp=5,dp=6,hp=7,$p=0,HE=1,GE=2,ca=0,Wx=1,qx=2,Yx=3,tm=4,jx=5,Zx=6,Kx=7,Qx=300,tr=301,po=302,Ah=303,Rh=304,ku=306,pp=1e3,Pa=1001,mp=1002,On=1003,VE=1004,ou=1005,zn=1006,wh=1007,Qs=1008,pi=1009,Jx=1010,$x=1011,Ll=1012,em=1013,fa=1014,Hi=1015,da=1016,nm=1017,im=1018,Ol=1020,tS=35902,eS=35899,nS=1021,iS=1022,Gi=1023,za=1026,Js=1027,am=1028,sm=1029,er=1030,rm=1031,om=1033,Du=33776,Uu=33777,Lu=33778,Ou=33779,gp=35840,vp=35841,_p=35842,xp=35843,Sp=36196,yp=37492,Mp=37496,Ep=37488,bp=37489,Iu=37490,Tp=37491,Ap=37808,Rp=37809,wp=37810,Cp=37811,Np=37812,Dp=37813,Up=37814,Lp=37815,Op=37816,Pp=37817,Ip=37818,Bp=37819,zp=37820,Fp=37821,Hp=36492,Gp=36494,Vp=36495,kp=36283,Xp=36284,Bu=36285,Wp=36286,kE=3200,zu=0,XE=1,ys="",ti="srgb",Fu="srgb-linear",Hu="linear",Ye="srgb",Ch=7680,WE=519,qE=512,YE=513,jE=514,lm=515,ZE=516,KE=517,cm=518,QE=519,JE=35044,H_="300 es",la=2e3,Pl=2001;function $E(o){for(let e=o.length-1;e>=0;--e)if(o[e]>=65535)return!0;return!1}function Gu(o){return document.createElementNS("http://www.w3.org/1999/xhtml",o)}function tb(){const o=Gu("canvas");return o.style.display="block",o}const G_={};function V_(...o){const e="THREE."+o.shift();console.log(e,...o)}function aS(o){const e=o[0];if(typeof e=="string"&&e.startsWith("TSL:")){const i=o[1];i&&i.isStackTrace?o[0]+=" "+i.getLocation():o[1]='Stack trace not available. Enable "THREE.Node.captureStackTrace" to capture stack traces.'}return o}function oe(...o){o=aS(o);const e="THREE."+o.shift();{const i=o[0];i&&i.isStackTrace?console.warn(i.getError(e)):console.warn(e,...o)}}function ze(...o){o=aS(o);const e="THREE."+o.shift();{const i=o[0];i&&i.isStackTrace?console.error(i.getError(e)):console.error(e,...o)}}function fo(...o){const e=o.join(" ");e in G_||(G_[e]=!0,oe(...o))}function eb(o,e,i){return new Promise(function(r,c){function f(){switch(o.clientWaitSync(e,o.SYNC_FLUSH_COMMANDS_BIT,0)){case o.WAIT_FAILED:c();break;case o.TIMEOUT_EXPIRED:setTimeout(f,i);break;default:r()}}setTimeout(f,i)})}const nb={[op]:lp,[cp]:dp,[up]:hp,[Ul]:fp,[lp]:op,[dp]:cp,[hp]:up,[fp]:Ul};class nr{addEventListener(e,i){this._listeners===void 0&&(this._listeners={});const r=this._listeners;r[e]===void 0&&(r[e]=[]),r[e].indexOf(i)===-1&&r[e].push(i)}hasEventListener(e,i){const r=this._listeners;return r===void 0?!1:r[e]!==void 0&&r[e].indexOf(i)!==-1}removeEventListener(e,i){const r=this._listeners;if(r===void 0)return;const c=r[e];if(c!==void 0){const f=c.indexOf(i);f!==-1&&c.splice(f,1)}}dispatchEvent(e){const i=this._listeners;if(i===void 0)return;const r=i[e.type];if(r!==void 0){e.target=this;const c=r.slice(0);for(let f=0,d=c.length;f<d;f++)c[f].call(this,e);e.target=null}}}const In=["00","01","02","03","04","05","06","07","08","09","0a","0b","0c","0d","0e","0f","10","11","12","13","14","15","16","17","18","19","1a","1b","1c","1d","1e","1f","20","21","22","23","24","25","26","27","28","29","2a","2b","2c","2d","2e","2f","30","31","32","33","34","35","36","37","38","39","3a","3b","3c","3d","3e","3f","40","41","42","43","44","45","46","47","48","49","4a","4b","4c","4d","4e","4f","50","51","52","53","54","55","56","57","58","59","5a","5b","5c","5d","5e","5f","60","61","62","63","64","65","66","67","68","69","6a","6b","6c","6d","6e","6f","70","71","72","73","74","75","76","77","78","79","7a","7b","7c","7d","7e","7f","80","81","82","83","84","85","86","87","88","89","8a","8b","8c","8d","8e","8f","90","91","92","93","94","95","96","97","98","99","9a","9b","9c","9d","9e","9f","a0","a1","a2","a3","a4","a5","a6","a7","a8","a9","aa","ab","ac","ad","ae","af","b0","b1","b2","b3","b4","b5","b6","b7","b8","b9","ba","bb","bc","bd","be","bf","c0","c1","c2","c3","c4","c5","c6","c7","c8","c9","ca","cb","cc","cd","ce","cf","d0","d1","d2","d3","d4","d5","d6","d7","d8","d9","da","db","dc","dd","de","df","e0","e1","e2","e3","e4","e5","e6","e7","e8","e9","ea","eb","ec","ed","ee","ef","f0","f1","f2","f3","f4","f5","f6","f7","f8","f9","fa","fb","fc","fd","fe","ff"],Nh=Math.PI/180,qp=180/Math.PI;function Bl(){const o=Math.random()*4294967295|0,e=Math.random()*4294967295|0,i=Math.random()*4294967295|0,r=Math.random()*4294967295|0;return(In[o&255]+In[o>>8&255]+In[o>>16&255]+In[o>>24&255]+"-"+In[e&255]+In[e>>8&255]+"-"+In[e>>16&15|64]+In[e>>24&255]+"-"+In[i&63|128]+In[i>>8&255]+"-"+In[i>>16&255]+In[i>>24&255]+In[r&255]+In[r>>8&255]+In[r>>16&255]+In[r>>24&255]).toLowerCase()}function De(o,e,i){return Math.max(e,Math.min(i,o))}function ib(o,e){return(o%e+e)%e}function Dh(o,e,i){return(1-i)*o+i*e}function _l(o,e){switch(e.constructor){case Float32Array:return o;case Uint32Array:return o/4294967295;case Uint16Array:return o/65535;case Uint8Array:case Uint8ClampedArray:return o/255;case Int32Array:return Math.max(o/2147483647,-1);case Int16Array:return Math.max(o/32767,-1);case Int8Array:return Math.max(o/127,-1);default:throw new Error("THREE.MathUtils: Invalid component type.")}}function $n(o,e){switch(e.constructor){case Float32Array:return o;case Uint32Array:return Math.round(o*4294967295);case Uint16Array:return Math.round(o*65535);case Uint8Array:case Uint8ClampedArray:return Math.round(o*255);case Int32Array:return Math.round(o*2147483647);case Int16Array:return Math.round(o*32767);case Int8Array:return Math.round(o*127);default:throw new Error("THREE.MathUtils: Invalid component type.")}}const vm=class vm{constructor(e=0,i=0){this.x=e,this.y=i}get width(){return this.x}set width(e){this.x=e}get height(){return this.y}set height(e){this.y=e}set(e,i){return this.x=e,this.y=i,this}setScalar(e){return this.x=e,this.y=e,this}setX(e){return this.x=e,this}setY(e){return this.y=e,this}setComponent(e,i){switch(e){case 0:this.x=i;break;case 1:this.y=i;break;default:throw new Error("THREE.Vector2: index is out of range: "+e)}return this}getComponent(e){switch(e){case 0:return this.x;case 1:return this.y;default:throw new Error("THREE.Vector2: index is out of range: "+e)}}clone(){return new this.constructor(this.x,this.y)}copy(e){return this.x=e.x,this.y=e.y,this}add(e){return this.x+=e.x,this.y+=e.y,this}addScalar(e){return this.x+=e,this.y+=e,this}addVectors(e,i){return this.x=e.x+i.x,this.y=e.y+i.y,this}addScaledVector(e,i){return this.x+=e.x*i,this.y+=e.y*i,this}sub(e){return this.x-=e.x,this.y-=e.y,this}subScalar(e){return this.x-=e,this.y-=e,this}subVectors(e,i){return this.x=e.x-i.x,this.y=e.y-i.y,this}multiply(e){return this.x*=e.x,this.y*=e.y,this}multiplyScalar(e){return this.x*=e,this.y*=e,this}divide(e){return this.x/=e.x,this.y/=e.y,this}divideScalar(e){return this.multiplyScalar(1/e)}applyMatrix3(e){const i=this.x,r=this.y,c=e.elements;return this.x=c[0]*i+c[3]*r+c[6],this.y=c[1]*i+c[4]*r+c[7],this}min(e){return this.x=Math.min(this.x,e.x),this.y=Math.min(this.y,e.y),this}max(e){return this.x=Math.max(this.x,e.x),this.y=Math.max(this.y,e.y),this}clamp(e,i){return this.x=De(this.x,e.x,i.x),this.y=De(this.y,e.y,i.y),this}clampScalar(e,i){return this.x=De(this.x,e,i),this.y=De(this.y,e,i),this}clampLength(e,i){const r=this.length();return this.divideScalar(r||1).multiplyScalar(De(r,e,i))}floor(){return this.x=Math.floor(this.x),this.y=Math.floor(this.y),this}ceil(){return this.x=Math.ceil(this.x),this.y=Math.ceil(this.y),this}round(){return this.x=Math.round(this.x),this.y=Math.round(this.y),this}roundToZero(){return this.x=Math.trunc(this.x),this.y=Math.trunc(this.y),this}negate(){return this.x=-this.x,this.y=-this.y,this}dot(e){return this.x*e.x+this.y*e.y}cross(e){return this.x*e.y-this.y*e.x}lengthSq(){return this.x*this.x+this.y*this.y}length(){return Math.sqrt(this.x*this.x+this.y*this.y)}manhattanLength(){return Math.abs(this.x)+Math.abs(this.y)}normalize(){return this.divideScalar(this.length()||1)}angle(){return Math.atan2(-this.y,-this.x)+Math.PI}angleTo(e){const i=Math.sqrt(this.lengthSq()*e.lengthSq());if(i===0)return Math.PI/2;const r=this.dot(e)/i;return Math.acos(De(r,-1,1))}distanceTo(e){return Math.sqrt(this.distanceToSquared(e))}distanceToSquared(e){const i=this.x-e.x,r=this.y-e.y;return i*i+r*r}manhattanDistanceTo(e){return Math.abs(this.x-e.x)+Math.abs(this.y-e.y)}setLength(e){return this.normalize().multiplyScalar(e)}lerp(e,i){return this.x+=(e.x-this.x)*i,this.y+=(e.y-this.y)*i,this}lerpVectors(e,i,r){return this.x=e.x+(i.x-e.x)*r,this.y=e.y+(i.y-e.y)*r,this}equals(e){return e.x===this.x&&e.y===this.y}fromArray(e,i=0){return this.x=e[i],this.y=e[i+1],this}toArray(e=[],i=0){return e[i]=this.x,e[i+1]=this.y,e}fromBufferAttribute(e,i){return this.x=e.getX(i),this.y=e.getY(i),this}rotateAround(e,i){const r=Math.cos(i),c=Math.sin(i),f=this.x-e.x,d=this.y-e.y;return this.x=f*r-d*c+e.x,this.y=f*c+d*r+e.y,this}random(){return this.x=Math.random(),this.y=Math.random(),this}*[Symbol.iterator](){yield this.x,yield this.y}};vm.prototype.isVector2=!0;let Re=vm;class _o{constructor(e=0,i=0,r=0,c=1){this.isQuaternion=!0,this._x=e,this._y=i,this._z=r,this._w=c}static slerpFlat(e,i,r,c,f,d,h){let m=r[c+0],p=r[c+1],x=r[c+2],v=r[c+3],_=f[d+0],M=f[d+1],R=f[d+2],N=f[d+3];if(v!==N||m!==_||p!==M||x!==R){let y=m*_+p*M+x*R+v*N;y<0&&(_=-_,M=-M,R=-R,N=-N,y=-y);let S=1-h;if(y<.9995){const O=Math.acos(y),k=Math.sin(O);S=Math.sin(S*O)/k,h=Math.sin(h*O)/k,m=m*S+_*h,p=p*S+M*h,x=x*S+R*h,v=v*S+N*h}else{m=m*S+_*h,p=p*S+M*h,x=x*S+R*h,v=v*S+N*h;const O=1/Math.sqrt(m*m+p*p+x*x+v*v);m*=O,p*=O,x*=O,v*=O}}e[i]=m,e[i+1]=p,e[i+2]=x,e[i+3]=v}static multiplyQuaternionsFlat(e,i,r,c,f,d){const h=r[c],m=r[c+1],p=r[c+2],x=r[c+3],v=f[d],_=f[d+1],M=f[d+2],R=f[d+3];return e[i]=h*R+x*v+m*M-p*_,e[i+1]=m*R+x*_+p*v-h*M,e[i+2]=p*R+x*M+h*_-m*v,e[i+3]=x*R-h*v-m*_-p*M,e}get x(){return this._x}set x(e){this._x=e,this._onChangeCallback()}get y(){return this._y}set y(e){this._y=e,this._onChangeCallback()}get z(){return this._z}set z(e){this._z=e,this._onChangeCallback()}get w(){return this._w}set w(e){this._w=e,this._onChangeCallback()}set(e,i,r,c){return this._x=e,this._y=i,this._z=r,this._w=c,this._onChangeCallback(),this}clone(){return new this.constructor(this._x,this._y,this._z,this._w)}copy(e){return this._x=e.x,this._y=e.y,this._z=e.z,this._w=e.w,this._onChangeCallback(),this}setFromEuler(e,i=!0){const r=e._x,c=e._y,f=e._z,d=e._order,h=Math.cos,m=Math.sin,p=h(r/2),x=h(c/2),v=h(f/2),_=m(r/2),M=m(c/2),R=m(f/2);switch(d){case"XYZ":this._x=_*x*v+p*M*R,this._y=p*M*v-_*x*R,this._z=p*x*R+_*M*v,this._w=p*x*v-_*M*R;break;case"YXZ":this._x=_*x*v+p*M*R,this._y=p*M*v-_*x*R,this._z=p*x*R-_*M*v,this._w=p*x*v+_*M*R;break;case"ZXY":this._x=_*x*v-p*M*R,this._y=p*M*v+_*x*R,this._z=p*x*R+_*M*v,this._w=p*x*v-_*M*R;break;case"ZYX":this._x=_*x*v-p*M*R,this._y=p*M*v+_*x*R,this._z=p*x*R-_*M*v,this._w=p*x*v+_*M*R;break;case"YZX":this._x=_*x*v+p*M*R,this._y=p*M*v+_*x*R,this._z=p*x*R-_*M*v,this._w=p*x*v-_*M*R;break;case"XZY":this._x=_*x*v-p*M*R,this._y=p*M*v-_*x*R,this._z=p*x*R+_*M*v,this._w=p*x*v+_*M*R;break;default:oe("Quaternion: .setFromEuler() encountered an unknown order: "+d)}return i===!0&&this._onChangeCallback(),this}setFromAxisAngle(e,i){const r=i/2,c=Math.sin(r);return this._x=e.x*c,this._y=e.y*c,this._z=e.z*c,this._w=Math.cos(r),this._onChangeCallback(),this}setFromRotationMatrix(e){const i=e.elements,r=i[0],c=i[4],f=i[8],d=i[1],h=i[5],m=i[9],p=i[2],x=i[6],v=i[10],_=r+h+v;if(_>0){const M=.5/Math.sqrt(_+1);this._w=.25/M,this._x=(x-m)*M,this._y=(f-p)*M,this._z=(d-c)*M}else if(r>h&&r>v){const M=2*Math.sqrt(1+r-h-v);this._w=(x-m)/M,this._x=.25*M,this._y=(c+d)/M,this._z=(f+p)/M}else if(h>v){const M=2*Math.sqrt(1+h-r-v);this._w=(f-p)/M,this._x=(c+d)/M,this._y=.25*M,this._z=(m+x)/M}else{const M=2*Math.sqrt(1+v-r-h);this._w=(d-c)/M,this._x=(f+p)/M,this._y=(m+x)/M,this._z=.25*M}return this._onChangeCallback(),this}setFromUnitVectors(e,i){let r=e.dot(i)+1;return r<1e-8?(r=0,Math.abs(e.x)>Math.abs(e.z)?(this._x=-e.y,this._y=e.x,this._z=0,this._w=r):(this._x=0,this._y=-e.z,this._z=e.y,this._w=r)):(this._x=e.y*i.z-e.z*i.y,this._y=e.z*i.x-e.x*i.z,this._z=e.x*i.y-e.y*i.x,this._w=r),this.normalize()}angleTo(e){return 2*Math.acos(Math.abs(De(this.dot(e),-1,1)))}rotateTowards(e,i){const r=this.angleTo(e);if(r===0)return this;const c=Math.min(1,i/r);return this.slerp(e,c),this}identity(){return this.set(0,0,0,1)}invert(){return this.conjugate()}conjugate(){return this._x*=-1,this._y*=-1,this._z*=-1,this._onChangeCallback(),this}dot(e){return this._x*e._x+this._y*e._y+this._z*e._z+this._w*e._w}lengthSq(){return this._x*this._x+this._y*this._y+this._z*this._z+this._w*this._w}length(){return Math.sqrt(this._x*this._x+this._y*this._y+this._z*this._z+this._w*this._w)}normalize(){let e=this.length();return e===0?(this._x=0,this._y=0,this._z=0,this._w=1):(e=1/e,this._x=this._x*e,this._y=this._y*e,this._z=this._z*e,this._w=this._w*e),this._onChangeCallback(),this}multiply(e){return this.multiplyQuaternions(this,e)}premultiply(e){return this.multiplyQuaternions(e,this)}multiplyQuaternions(e,i){const r=e._x,c=e._y,f=e._z,d=e._w,h=i._x,m=i._y,p=i._z,x=i._w;return this._x=r*x+d*h+c*p-f*m,this._y=c*x+d*m+f*h-r*p,this._z=f*x+d*p+r*m-c*h,this._w=d*x-r*h-c*m-f*p,this._onChangeCallback(),this}slerp(e,i){let r=e._x,c=e._y,f=e._z,d=e._w,h=this.dot(e);h<0&&(r=-r,c=-c,f=-f,d=-d,h=-h);let m=1-i;if(h<.9995){const p=Math.acos(h),x=Math.sin(p);m=Math.sin(m*p)/x,i=Math.sin(i*p)/x,this._x=this._x*m+r*i,this._y=this._y*m+c*i,this._z=this._z*m+f*i,this._w=this._w*m+d*i,this._onChangeCallback()}else this._x=this._x*m+r*i,this._y=this._y*m+c*i,this._z=this._z*m+f*i,this._w=this._w*m+d*i,this.normalize();return this}slerpQuaternions(e,i,r){return this.copy(e).slerp(i,r)}random(){const e=2*Math.PI*Math.random(),i=2*Math.PI*Math.random(),r=Math.random(),c=Math.sqrt(1-r),f=Math.sqrt(r);return this.set(c*Math.sin(e),c*Math.cos(e),f*Math.sin(i),f*Math.cos(i))}equals(e){return e._x===this._x&&e._y===this._y&&e._z===this._z&&e._w===this._w}fromArray(e,i=0){return this._x=e[i],this._y=e[i+1],this._z=e[i+2],this._w=e[i+3],this._onChangeCallback(),this}toArray(e=[],i=0){return e[i]=this._x,e[i+1]=this._y,e[i+2]=this._z,e[i+3]=this._w,e}fromBufferAttribute(e,i){return this._x=e.getX(i),this._y=e.getY(i),this._z=e.getZ(i),this._w=e.getW(i),this._onChangeCallback(),this}toJSON(){return this.toArray()}_onChange(e){return this._onChangeCallback=e,this}_onChangeCallback(){}*[Symbol.iterator](){yield this._x,yield this._y,yield this._z,yield this._w}}const _m=class _m{constructor(e=0,i=0,r=0){this.x=e,this.y=i,this.z=r}set(e,i,r){return r===void 0&&(r=this.z),this.x=e,this.y=i,this.z=r,this}setScalar(e){return this.x=e,this.y=e,this.z=e,this}setX(e){return this.x=e,this}setY(e){return this.y=e,this}setZ(e){return this.z=e,this}setComponent(e,i){switch(e){case 0:this.x=i;break;case 1:this.y=i;break;case 2:this.z=i;break;default:throw new Error("THREE.Vector3: index is out of range: "+e)}return this}getComponent(e){switch(e){case 0:return this.x;case 1:return this.y;case 2:return this.z;default:throw new Error("THREE.Vector3: index is out of range: "+e)}}clone(){return new this.constructor(this.x,this.y,this.z)}copy(e){return this.x=e.x,this.y=e.y,this.z=e.z,this}add(e){return this.x+=e.x,this.y+=e.y,this.z+=e.z,this}addScalar(e){return this.x+=e,this.y+=e,this.z+=e,this}addVectors(e,i){return this.x=e.x+i.x,this.y=e.y+i.y,this.z=e.z+i.z,this}addScaledVector(e,i){return this.x+=e.x*i,this.y+=e.y*i,this.z+=e.z*i,this}sub(e){return this.x-=e.x,this.y-=e.y,this.z-=e.z,this}subScalar(e){return this.x-=e,this.y-=e,this.z-=e,this}subVectors(e,i){return this.x=e.x-i.x,this.y=e.y-i.y,this.z=e.z-i.z,this}multiply(e){return this.x*=e.x,this.y*=e.y,this.z*=e.z,this}multiplyScalar(e){return this.x*=e,this.y*=e,this.z*=e,this}multiplyVectors(e,i){return this.x=e.x*i.x,this.y=e.y*i.y,this.z=e.z*i.z,this}applyEuler(e){return this.applyQuaternion(k_.setFromEuler(e))}applyAxisAngle(e,i){return this.applyQuaternion(k_.setFromAxisAngle(e,i))}applyMatrix3(e){const i=this.x,r=this.y,c=this.z,f=e.elements;return this.x=f[0]*i+f[3]*r+f[6]*c,this.y=f[1]*i+f[4]*r+f[7]*c,this.z=f[2]*i+f[5]*r+f[8]*c,this}applyNormalMatrix(e){return this.applyMatrix3(e).normalize()}applyMatrix4(e){const i=this.x,r=this.y,c=this.z,f=e.elements,d=1/(f[3]*i+f[7]*r+f[11]*c+f[15]);return this.x=(f[0]*i+f[4]*r+f[8]*c+f[12])*d,this.y=(f[1]*i+f[5]*r+f[9]*c+f[13])*d,this.z=(f[2]*i+f[6]*r+f[10]*c+f[14])*d,this}applyQuaternion(e){const i=this.x,r=this.y,c=this.z,f=e.x,d=e.y,h=e.z,m=e.w,p=2*(d*c-h*r),x=2*(h*i-f*c),v=2*(f*r-d*i);return this.x=i+m*p+d*v-h*x,this.y=r+m*x+h*p-f*v,this.z=c+m*v+f*x-d*p,this}project(e){return this.applyMatrix4(e.matrixWorldInverse).applyMatrix4(e.projectionMatrix)}unproject(e){return this.applyMatrix4(e.projectionMatrixInverse).applyMatrix4(e.matrixWorld)}transformDirection(e){const i=this.x,r=this.y,c=this.z,f=e.elements;return this.x=f[0]*i+f[4]*r+f[8]*c,this.y=f[1]*i+f[5]*r+f[9]*c,this.z=f[2]*i+f[6]*r+f[10]*c,this.normalize()}divide(e){return this.x/=e.x,this.y/=e.y,this.z/=e.z,this}divideScalar(e){return this.multiplyScalar(1/e)}min(e){return this.x=Math.min(this.x,e.x),this.y=Math.min(this.y,e.y),this.z=Math.min(this.z,e.z),this}max(e){return this.x=Math.max(this.x,e.x),this.y=Math.max(this.y,e.y),this.z=Math.max(this.z,e.z),this}clamp(e,i){return this.x=De(this.x,e.x,i.x),this.y=De(this.y,e.y,i.y),this.z=De(this.z,e.z,i.z),this}clampScalar(e,i){return this.x=De(this.x,e,i),this.y=De(this.y,e,i),this.z=De(this.z,e,i),this}clampLength(e,i){const r=this.length();return this.divideScalar(r||1).multiplyScalar(De(r,e,i))}floor(){return this.x=Math.floor(this.x),this.y=Math.floor(this.y),this.z=Math.floor(this.z),this}ceil(){return this.x=Math.ceil(this.x),this.y=Math.ceil(this.y),this.z=Math.ceil(this.z),this}round(){return this.x=Math.round(this.x),this.y=Math.round(this.y),this.z=Math.round(this.z),this}roundToZero(){return this.x=Math.trunc(this.x),this.y=Math.trunc(this.y),this.z=Math.trunc(this.z),this}negate(){return this.x=-this.x,this.y=-this.y,this.z=-this.z,this}dot(e){return this.x*e.x+this.y*e.y+this.z*e.z}lengthSq(){return this.x*this.x+this.y*this.y+this.z*this.z}length(){return Math.sqrt(this.x*this.x+this.y*this.y+this.z*this.z)}manhattanLength(){return Math.abs(this.x)+Math.abs(this.y)+Math.abs(this.z)}normalize(){return this.divideScalar(this.length()||1)}setLength(e){return this.normalize().multiplyScalar(e)}lerp(e,i){return this.x+=(e.x-this.x)*i,this.y+=(e.y-this.y)*i,this.z+=(e.z-this.z)*i,this}lerpVectors(e,i,r){return this.x=e.x+(i.x-e.x)*r,this.y=e.y+(i.y-e.y)*r,this.z=e.z+(i.z-e.z)*r,this}cross(e){return this.crossVectors(this,e)}crossVectors(e,i){const r=e.x,c=e.y,f=e.z,d=i.x,h=i.y,m=i.z;return this.x=c*m-f*h,this.y=f*d-r*m,this.z=r*h-c*d,this}projectOnVector(e){const i=e.lengthSq();if(i===0)return this.set(0,0,0);const r=e.dot(this)/i;return this.copy(e).multiplyScalar(r)}projectOnPlane(e){return Uh.copy(this).projectOnVector(e),this.sub(Uh)}reflect(e){return this.sub(Uh.copy(e).multiplyScalar(2*this.dot(e)))}angleTo(e){const i=Math.sqrt(this.lengthSq()*e.lengthSq());if(i===0)return Math.PI/2;const r=this.dot(e)/i;return Math.acos(De(r,-1,1))}distanceTo(e){return Math.sqrt(this.distanceToSquared(e))}distanceToSquared(e){const i=this.x-e.x,r=this.y-e.y,c=this.z-e.z;return i*i+r*r+c*c}manhattanDistanceTo(e){return Math.abs(this.x-e.x)+Math.abs(this.y-e.y)+Math.abs(this.z-e.z)}setFromSpherical(e){return this.setFromSphericalCoords(e.radius,e.phi,e.theta)}setFromSphericalCoords(e,i,r){const c=Math.sin(i)*e;return this.x=c*Math.sin(r),this.y=Math.cos(i)*e,this.z=c*Math.cos(r),this}setFromCylindrical(e){return this.setFromCylindricalCoords(e.radius,e.theta,e.y)}setFromCylindricalCoords(e,i,r){return this.x=e*Math.sin(i),this.y=r,this.z=e*Math.cos(i),this}setFromMatrixPosition(e){const i=e.elements;return this.x=i[12],this.y=i[13],this.z=i[14],this}setFromMatrixScale(e){const i=this.setFromMatrixColumn(e,0).length(),r=this.setFromMatrixColumn(e,1).length(),c=this.setFromMatrixColumn(e,2).length();return this.x=i,this.y=r,this.z=c,this}setFromMatrixColumn(e,i){return this.fromArray(e.elements,i*4)}setFromMatrix3Column(e,i){return this.fromArray(e.elements,i*3)}setFromEuler(e){return this.x=e._x,this.y=e._y,this.z=e._z,this}setFromColor(e){return this.x=e.r,this.y=e.g,this.z=e.b,this}equals(e){return e.x===this.x&&e.y===this.y&&e.z===this.z}fromArray(e,i=0){return this.x=e[i],this.y=e[i+1],this.z=e[i+2],this}toArray(e=[],i=0){return e[i]=this.x,e[i+1]=this.y,e[i+2]=this.z,e}fromBufferAttribute(e,i){return this.x=e.getX(i),this.y=e.getY(i),this.z=e.getZ(i),this}random(){return this.x=Math.random(),this.y=Math.random(),this.z=Math.random(),this}randomDirection(){const e=Math.random()*Math.PI*2,i=Math.random()*2-1,r=Math.sqrt(1-i*i);return this.x=r*Math.cos(e),this.y=i,this.z=r*Math.sin(e),this}*[Symbol.iterator](){yield this.x,yield this.y,yield this.z}};_m.prototype.isVector3=!0;let dt=_m;const Uh=new dt,k_=new _o,xm=class xm{constructor(e,i,r,c,f,d,h,m,p){this.elements=[1,0,0,0,1,0,0,0,1],e!==void 0&&this.set(e,i,r,c,f,d,h,m,p)}set(e,i,r,c,f,d,h,m,p){const x=this.elements;return x[0]=e,x[1]=c,x[2]=h,x[3]=i,x[4]=f,x[5]=m,x[6]=r,x[7]=d,x[8]=p,this}identity(){return this.set(1,0,0,0,1,0,0,0,1),this}copy(e){const i=this.elements,r=e.elements;return i[0]=r[0],i[1]=r[1],i[2]=r[2],i[3]=r[3],i[4]=r[4],i[5]=r[5],i[6]=r[6],i[7]=r[7],i[8]=r[8],this}extractBasis(e,i,r){return e.setFromMatrix3Column(this,0),i.setFromMatrix3Column(this,1),r.setFromMatrix3Column(this,2),this}setFromMatrix4(e){const i=e.elements;return this.set(i[0],i[4],i[8],i[1],i[5],i[9],i[2],i[6],i[10]),this}multiply(e){return this.multiplyMatrices(this,e)}premultiply(e){return this.multiplyMatrices(e,this)}multiplyMatrices(e,i){const r=e.elements,c=i.elements,f=this.elements,d=r[0],h=r[3],m=r[6],p=r[1],x=r[4],v=r[7],_=r[2],M=r[5],R=r[8],N=c[0],y=c[3],S=c[6],O=c[1],k=c[4],C=c[7],D=c[2],U=c[5],B=c[8];return f[0]=d*N+h*O+m*D,f[3]=d*y+h*k+m*U,f[6]=d*S+h*C+m*B,f[1]=p*N+x*O+v*D,f[4]=p*y+x*k+v*U,f[7]=p*S+x*C+v*B,f[2]=_*N+M*O+R*D,f[5]=_*y+M*k+R*U,f[8]=_*S+M*C+R*B,this}multiplyScalar(e){const i=this.elements;return i[0]*=e,i[3]*=e,i[6]*=e,i[1]*=e,i[4]*=e,i[7]*=e,i[2]*=e,i[5]*=e,i[8]*=e,this}determinant(){const e=this.elements,i=e[0],r=e[1],c=e[2],f=e[3],d=e[4],h=e[5],m=e[6],p=e[7],x=e[8];return i*d*x-i*h*p-r*f*x+r*h*m+c*f*p-c*d*m}invert(){const e=this.elements,i=e[0],r=e[1],c=e[2],f=e[3],d=e[4],h=e[5],m=e[6],p=e[7],x=e[8],v=x*d-h*p,_=h*m-x*f,M=p*f-d*m,R=i*v+r*_+c*M;if(R===0)return this.set(0,0,0,0,0,0,0,0,0);const N=1/R;return e[0]=v*N,e[1]=(c*p-x*r)*N,e[2]=(h*r-c*d)*N,e[3]=_*N,e[4]=(x*i-c*m)*N,e[5]=(c*f-h*i)*N,e[6]=M*N,e[7]=(r*m-p*i)*N,e[8]=(d*i-r*f)*N,this}transpose(){let e;const i=this.elements;return e=i[1],i[1]=i[3],i[3]=e,e=i[2],i[2]=i[6],i[6]=e,e=i[5],i[5]=i[7],i[7]=e,this}getNormalMatrix(e){return this.setFromMatrix4(e).invert().transpose()}transposeIntoArray(e){const i=this.elements;return e[0]=i[0],e[1]=i[3],e[2]=i[6],e[3]=i[1],e[4]=i[4],e[5]=i[7],e[6]=i[2],e[7]=i[5],e[8]=i[8],this}setUvTransform(e,i,r,c,f,d,h){const m=Math.cos(f),p=Math.sin(f);return this.set(r*m,r*p,-r*(m*d+p*h)+d+e,-c*p,c*m,-c*(-p*d+m*h)+h+i,0,0,1),this}scale(e,i){return fo("Matrix3: .scale() is deprecated. Use .makeScale() instead."),this.premultiply(Lh.makeScale(e,i)),this}rotate(e){return fo("Matrix3: .rotate() is deprecated. Use .makeRotation() instead."),this.premultiply(Lh.makeRotation(-e)),this}translate(e,i){return fo("Matrix3: .translate() is deprecated. Use .makeTranslation() instead."),this.premultiply(Lh.makeTranslation(e,i)),this}makeTranslation(e,i){return e.isVector2?this.set(1,0,e.x,0,1,e.y,0,0,1):this.set(1,0,e,0,1,i,0,0,1),this}makeRotation(e){const i=Math.cos(e),r=Math.sin(e);return this.set(i,-r,0,r,i,0,0,0,1),this}makeScale(e,i){return this.set(e,0,0,0,i,0,0,0,1),this}equals(e){const i=this.elements,r=e.elements;for(let c=0;c<9;c++)if(i[c]!==r[c])return!1;return!0}fromArray(e,i=0){for(let r=0;r<9;r++)this.elements[r]=e[r+i];return this}toArray(e=[],i=0){const r=this.elements;return e[i]=r[0],e[i+1]=r[1],e[i+2]=r[2],e[i+3]=r[3],e[i+4]=r[4],e[i+5]=r[5],e[i+6]=r[6],e[i+7]=r[7],e[i+8]=r[8],e}clone(){return new this.constructor().fromArray(this.elements)}};xm.prototype.isMatrix3=!0;let de=xm;const Lh=new de,X_=new de().set(.4123908,.3575843,.1804808,.212639,.7151687,.0721923,.0193308,.1191948,.9505322),W_=new de().set(3.2409699,-1.5373832,-.4986108,-.9692436,1.8759675,.0415551,.0556301,-.203977,1.0569715);function ab(){const o={enabled:!0,workingColorSpace:Fu,spaces:{},convert:function(c,f,d){return this.enabled===!1||f===d||!f||!d||(this.spaces[f].transfer===Ye&&(c.r=Ba(c.r),c.g=Ba(c.g),c.b=Ba(c.b)),this.spaces[f].primaries!==this.spaces[d].primaries&&(c.applyMatrix3(this.spaces[f].toXYZ),c.applyMatrix3(this.spaces[d].fromXYZ)),this.spaces[d].transfer===Ye&&(c.r=ho(c.r),c.g=ho(c.g),c.b=ho(c.b))),c},workingToColorSpace:function(c,f){return this.convert(c,this.workingColorSpace,f)},colorSpaceToWorking:function(c,f){return this.convert(c,f,this.workingColorSpace)},getPrimaries:function(c){return this.spaces[c].primaries},getTransfer:function(c){return c===ys?Hu:this.spaces[c].transfer},getToneMappingMode:function(c){return this.spaces[c].outputColorSpaceConfig.toneMappingMode||"standard"},getLuminanceCoefficients:function(c,f=this.workingColorSpace){return c.fromArray(this.spaces[f].luminanceCoefficients)},define:function(c){Object.assign(this.spaces,c)},_getMatrix:function(c,f,d){return c.copy(this.spaces[f].toXYZ).multiply(this.spaces[d].fromXYZ)},_getDrawingBufferColorSpace:function(c){return this.spaces[c].outputColorSpaceConfig.drawingBufferColorSpace},_getUnpackColorSpace:function(c=this.workingColorSpace){return this.spaces[c].workingColorSpaceConfig.unpackColorSpace},fromWorkingColorSpace:function(c,f){return fo("ColorManagement: .fromWorkingColorSpace() has been renamed to .workingToColorSpace()."),o.workingToColorSpace(c,f)},toWorkingColorSpace:function(c,f){return fo("ColorManagement: .toWorkingColorSpace() has been renamed to .colorSpaceToWorking()."),o.colorSpaceToWorking(c,f)}},e=[.64,.33,.3,.6,.15,.06],i=[.2126,.7152,.0722],r=[.3127,.329];return o.define({[Fu]:{primaries:e,whitePoint:r,transfer:Hu,toXYZ:X_,fromXYZ:W_,luminanceCoefficients:i,workingColorSpaceConfig:{unpackColorSpace:ti},outputColorSpaceConfig:{drawingBufferColorSpace:ti}},[ti]:{primaries:e,whitePoint:r,transfer:Ye,toXYZ:X_,fromXYZ:W_,luminanceCoefficients:i,outputColorSpaceConfig:{drawingBufferColorSpace:ti}}}),o}const Pe=ab();function Ba(o){return o<.04045?o*.0773993808:Math.pow(o*.9478672986+.0521327014,2.4)}function ho(o){return o<.0031308?o*12.92:1.055*Math.pow(o,.41666)-.055}let qr;class sb{static getDataURL(e,i="image/png"){if(/^data:/i.test(e.src)||typeof HTMLCanvasElement>"u")return e.src;let r;if(e instanceof HTMLCanvasElement)r=e;else{qr===void 0&&(qr=Gu("canvas")),qr.width=e.width,qr.height=e.height;const c=qr.getContext("2d");e instanceof ImageData?c.putImageData(e,0,0):c.drawImage(e,0,0,e.width,e.height),r=qr}return r.toDataURL(i)}static sRGBToLinear(e){if(typeof HTMLImageElement<"u"&&e instanceof HTMLImageElement||typeof HTMLCanvasElement<"u"&&e instanceof HTMLCanvasElement||typeof ImageBitmap<"u"&&e instanceof ImageBitmap){const i=Gu("canvas");i.width=e.width,i.height=e.height;const r=i.getContext("2d");r.drawImage(e,0,0,e.width,e.height);const c=r.getImageData(0,0,e.width,e.height),f=c.data;for(let d=0;d<f.length;d++)f[d]=Ba(f[d]/255)*255;return r.putImageData(c,0,0),i}else if(e.data){const i=e.data.slice(0);for(let r=0;r<i.length;r++)i instanceof Uint8Array||i instanceof Uint8ClampedArray?i[r]=Math.floor(Ba(i[r]/255)*255):i[r]=Ba(i[r]);return{data:i,width:e.width,height:e.height}}else return oe("ImageUtils.sRGBToLinear(): Unsupported image type. No color space conversion applied."),e}}let rb=0;class um{constructor(e=null){this.isTextureSource=!0,Object.defineProperty(this,"id",{value:rb++}),this.uuid=Bl(),this.data=e,this.dataReady=!0,this.version=0}getSize(e){const i=this.data;return typeof HTMLVideoElement<"u"&&i instanceof HTMLVideoElement?e.set(i.videoWidth,i.videoHeight,0):typeof VideoFrame<"u"&&i instanceof VideoFrame?e.set(i.displayWidth,i.displayHeight,0):i!==null?e.set(i.width,i.height,i.depth||0):e.set(0,0,0),e}set needsUpdate(e){e===!0&&this.version++}toJSON(e){const i=e===void 0||typeof e=="string";if(!i&&e.images[this.uuid]!==void 0)return e.images[this.uuid];const r={uuid:this.uuid,url:""},c=this.data;if(c!==null){let f;if(Array.isArray(c)){f=[];for(let d=0,h=c.length;d<h;d++)c[d].isDataTexture?f.push(Oh(c[d].image)):f.push(Oh(c[d]))}else f=Oh(c);r.url=f}return i||(e.images[this.uuid]=r),r}}function Oh(o){return typeof HTMLImageElement<"u"&&o instanceof HTMLImageElement||typeof HTMLCanvasElement<"u"&&o instanceof HTMLCanvasElement||typeof ImageBitmap<"u"&&o instanceof ImageBitmap?sb.getDataURL(o):o.data?{data:Array.from(o.data),width:o.width,height:o.height,type:o.data.constructor.name}:(oe("Texture: Unable to serialize Texture."),{})}let ob=0;const Ph=new dt;class Fn extends nr{constructor(e=Fn.DEFAULT_IMAGE,i=Fn.DEFAULT_MAPPING,r=Pa,c=Pa,f=zn,d=Qs,h=Gi,m=pi,p=Fn.DEFAULT_ANISOTROPY,x=ys){super(),this.isTexture=!0,Object.defineProperty(this,"id",{value:ob++}),this.uuid=Bl(),this.name="",this.source=new um(e),this.mipmaps=[],this.mapping=i,this.channel=0,this.wrapS=r,this.wrapT=c,this.magFilter=f,this.minFilter=d,this.anisotropy=p,this.format=h,this.internalFormat=null,this.type=m,this.offset=new Re(0,0),this.repeat=new Re(1,1),this.center=new Re(0,0),this.rotation=0,this.matrixAutoUpdate=!0,this.matrix=new de,this.generateMipmaps=!0,this.premultiplyAlpha=!1,this.flipY=!0,this.unpackAlignment=4,this.colorSpace=x,this.userData={},this.updateRanges=[],this.version=0,this.onUpdate=null,this.renderTarget=null,this.isRenderTargetTexture=!1,this.isArrayTexture=!!(e&&e.depth&&e.depth>1),this.pmremVersion=0,this.normalized=!1}get width(){return this.source.getSize(Ph).x}get height(){return this.source.getSize(Ph).y}get depth(){return this.source.getSize(Ph).z}get image(){return this.source.data}set image(e){this.source.data=e}updateMatrix(){this.matrix.setUvTransform(this.offset.x,this.offset.y,this.repeat.x,this.repeat.y,this.rotation,this.center.x,this.center.y)}addUpdateRange(e,i){this.updateRanges.push({start:e,count:i})}clearUpdateRanges(){this.updateRanges.length=0}clone(){return new this.constructor().copy(this)}copy(e){return this.name=e.name,this.source=e.source,this.mipmaps=e.mipmaps.slice(0),this.mapping=e.mapping,this.channel=e.channel,this.wrapS=e.wrapS,this.wrapT=e.wrapT,this.magFilter=e.magFilter,this.minFilter=e.minFilter,this.anisotropy=e.anisotropy,this.format=e.format,this.internalFormat=e.internalFormat,this.type=e.type,this.normalized=e.normalized,this.offset.copy(e.offset),this.repeat.copy(e.repeat),this.center.copy(e.center),this.rotation=e.rotation,this.matrixAutoUpdate=e.matrixAutoUpdate,this.matrix.copy(e.matrix),this.generateMipmaps=e.generateMipmaps,this.premultiplyAlpha=e.premultiplyAlpha,this.flipY=e.flipY,this.unpackAlignment=e.unpackAlignment,this.colorSpace=e.colorSpace,this.renderTarget=e.renderTarget,this.isRenderTargetTexture=e.isRenderTargetTexture,this.isArrayTexture=e.isArrayTexture,this.userData=JSON.parse(JSON.stringify(e.userData)),this.needsUpdate=!0,this}setValues(e){for(const i in e){const r=e[i];if(r===void 0){oe(`Texture.setValues(): parameter '${i}' has value of undefined.`);continue}const c=this[i];if(c===void 0){oe(`Texture.setValues(): property '${i}' does not exist.`);continue}c&&r&&c.isVector2&&r.isVector2||c&&r&&c.isVector3&&r.isVector3||c&&r&&c.isMatrix3&&r.isMatrix3?c.copy(r):this[i]=r}}toJSON(e){const i=e===void 0||typeof e=="string";if(!i&&e.textures[this.uuid]!==void 0)return e.textures[this.uuid];const r={metadata:{version:4.7,type:"Texture",generator:"Texture.toJSON"},uuid:this.uuid,name:this.name,image:this.source.toJSON(e).uuid,mapping:this.mapping,channel:this.channel,repeat:[this.repeat.x,this.repeat.y],offset:[this.offset.x,this.offset.y],center:[this.center.x,this.center.y],rotation:this.rotation,wrap:[this.wrapS,this.wrapT],format:this.format,internalFormat:this.internalFormat,type:this.type,normalized:this.normalized,colorSpace:this.colorSpace,minFilter:this.minFilter,magFilter:this.magFilter,anisotropy:this.anisotropy,flipY:this.flipY,generateMipmaps:this.generateMipmaps,premultiplyAlpha:this.premultiplyAlpha,unpackAlignment:this.unpackAlignment};return Object.keys(this.userData).length>0&&(r.userData=this.userData),i||(e.textures[this.uuid]=r),r}dispose(){this.dispatchEvent({type:"dispose"})}transformUv(e){if(this.mapping!==Qx)return e;if(e.applyMatrix3(this.matrix),e.x<0||e.x>1)switch(this.wrapS){case pp:e.x=e.x-Math.floor(e.x);break;case Pa:e.x=e.x<0?0:1;break;case mp:Math.abs(Math.floor(e.x)%2)===1?e.x=Math.ceil(e.x)-e.x:e.x=e.x-Math.floor(e.x);break}if(e.y<0||e.y>1)switch(this.wrapT){case pp:e.y=e.y-Math.floor(e.y);break;case Pa:e.y=e.y<0?0:1;break;case mp:Math.abs(Math.floor(e.y)%2)===1?e.y=Math.ceil(e.y)-e.y:e.y=e.y-Math.floor(e.y);break}return this.flipY&&(e.y=1-e.y),e}set needsUpdate(e){e===!0&&(this.version++,this.source.needsUpdate=!0)}set needsPMREMUpdate(e){e===!0&&this.pmremVersion++}}Fn.DEFAULT_IMAGE=null;Fn.DEFAULT_MAPPING=Qx;Fn.DEFAULT_ANISOTROPY=1;const Sm=class Sm{constructor(e=0,i=0,r=0,c=1){this.x=e,this.y=i,this.z=r,this.w=c}get width(){return this.z}set width(e){this.z=e}get height(){return this.w}set height(e){this.w=e}set(e,i,r,c){return this.x=e,this.y=i,this.z=r,this.w=c,this}setScalar(e){return this.x=e,this.y=e,this.z=e,this.w=e,this}setX(e){return this.x=e,this}setY(e){return this.y=e,this}setZ(e){return this.z=e,this}setW(e){return this.w=e,this}setComponent(e,i){switch(e){case 0:this.x=i;break;case 1:this.y=i;break;case 2:this.z=i;break;case 3:this.w=i;break;default:throw new Error("THREE.Vector4: index is out of range: "+e)}return this}getComponent(e){switch(e){case 0:return this.x;case 1:return this.y;case 2:return this.z;case 3:return this.w;default:throw new Error("THREE.Vector4: index is out of range: "+e)}}clone(){return new this.constructor(this.x,this.y,this.z,this.w)}copy(e){return this.x=e.x,this.y=e.y,this.z=e.z,this.w=e.w!==void 0?e.w:1,this}add(e){return this.x+=e.x,this.y+=e.y,this.z+=e.z,this.w+=e.w,this}addScalar(e){return this.x+=e,this.y+=e,this.z+=e,this.w+=e,this}addVectors(e,i){return this.x=e.x+i.x,this.y=e.y+i.y,this.z=e.z+i.z,this.w=e.w+i.w,this}addScaledVector(e,i){return this.x+=e.x*i,this.y+=e.y*i,this.z+=e.z*i,this.w+=e.w*i,this}sub(e){return this.x-=e.x,this.y-=e.y,this.z-=e.z,this.w-=e.w,this}subScalar(e){return this.x-=e,this.y-=e,this.z-=e,this.w-=e,this}subVectors(e,i){return this.x=e.x-i.x,this.y=e.y-i.y,this.z=e.z-i.z,this.w=e.w-i.w,this}multiply(e){return this.x*=e.x,this.y*=e.y,this.z*=e.z,this.w*=e.w,this}multiplyScalar(e){return this.x*=e,this.y*=e,this.z*=e,this.w*=e,this}applyMatrix4(e){const i=this.x,r=this.y,c=this.z,f=this.w,d=e.elements;return this.x=d[0]*i+d[4]*r+d[8]*c+d[12]*f,this.y=d[1]*i+d[5]*r+d[9]*c+d[13]*f,this.z=d[2]*i+d[6]*r+d[10]*c+d[14]*f,this.w=d[3]*i+d[7]*r+d[11]*c+d[15]*f,this}divide(e){return this.x/=e.x,this.y/=e.y,this.z/=e.z,this.w/=e.w,this}divideScalar(e){return this.multiplyScalar(1/e)}setAxisAngleFromQuaternion(e){this.w=2*Math.acos(e.w);const i=Math.sqrt(1-e.w*e.w);return i<1e-4?(this.x=1,this.y=0,this.z=0):(this.x=e.x/i,this.y=e.y/i,this.z=e.z/i),this}setAxisAngleFromRotationMatrix(e){let i,r,c,f;const m=e.elements,p=m[0],x=m[4],v=m[8],_=m[1],M=m[5],R=m[9],N=m[2],y=m[6],S=m[10];if(Math.abs(x-_)<.01&&Math.abs(v-N)<.01&&Math.abs(R-y)<.01){if(Math.abs(x+_)<.1&&Math.abs(v+N)<.1&&Math.abs(R+y)<.1&&Math.abs(p+M+S-3)<.1)return this.set(1,0,0,0),this;i=Math.PI;const k=(p+1)/2,C=(M+1)/2,D=(S+1)/2,U=(x+_)/4,B=(v+N)/4,b=(R+y)/4;return k>C&&k>D?k<.01?(r=0,c=.707106781,f=.707106781):(r=Math.sqrt(k),c=U/r,f=B/r):C>D?C<.01?(r=.707106781,c=0,f=.707106781):(c=Math.sqrt(C),r=U/c,f=b/c):D<.01?(r=.707106781,c=.707106781,f=0):(f=Math.sqrt(D),r=B/f,c=b/f),this.set(r,c,f,i),this}let O=Math.sqrt((y-R)*(y-R)+(v-N)*(v-N)+(_-x)*(_-x));return Math.abs(O)<.001&&(O=1),this.x=(y-R)/O,this.y=(v-N)/O,this.z=(_-x)/O,this.w=Math.acos((p+M+S-1)/2),this}setFromMatrixPosition(e){const i=e.elements;return this.x=i[12],this.y=i[13],this.z=i[14],this.w=i[15],this}min(e){return this.x=Math.min(this.x,e.x),this.y=Math.min(this.y,e.y),this.z=Math.min(this.z,e.z),this.w=Math.min(this.w,e.w),this}max(e){return this.x=Math.max(this.x,e.x),this.y=Math.max(this.y,e.y),this.z=Math.max(this.z,e.z),this.w=Math.max(this.w,e.w),this}clamp(e,i){return this.x=De(this.x,e.x,i.x),this.y=De(this.y,e.y,i.y),this.z=De(this.z,e.z,i.z),this.w=De(this.w,e.w,i.w),this}clampScalar(e,i){return this.x=De(this.x,e,i),this.y=De(this.y,e,i),this.z=De(this.z,e,i),this.w=De(this.w,e,i),this}clampLength(e,i){const r=this.length();return this.divideScalar(r||1).multiplyScalar(De(r,e,i))}floor(){return this.x=Math.floor(this.x),this.y=Math.floor(this.y),this.z=Math.floor(this.z),this.w=Math.floor(this.w),this}ceil(){return this.x=Math.ceil(this.x),this.y=Math.ceil(this.y),this.z=Math.ceil(this.z),this.w=Math.ceil(this.w),this}round(){return this.x=Math.round(this.x),this.y=Math.round(this.y),this.z=Math.round(this.z),this.w=Math.round(this.w),this}roundToZero(){return this.x=Math.trunc(this.x),this.y=Math.trunc(this.y),this.z=Math.trunc(this.z),this.w=Math.trunc(this.w),this}negate(){return this.x=-this.x,this.y=-this.y,this.z=-this.z,this.w=-this.w,this}dot(e){return this.x*e.x+this.y*e.y+this.z*e.z+this.w*e.w}lengthSq(){return this.x*this.x+this.y*this.y+this.z*this.z+this.w*this.w}length(){return Math.sqrt(this.x*this.x+this.y*this.y+this.z*this.z+this.w*this.w)}manhattanLength(){return Math.abs(this.x)+Math.abs(this.y)+Math.abs(this.z)+Math.abs(this.w)}normalize(){return this.divideScalar(this.length()||1)}setLength(e){return this.normalize().multiplyScalar(e)}lerp(e,i){return this.x+=(e.x-this.x)*i,this.y+=(e.y-this.y)*i,this.z+=(e.z-this.z)*i,this.w+=(e.w-this.w)*i,this}lerpVectors(e,i,r){return this.x=e.x+(i.x-e.x)*r,this.y=e.y+(i.y-e.y)*r,this.z=e.z+(i.z-e.z)*r,this.w=e.w+(i.w-e.w)*r,this}equals(e){return e.x===this.x&&e.y===this.y&&e.z===this.z&&e.w===this.w}fromArray(e,i=0){return this.x=e[i],this.y=e[i+1],this.z=e[i+2],this.w=e[i+3],this}toArray(e=[],i=0){return e[i]=this.x,e[i+1]=this.y,e[i+2]=this.z,e[i+3]=this.w,e}fromBufferAttribute(e,i){return this.x=e.getX(i),this.y=e.getY(i),this.z=e.getZ(i),this.w=e.getW(i),this}random(){return this.x=Math.random(),this.y=Math.random(),this.z=Math.random(),this.w=Math.random(),this}*[Symbol.iterator](){yield this.x,yield this.y,yield this.z,yield this.w}};Sm.prototype.isVector4=!0;let on=Sm;class lb extends nr{constructor(e=1,i=1,r={}){super(),r=Object.assign({generateMipmaps:!1,internalFormat:null,minFilter:zn,depthBuffer:!0,stencilBuffer:!1,resolveColorBuffer:!0,resolveDepthBuffer:!0,resolveStencilBuffer:!0,storeMultisampledColorBuffer:!0,storeMultisampledDepthBuffer:!0,storeMultisampledStencilBuffer:!0,depthTexture:null,samples:0,count:1,depth:1,multiview:!1,useArrayDepthTexture:!1},r),this.isRenderTarget=!0,this.width=e,this.height=i,this.depth=r.depth,this.scissor=new on(0,0,e,i),this.scissorTest=!1,this.viewport=new on(0,0,e,i),this.textures=[];const c={width:e,height:i,depth:r.depth},f=new Fn(c),d=r.count;for(let h=0;h<d;h++)this.textures[h]=f.clone(),this.textures[h].isRenderTargetTexture=!0,this.textures[h].renderTarget=this;this._setTextureOptions(r),this.depthBuffer=r.depthBuffer,this.stencilBuffer=r.stencilBuffer,this.resolveColorBuffer=r.resolveColorBuffer,this.resolveDepthBuffer=r.resolveDepthBuffer,this.resolveStencilBuffer=r.resolveStencilBuffer,this.storeMultisampledColorBuffer=r.storeMultisampledColorBuffer,this.storeMultisampledDepthBuffer=r.storeMultisampledDepthBuffer,this.storeMultisampledStencilBuffer=r.storeMultisampledStencilBuffer,this._depthTexture=null,this.depthTexture=r.depthTexture,this.samples=r.samples,this.multiview=r.multiview,this.useArrayDepthTexture=r.useArrayDepthTexture}_setTextureOptions(e={}){const i={minFilter:zn,generateMipmaps:!1,flipY:!1,internalFormat:null};e.mapping!==void 0&&(i.mapping=e.mapping),e.wrapS!==void 0&&(i.wrapS=e.wrapS),e.wrapT!==void 0&&(i.wrapT=e.wrapT),e.wrapR!==void 0&&(i.wrapR=e.wrapR),e.magFilter!==void 0&&(i.magFilter=e.magFilter),e.minFilter!==void 0&&(i.minFilter=e.minFilter),e.format!==void 0&&(i.format=e.format),e.type!==void 0&&(i.type=e.type),e.anisotropy!==void 0&&(i.anisotropy=e.anisotropy),e.colorSpace!==void 0&&(i.colorSpace=e.colorSpace),e.flipY!==void 0&&(i.flipY=e.flipY),e.generateMipmaps!==void 0&&(i.generateMipmaps=e.generateMipmaps),e.internalFormat!==void 0&&(i.internalFormat=e.internalFormat);for(let r=0;r<this.textures.length;r++)this.textures[r].setValues(i)}get texture(){return this.textures[0]}set texture(e){this.textures[0]=e}set depthTexture(e){this._depthTexture!==null&&this._depthTexture.renderTarget===this&&(this._depthTexture.renderTarget=null),e!==null&&e.renderTarget===null&&(e.renderTarget=this),this._depthTexture=e}get depthTexture(){return this._depthTexture}setSize(e,i,r=1){if(this.width!==e||this.height!==i||this.depth!==r){this.width=e,this.height=i,this.depth=r;for(let c=0,f=this.textures.length;c<f;c++)this.textures[c].image.width=e,this.textures[c].image.height=i,this.textures[c].image.depth=r,this.textures[c].isData3DTexture!==!0&&(this.textures[c].isArrayTexture=this.textures[c].image.depth>1);this.dispose()}this.viewport.set(0,0,e,i),this.scissor.set(0,0,e,i)}clone(){return new this.constructor().copy(this)}copy(e){this.width=e.width,this.height=e.height,this.depth=e.depth,this.scissor.copy(e.scissor),this.scissorTest=e.scissorTest,this.viewport.copy(e.viewport),this.textures.length=0;for(let i=0,r=e.textures.length;i<r;i++){this.textures[i]=e.textures[i].clone(),this.textures[i].isRenderTargetTexture=!0,this.textures[i].renderTarget=this;const c=Object.assign({},e.textures[i].image);this.textures[i].source=new um(c)}if(this.depthBuffer=e.depthBuffer,this.stencilBuffer=e.stencilBuffer,this.resolveColorBuffer=e.resolveColorBuffer,this.resolveDepthBuffer=e.resolveDepthBuffer,this.resolveStencilBuffer=e.resolveStencilBuffer,this.storeMultisampledColorBuffer=e.storeMultisampledColorBuffer,this.storeMultisampledDepthBuffer=e.storeMultisampledDepthBuffer,this.storeMultisampledStencilBuffer=e.storeMultisampledStencilBuffer,e.depthTexture!==null)if(e.depthTexture.renderTarget===e){const i=e.depthTexture.clone();i.renderTarget=null,this.depthTexture=i}else this.depthTexture=e.depthTexture;return this.samples=e.samples,this.multiview=e.multiview,this.useArrayDepthTexture=e.useArrayDepthTexture,this}dispose(){this.dispatchEvent({type:"dispose"})}}class Vi extends lb{constructor(e=1,i=1,r={}){super(e,i,r),this.isWebGLRenderTarget=!0}}class sS extends Fn{constructor(e=null,i=1,r=1,c=1){super(null),this.isDataArrayTexture=!0,this.image={data:e,width:i,height:r,depth:c},this.magFilter=On,this.minFilter=On,this.wrapR=Pa,this.generateMipmaps=!1,this.flipY=!1,this.unpackAlignment=1,this.layerUpdates=new Set}copy(e){return super.copy(e),this.wrapR=e.wrapR,this}addLayerUpdate(e){this.layerUpdates.add(e)}clearLayerUpdates(){this.layerUpdates.clear()}}class cb extends Fn{constructor(e=null,i=1,r=1,c=1){super(null),this.isData3DTexture=!0,this.image={data:e,width:i,height:r,depth:c},this.magFilter=On,this.minFilter=On,this.wrapR=Pa,this.generateMipmaps=!1,this.flipY=!1,this.unpackAlignment=1}copy(e){return super.copy(e),this.wrapR=e.wrapR,this}}const Vu=class Vu{constructor(e,i,r,c,f,d,h,m,p,x,v,_,M,R,N,y){this.elements=[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1],e!==void 0&&this.set(e,i,r,c,f,d,h,m,p,x,v,_,M,R,N,y)}set(e,i,r,c,f,d,h,m,p,x,v,_,M,R,N,y){const S=this.elements;return S[0]=e,S[4]=i,S[8]=r,S[12]=c,S[1]=f,S[5]=d,S[9]=h,S[13]=m,S[2]=p,S[6]=x,S[10]=v,S[14]=_,S[3]=M,S[7]=R,S[11]=N,S[15]=y,this}identity(){return this.set(1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1),this}clone(){return new Vu().fromArray(this.elements)}copy(e){const i=this.elements,r=e.elements;return i[0]=r[0],i[1]=r[1],i[2]=r[2],i[3]=r[3],i[4]=r[4],i[5]=r[5],i[6]=r[6],i[7]=r[7],i[8]=r[8],i[9]=r[9],i[10]=r[10],i[11]=r[11],i[12]=r[12],i[13]=r[13],i[14]=r[14],i[15]=r[15],this}copyPosition(e){const i=this.elements,r=e.elements;return i[12]=r[12],i[13]=r[13],i[14]=r[14],this}setFromMatrix3(e){const i=e.elements;return this.set(i[0],i[3],i[6],0,i[1],i[4],i[7],0,i[2],i[5],i[8],0,0,0,0,1),this}extractBasis(e,i,r){return this.determinantAffine()===0?(e.set(1,0,0),i.set(0,1,0),r.set(0,0,1),this):(e.setFromMatrixColumn(this,0),i.setFromMatrixColumn(this,1),r.setFromMatrixColumn(this,2),this)}makeBasis(e,i,r){return this.set(e.x,i.x,r.x,0,e.y,i.y,r.y,0,e.z,i.z,r.z,0,0,0,0,1),this}extractRotation(e){if(e.determinantAffine()===0)return this.identity();const i=this.elements,r=e.elements,c=1/Yr.setFromMatrixColumn(e,0).length(),f=1/Yr.setFromMatrixColumn(e,1).length(),d=1/Yr.setFromMatrixColumn(e,2).length();return i[0]=r[0]*c,i[1]=r[1]*c,i[2]=r[2]*c,i[3]=0,i[4]=r[4]*f,i[5]=r[5]*f,i[6]=r[6]*f,i[7]=0,i[8]=r[8]*d,i[9]=r[9]*d,i[10]=r[10]*d,i[11]=0,i[12]=0,i[13]=0,i[14]=0,i[15]=1,this}makeRotationFromEuler(e){const i=this.elements,r=e.x,c=e.y,f=e.z,d=Math.cos(r),h=Math.sin(r),m=Math.cos(c),p=Math.sin(c),x=Math.cos(f),v=Math.sin(f);if(e.order==="XYZ"){const _=d*x,M=d*v,R=h*x,N=h*v;i[0]=m*x,i[4]=-m*v,i[8]=p,i[1]=M+R*p,i[5]=_-N*p,i[9]=-h*m,i[2]=N-_*p,i[6]=R+M*p,i[10]=d*m}else if(e.order==="YXZ"){const _=m*x,M=m*v,R=p*x,N=p*v;i[0]=_+N*h,i[4]=R*h-M,i[8]=d*p,i[1]=d*v,i[5]=d*x,i[9]=-h,i[2]=M*h-R,i[6]=N+_*h,i[10]=d*m}else if(e.order==="ZXY"){const _=m*x,M=m*v,R=p*x,N=p*v;i[0]=_-N*h,i[4]=-d*v,i[8]=R+M*h,i[1]=M+R*h,i[5]=d*x,i[9]=N-_*h,i[2]=-d*p,i[6]=h,i[10]=d*m}else if(e.order==="ZYX"){const _=d*x,M=d*v,R=h*x,N=h*v;i[0]=m*x,i[4]=R*p-M,i[8]=_*p+N,i[1]=m*v,i[5]=N*p+_,i[9]=M*p-R,i[2]=-p,i[6]=h*m,i[10]=d*m}else if(e.order==="YZX"){const _=d*m,M=d*p,R=h*m,N=h*p;i[0]=m*x,i[4]=N-_*v,i[8]=R*v+M,i[1]=v,i[5]=d*x,i[9]=-h*x,i[2]=-p*x,i[6]=M*v+R,i[10]=_-N*v}else if(e.order==="XZY"){const _=d*m,M=d*p,R=h*m,N=h*p;i[0]=m*x,i[4]=-v,i[8]=p*x,i[1]=_*v+N,i[5]=d*x,i[9]=M*v-R,i[2]=R*v-M,i[6]=h*x,i[10]=N*v+_}return i[3]=0,i[7]=0,i[11]=0,i[12]=0,i[13]=0,i[14]=0,i[15]=1,this}makeRotationFromQuaternion(e){return this.compose(ub,e,fb)}lookAt(e,i,r){const c=this.elements;return di.subVectors(e,i),di.lengthSq()===0&&(di.z=1),di.normalize(),ps.crossVectors(r,di),ps.lengthSq()===0&&(Math.abs(r.z)===1?di.x+=1e-4:di.z+=1e-4,di.normalize(),ps.crossVectors(r,di)),ps.normalize(),lu.crossVectors(di,ps),c[0]=ps.x,c[4]=lu.x,c[8]=di.x,c[1]=ps.y,c[5]=lu.y,c[9]=di.y,c[2]=ps.z,c[6]=lu.z,c[10]=di.z,this}multiply(e){return this.multiplyMatrices(this,e)}premultiply(e){return this.multiplyMatrices(e,this)}multiplyMatrices(e,i){const r=e.elements,c=i.elements,f=this.elements,d=r[0],h=r[4],m=r[8],p=r[12],x=r[1],v=r[5],_=r[9],M=r[13],R=r[2],N=r[6],y=r[10],S=r[14],O=r[3],k=r[7],C=r[11],D=r[15],U=c[0],B=c[4],b=c[8],L=c[12],X=c[1],j=c[5],J=c[9],P=c[13],q=c[2],$=c[6],Y=c[10],Q=c[14],pt=c[3],ot=c[7],vt=c[11],Tt=c[15];return f[0]=d*U+h*X+m*q+p*pt,f[4]=d*B+h*j+m*$+p*ot,f[8]=d*b+h*J+m*Y+p*vt,f[12]=d*L+h*P+m*Q+p*Tt,f[1]=x*U+v*X+_*q+M*pt,f[5]=x*B+v*j+_*$+M*ot,f[9]=x*b+v*J+_*Y+M*vt,f[13]=x*L+v*P+_*Q+M*Tt,f[2]=R*U+N*X+y*q+S*pt,f[6]=R*B+N*j+y*$+S*ot,f[10]=R*b+N*J+y*Y+S*vt,f[14]=R*L+N*P+y*Q+S*Tt,f[3]=O*U+k*X+C*q+D*pt,f[7]=O*B+k*j+C*$+D*ot,f[11]=O*b+k*J+C*Y+D*vt,f[15]=O*L+k*P+C*Q+D*Tt,this}multiplyScalar(e){const i=this.elements;return i[0]*=e,i[4]*=e,i[8]*=e,i[12]*=e,i[1]*=e,i[5]*=e,i[9]*=e,i[13]*=e,i[2]*=e,i[6]*=e,i[10]*=e,i[14]*=e,i[3]*=e,i[7]*=e,i[11]*=e,i[15]*=e,this}determinant(){const e=this.elements,i=e[0],r=e[4],c=e[8],f=e[12],d=e[1],h=e[5],m=e[9],p=e[13],x=e[2],v=e[6],_=e[10],M=e[14],R=e[3],N=e[7],y=e[11],S=e[15],O=m*M-p*_,k=h*M-p*v,C=h*_-m*v,D=d*M-p*x,U=d*_-m*x,B=d*v-h*x;return i*(N*O-y*k+S*C)-r*(R*O-y*D+S*U)+c*(R*k-N*D+S*B)-f*(R*C-N*U+y*B)}determinantAffine(){const e=this.elements,i=e[0],r=e[4],c=e[8],f=e[1],d=e[5],h=e[9],m=e[2],p=e[6],x=e[10];return i*(d*x-h*p)-r*(f*x-h*m)+c*(f*p-d*m)}transpose(){const e=this.elements;let i;return i=e[1],e[1]=e[4],e[4]=i,i=e[2],e[2]=e[8],e[8]=i,i=e[6],e[6]=e[9],e[9]=i,i=e[3],e[3]=e[12],e[12]=i,i=e[7],e[7]=e[13],e[13]=i,i=e[11],e[11]=e[14],e[14]=i,this}setPosition(e,i,r){const c=this.elements;return e.isVector3?(c[12]=e.x,c[13]=e.y,c[14]=e.z):(c[12]=e,c[13]=i,c[14]=r),this}invert(){const e=this.elements,i=e[0],r=e[1],c=e[2],f=e[3],d=e[4],h=e[5],m=e[6],p=e[7],x=e[8],v=e[9],_=e[10],M=e[11],R=e[12],N=e[13],y=e[14],S=e[15],O=i*h-r*d,k=i*m-c*d,C=i*p-f*d,D=r*m-c*h,U=r*p-f*h,B=c*p-f*m,b=x*N-v*R,L=x*y-_*R,X=x*S-M*R,j=v*y-_*N,J=v*S-M*N,P=_*S-M*y,q=O*P-k*J+C*j+D*X-U*L+B*b;if(q===0)return this.set(0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0);const $=1/q;return e[0]=(h*P-m*J+p*j)*$,e[1]=(c*J-r*P-f*j)*$,e[2]=(N*B-y*U+S*D)*$,e[3]=(_*U-v*B-M*D)*$,e[4]=(m*X-d*P-p*L)*$,e[5]=(i*P-c*X+f*L)*$,e[6]=(y*C-R*B-S*k)*$,e[7]=(x*B-_*C+M*k)*$,e[8]=(d*J-h*X+p*b)*$,e[9]=(r*X-i*J-f*b)*$,e[10]=(R*U-N*C+S*O)*$,e[11]=(v*C-x*U-M*O)*$,e[12]=(h*L-d*j-m*b)*$,e[13]=(i*j-r*L+c*b)*$,e[14]=(N*k-R*D-y*O)*$,e[15]=(x*D-v*k+_*O)*$,this}scale(e){const i=this.elements,r=e.x,c=e.y,f=e.z;return i[0]*=r,i[4]*=c,i[8]*=f,i[1]*=r,i[5]*=c,i[9]*=f,i[2]*=r,i[6]*=c,i[10]*=f,i[3]*=r,i[7]*=c,i[11]*=f,this}getMaxScaleOnAxis(){const e=this.elements,i=e[0]*e[0]+e[1]*e[1]+e[2]*e[2],r=e[4]*e[4]+e[5]*e[5]+e[6]*e[6],c=e[8]*e[8]+e[9]*e[9]+e[10]*e[10];return Math.sqrt(Math.max(i,r,c))}makeTranslation(e,i,r){return e.isVector3?this.set(1,0,0,e.x,0,1,0,e.y,0,0,1,e.z,0,0,0,1):this.set(1,0,0,e,0,1,0,i,0,0,1,r,0,0,0,1),this}makeRotationX(e){const i=Math.cos(e),r=Math.sin(e);return this.set(1,0,0,0,0,i,-r,0,0,r,i,0,0,0,0,1),this}makeRotationY(e){const i=Math.cos(e),r=Math.sin(e);return this.set(i,0,r,0,0,1,0,0,-r,0,i,0,0,0,0,1),this}makeRotationZ(e){const i=Math.cos(e),r=Math.sin(e);return this.set(i,-r,0,0,r,i,0,0,0,0,1,0,0,0,0,1),this}makeRotationAxis(e,i){const r=Math.cos(i),c=Math.sin(i),f=1-r,d=e.x,h=e.y,m=e.z,p=f*d,x=f*h;return this.set(p*d+r,p*h-c*m,p*m+c*h,0,p*h+c*m,x*h+r,x*m-c*d,0,p*m-c*h,x*m+c*d,f*m*m+r,0,0,0,0,1),this}makeScale(e,i,r){return this.set(e,0,0,0,0,i,0,0,0,0,r,0,0,0,0,1),this}makeShear(e,i,r,c,f,d){return this.set(1,r,f,0,e,1,d,0,i,c,1,0,0,0,0,1),this}compose(e,i,r){const c=this.elements,f=i._x,d=i._y,h=i._z,m=i._w,p=f+f,x=d+d,v=h+h,_=f*p,M=f*x,R=f*v,N=d*x,y=d*v,S=h*v,O=m*p,k=m*x,C=m*v,D=r.x,U=r.y,B=r.z;return c[0]=(1-(N+S))*D,c[1]=(M+C)*D,c[2]=(R-k)*D,c[3]=0,c[4]=(M-C)*U,c[5]=(1-(_+S))*U,c[6]=(y+O)*U,c[7]=0,c[8]=(R+k)*B,c[9]=(y-O)*B,c[10]=(1-(_+N))*B,c[11]=0,c[12]=e.x,c[13]=e.y,c[14]=e.z,c[15]=1,this}decompose(e,i,r){const c=this.elements;e.x=c[12],e.y=c[13],e.z=c[14];const f=this.determinantAffine();if(f===0)return r.set(1,1,1),i.identity(),this;let d=Yr.set(c[0],c[1],c[2]).length();const h=Yr.set(c[4],c[5],c[6]).length(),m=Yr.set(c[8],c[9],c[10]).length();f<0&&(d=-d),Oi.copy(this);const p=1/d,x=1/h,v=1/m;return Oi.elements[0]*=p,Oi.elements[1]*=p,Oi.elements[2]*=p,Oi.elements[4]*=x,Oi.elements[5]*=x,Oi.elements[6]*=x,Oi.elements[8]*=v,Oi.elements[9]*=v,Oi.elements[10]*=v,i.setFromRotationMatrix(Oi),r.x=d,r.y=h,r.z=m,this}makePerspective(e,i,r,c,f,d,h=la,m=!1){const p=this.elements,x=2*f/(i-e),v=2*f/(r-c),_=(i+e)/(i-e),M=(r+c)/(r-c);let R,N;if(m)R=f/(d-f),N=d*f/(d-f);else if(h===la)R=-(d+f)/(d-f),N=-2*d*f/(d-f);else if(h===Pl)R=-d/(d-f),N=-d*f/(d-f);else throw new Error("THREE.Matrix4.makePerspective(): Invalid coordinate system: "+h);return p[0]=x,p[4]=0,p[8]=_,p[12]=0,p[1]=0,p[5]=v,p[9]=M,p[13]=0,p[2]=0,p[6]=0,p[10]=R,p[14]=N,p[3]=0,p[7]=0,p[11]=-1,p[15]=0,this}makeOrthographic(e,i,r,c,f,d,h=la,m=!1){const p=this.elements,x=2/(i-e),v=2/(r-c),_=-(i+e)/(i-e),M=-(r+c)/(r-c);let R,N;if(m)R=1/(d-f),N=d/(d-f);else if(h===la)R=-2/(d-f),N=-(d+f)/(d-f);else if(h===Pl)R=-1/(d-f),N=-f/(d-f);else throw new Error("THREE.Matrix4.makeOrthographic(): Invalid coordinate system: "+h);return p[0]=x,p[4]=0,p[8]=0,p[12]=_,p[1]=0,p[5]=v,p[9]=0,p[13]=M,p[2]=0,p[6]=0,p[10]=R,p[14]=N,p[3]=0,p[7]=0,p[11]=0,p[15]=1,this}equals(e){const i=this.elements,r=e.elements;for(let c=0;c<16;c++)if(i[c]!==r[c])return!1;return!0}fromArray(e,i=0){for(let r=0;r<16;r++)this.elements[r]=e[r+i];return this}toArray(e=[],i=0){const r=this.elements;return e[i]=r[0],e[i+1]=r[1],e[i+2]=r[2],e[i+3]=r[3],e[i+4]=r[4],e[i+5]=r[5],e[i+6]=r[6],e[i+7]=r[7],e[i+8]=r[8],e[i+9]=r[9],e[i+10]=r[10],e[i+11]=r[11],e[i+12]=r[12],e[i+13]=r[13],e[i+14]=r[14],e[i+15]=r[15],e}};Vu.prototype.isMatrix4=!0;let nn=Vu;const Yr=new dt,Oi=new nn,ub=new dt(0,0,0),fb=new dt(1,1,1),ps=new dt,lu=new dt,di=new dt,q_=new nn,Y_=new _o;class Fa{constructor(e=0,i=0,r=0,c=Fa.DEFAULT_ORDER){this.isEuler=!0,this._x=e,this._y=i,this._z=r,this._order=c}get x(){return this._x}set x(e){this._x=e,this._onChangeCallback()}get y(){return this._y}set y(e){this._y=e,this._onChangeCallback()}get z(){return this._z}set z(e){this._z=e,this._onChangeCallback()}get order(){return this._order}set order(e){this._order=e,this._onChangeCallback()}set(e,i,r,c=this._order){return this._x=e,this._y=i,this._z=r,this._order=c,this._onChangeCallback(),this}clone(){return new this.constructor(this._x,this._y,this._z,this._order)}copy(e){return this._x=e._x,this._y=e._y,this._z=e._z,this._order=e._order,this._onChangeCallback(),this}setFromRotationMatrix(e,i=this._order,r=!0){const c=e.elements,f=c[0],d=c[4],h=c[8],m=c[1],p=c[5],x=c[9],v=c[2],_=c[6],M=c[10];switch(i){case"XYZ":this._y=Math.asin(De(h,-1,1)),Math.abs(h)<.9999999?(this._x=Math.atan2(-x,M),this._z=Math.atan2(-d,f)):(this._x=Math.atan2(_,p),this._z=0);break;case"YXZ":this._x=Math.asin(-De(x,-1,1)),Math.abs(x)<.9999999?(this._y=Math.atan2(h,M),this._z=Math.atan2(m,p)):(this._y=Math.atan2(-v,f),this._z=0);break;case"ZXY":this._x=Math.asin(De(_,-1,1)),Math.abs(_)<.9999999?(this._y=Math.atan2(-v,M),this._z=Math.atan2(-d,p)):(this._y=0,this._z=Math.atan2(m,f));break;case"ZYX":this._y=Math.asin(-De(v,-1,1)),Math.abs(v)<.9999999?(this._x=Math.atan2(_,M),this._z=Math.atan2(m,f)):(this._x=0,this._z=Math.atan2(-d,p));break;case"YZX":this._z=Math.asin(De(m,-1,1)),Math.abs(m)<.9999999?(this._x=Math.atan2(-x,p),this._y=Math.atan2(-v,f)):(this._x=0,this._y=Math.atan2(h,M));break;case"XZY":this._z=Math.asin(-De(d,-1,1)),Math.abs(d)<.9999999?(this._x=Math.atan2(_,p),this._y=Math.atan2(h,f)):(this._x=Math.atan2(-x,M),this._y=0);break;default:oe("Euler: .setFromRotationMatrix() encountered an unknown order: "+i)}return this._order=i,r===!0&&this._onChangeCallback(),this}setFromQuaternion(e,i,r){return q_.makeRotationFromQuaternion(e),this.setFromRotationMatrix(q_,i,r)}setFromVector3(e,i=this._order){return this.set(e.x,e.y,e.z,i)}reorder(e){return Y_.setFromEuler(this),this.setFromQuaternion(Y_,e)}equals(e){return e._x===this._x&&e._y===this._y&&e._z===this._z&&e._order===this._order}fromArray(e){return this._x=e[0],this._y=e[1],this._z=e[2],e[3]!==void 0&&(this._order=e[3]),this._onChangeCallback(),this}toArray(e=[],i=0){return e[i]=this._x,e[i+1]=this._y,e[i+2]=this._z,e[i+3]=this._order,e}_onChange(e){return this._onChangeCallback=e,this}_onChangeCallback(){}*[Symbol.iterator](){yield this._x,yield this._y,yield this._z,yield this._order}}Fa.DEFAULT_ORDER="XYZ";class rS{constructor(){this.mask=1}set(e){this.mask=(1<<e|0)>>>0}enable(e){this.mask|=1<<e|0}enableAll(){this.mask=-1}toggle(e){this.mask^=1<<e|0}disable(e){this.mask&=~(1<<e|0)}disableAll(){this.mask=0}test(e){return(this.mask&e.mask)!==0}isEnabled(e){return(this.mask&(1<<e|0))!==0}}let db=0;const j_=new dt,jr=new _o,Ca=new nn,cu=new dt,xl=new dt,hb=new dt,pb=new _o,Z_=new dt(1,0,0),K_=new dt(0,1,0),Q_=new dt(0,0,1),J_={type:"added"},mb={type:"removed"},Zr={type:"childadded",child:null},Ih={type:"childremoved",child:null};class Rn extends nr{constructor(){super(),this.isObject3D=!0,Object.defineProperty(this,"id",{value:db++}),this.uuid=Bl(),this.name="",this.type="Object3D",this.parent=null,this.children=[],this.up=Rn.DEFAULT_UP.clone();const e=new dt,i=new Fa,r=new _o,c=new dt(1,1,1);function f(){r.setFromEuler(i,!1)}function d(){i.setFromQuaternion(r,void 0,!1)}i._onChange(f),r._onChange(d),Object.defineProperties(this,{position:{configurable:!0,enumerable:!0,value:e},rotation:{configurable:!0,enumerable:!0,value:i},quaternion:{configurable:!0,enumerable:!0,value:r},scale:{configurable:!0,enumerable:!0,value:c},modelViewMatrix:{value:new nn},normalMatrix:{value:new de}}),this.matrix=new nn,this.matrixWorld=new nn,this.matrixAutoUpdate=Rn.DEFAULT_MATRIX_AUTO_UPDATE,this.matrixWorldAutoUpdate=Rn.DEFAULT_MATRIX_WORLD_AUTO_UPDATE,this.matrixWorldNeedsUpdate=!1,this.layers=new rS,this.visible=!0,this.castShadow=!1,this.receiveShadow=!1,this.frustumCulled=!0,this.renderOrder=0,this.animations=[],this.customDepthMaterial=void 0,this.customDistanceMaterial=void 0,this.static=!1,this.userData={},this.pivot=null}onBeforeShadow(){}onAfterShadow(){}onBeforeRender(){}onAfterRender(){}applyMatrix4(e){this.matrixAutoUpdate&&this.updateMatrix(),this.matrix.premultiply(e),this.matrix.decompose(this.position,this.quaternion,this.scale)}applyQuaternion(e){return this.quaternion.premultiply(e),this}setRotationFromAxisAngle(e,i){this.quaternion.setFromAxisAngle(e,i)}setRotationFromEuler(e){this.quaternion.setFromEuler(e,!0)}setRotationFromMatrix(e){this.quaternion.setFromRotationMatrix(e)}setRotationFromQuaternion(e){this.quaternion.copy(e)}rotateOnAxis(e,i){return jr.setFromAxisAngle(e,i),this.quaternion.multiply(jr),this}rotateOnWorldAxis(e,i){return jr.setFromAxisAngle(e,i),this.quaternion.premultiply(jr),this}rotateX(e){return this.rotateOnAxis(Z_,e)}rotateY(e){return this.rotateOnAxis(K_,e)}rotateZ(e){return this.rotateOnAxis(Q_,e)}translateOnAxis(e,i){return j_.copy(e).applyQuaternion(this.quaternion),this.position.add(j_.multiplyScalar(i)),this}translateX(e){return this.translateOnAxis(Z_,e)}translateY(e){return this.translateOnAxis(K_,e)}translateZ(e){return this.translateOnAxis(Q_,e)}localToWorld(e){return this.updateWorldMatrix(!0,!1),e.applyMatrix4(this.matrixWorld)}worldToLocal(e){return this.updateWorldMatrix(!0,!1),e.applyMatrix4(Ca.copy(this.matrixWorld).invert())}lookAt(e,i,r){e.isVector3?cu.copy(e):cu.set(e,i,r);const c=this.parent;this.updateWorldMatrix(!0,!1),xl.setFromMatrixPosition(this.matrixWorld),this.isCamera||this.isLight?Ca.lookAt(xl,cu,this.up):Ca.lookAt(cu,xl,this.up),this.quaternion.setFromRotationMatrix(Ca),c&&(Ca.extractRotation(c.matrixWorld),jr.setFromRotationMatrix(Ca),this.quaternion.premultiply(jr.invert()))}add(e){if(arguments.length>1){for(let i=0;i<arguments.length;i++)this.add(arguments[i]);return this}return e===this?(ze("Object3D.add: object can't be added as a child of itself.",e),this):(e&&e.isObject3D?(e.removeFromParent(),e.parent=this,this.children.push(e),e.dispatchEvent(J_),Zr.child=e,this.dispatchEvent(Zr),Zr.child=null):ze("Object3D.add: object not an instance of THREE.Object3D.",e),this)}remove(e){if(arguments.length>1){for(let r=0;r<arguments.length;r++)this.remove(arguments[r]);return this}const i=this.children.indexOf(e);return i!==-1&&(e.parent=null,this.children.splice(i,1),e.dispatchEvent(mb),Ih.child=e,this.dispatchEvent(Ih),Ih.child=null),this}removeFromParent(){const e=this.parent;return e!==null&&e.remove(this),this}clear(){return this.remove(...this.children)}attach(e){return this.updateWorldMatrix(!0,!1),Ca.copy(this.matrixWorld).invert(),e.parent!==null&&(e.parent.updateWorldMatrix(!0,!1),Ca.multiply(e.parent.matrixWorld)),e.applyMatrix4(Ca),e.removeFromParent(),e.parent=this,this.children.push(e),e.updateWorldMatrix(!1,!0),e.dispatchEvent(J_),Zr.child=e,this.dispatchEvent(Zr),Zr.child=null,this}getObjectById(e){return this.getObjectByProperty("id",e)}getObjectByName(e){return this.getObjectByProperty("name",e)}getObjectByProperty(e,i){if(this[e]===i)return this;for(let r=0,c=this.children.length;r<c;r++){const d=this.children[r].getObjectByProperty(e,i);if(d!==void 0)return d}}getObjectsByProperty(e,i,r=[]){this[e]===i&&r.push(this);const c=this.children;for(let f=0,d=c.length;f<d;f++)c[f].getObjectsByProperty(e,i,r);return r}getWorldPosition(e){return this.updateWorldMatrix(!0,!1),e.setFromMatrixPosition(this.matrixWorld)}getWorldQuaternion(e){return this.updateWorldMatrix(!0,!1),this.matrixWorld.decompose(xl,e,hb),e}getWorldScale(e){return this.updateWorldMatrix(!0,!1),this.matrixWorld.decompose(xl,pb,e),e}getWorldDirection(e){this.updateWorldMatrix(!0,!1);const i=this.matrixWorld.elements;return e.set(i[8],i[9],i[10]).normalize()}raycast(){}intersectsFrustum(){}traverse(e){e(this);const i=this.children;for(let r=0,c=i.length;r<c;r++)i[r].traverse(e)}traverseVisible(e){if(this.visible===!1)return;e(this);const i=this.children;for(let r=0,c=i.length;r<c;r++)i[r].traverseVisible(e)}traverseAncestors(e){const i=this.parent;i!==null&&(e(i),i.traverseAncestors(e))}updateMatrix(){this.matrix.compose(this.position,this.quaternion,this.scale);const e=this.pivot;if(e!==null){const i=e.x,r=e.y,c=e.z,f=this.matrix.elements;f[12]+=i-f[0]*i-f[4]*r-f[8]*c,f[13]+=r-f[1]*i-f[5]*r-f[9]*c,f[14]+=c-f[2]*i-f[6]*r-f[10]*c}this.matrixWorldNeedsUpdate=!0}updateMatrixWorld(e){this.matrixAutoUpdate&&this.updateMatrix(),(this.matrixWorldNeedsUpdate||e)&&(this.matrixWorldAutoUpdate===!0&&(this.parent===null?this.matrixWorld.copy(this.matrix):this.matrixWorld.multiplyMatrices(this.parent.matrixWorld,this.matrix)),this.matrixWorldNeedsUpdate=!1,e=!0);const i=this.children;for(let r=0,c=i.length;r<c;r++)i[r].updateMatrixWorld(e)}updateWorldMatrix(e,i,r=!1){const c=this.parent;if(e===!0&&c!==null&&c.updateWorldMatrix(!0,!1),this.matrixAutoUpdate&&this.updateMatrix(),(this.matrixWorldNeedsUpdate||r)&&(this.matrixWorldAutoUpdate===!0&&(this.parent===null?this.matrixWorld.copy(this.matrix):this.matrixWorld.multiplyMatrices(this.parent.matrixWorld,this.matrix)),this.matrixWorldNeedsUpdate=!1,r=!0),i===!0){const f=this.children;for(let d=0,h=f.length;d<h;d++)f[d].updateWorldMatrix(!1,!0,r)}}toJSON(e){const i=e===void 0||typeof e=="string",r={};i&&(e={geometries:{},materials:{},textures:{},images:{},shapes:{},skeletons:{},animations:{},nodes:{}},r.metadata={version:4.7,type:"Object",generator:"Object3D.toJSON"});const c={};c.uuid=this.uuid,c.type=this.type,c.name=this.name,c.castShadow=this.castShadow,c.receiveShadow=this.receiveShadow,c.visible=this.visible,c.frustumCulled=this.frustumCulled,c.renderOrder=this.renderOrder,c.static=this.static,c.matrixAutoUpdate=this.matrixAutoUpdate,Object.keys(this.userData).length>0&&(c.userData=this.userData),c.layers=this.layers.mask,c.matrix=this.matrix.toArray(),c.up=this.up.toArray(),this.pivot!==null&&(c.pivot=this.pivot.toArray()),this.morphTargetDictionary!==void 0&&(c.morphTargetDictionary=Object.assign({},this.morphTargetDictionary)),this.morphTargetInfluences!==void 0&&(c.morphTargetInfluences=this.morphTargetInfluences.slice()),this.isInstancedMesh&&(c.type="InstancedMesh",c.count=this.count,c.instanceMatrix=this.instanceMatrix.toJSON(),this.instanceColor!==null&&(c.instanceColor=this.instanceColor.toJSON())),this.isBatchedMesh&&(c.type="BatchedMesh",c.perObjectFrustumCulled=this.perObjectFrustumCulled,c.sortObjects=this.sortObjects,c.drawRanges=this._drawRanges,c.reservedRanges=this._reservedRanges,c.geometryInfo=this._geometryInfo.map(h=>({...h,boundingBox:h.boundingBox?h.boundingBox.toJSON():void 0,boundingSphere:h.boundingSphere?h.boundingSphere.toJSON():void 0})),c.instanceInfo=this._instanceInfo.map(h=>({...h})),c.availableInstanceIds=this._availableInstanceIds.slice(),c.availableGeometryIds=this._availableGeometryIds.slice(),c.nextIndexStart=this._nextIndexStart,c.nextVertexStart=this._nextVertexStart,c.geometryCount=this._geometryCount,c.maxInstanceCount=this._maxInstanceCount,c.maxVertexCount=this._maxVertexCount,c.maxIndexCount=this._maxIndexCount,c.geometryInitialized=this._geometryInitialized,c.matricesTexture=this._matricesTexture.toJSON(e),c.indirectTexture=this._indirectTexture.toJSON(e),this._colorsTexture!==null&&(c.colorsTexture=this._colorsTexture.toJSON(e)),this.boundingSphere!==null&&(c.boundingSphere=this.boundingSphere.toJSON()),this.boundingBox!==null&&(c.boundingBox=this.boundingBox.toJSON()));function f(h,m){return h[m.uuid]===void 0&&(h[m.uuid]=m.toJSON(e)),m.uuid}if(this.isScene)this.background&&(this.background.isColor?c.background=this.background.toJSON():this.background.isTexture&&(c.background=this.background.toJSON(e).uuid)),this.environment&&this.environment.isTexture&&this.environment.isRenderTargetTexture!==!0&&(c.environment=this.environment.toJSON(e).uuid);else if(this.isMesh||this.isLine||this.isPoints){c.geometry=f(e.geometries,this.geometry);const h=this.geometry.parameters;if(h!==void 0&&h.shapes!==void 0){const m=h.shapes;if(Array.isArray(m))for(let p=0,x=m.length;p<x;p++){const v=m[p];f(e.shapes,v)}else f(e.shapes,m)}}if(this.isSkinnedMesh&&(c.bindMode=this.bindMode,c.bindMatrix=this.bindMatrix.toArray(),this.skeleton!==void 0&&(f(e.skeletons,this.skeleton),c.skeleton=this.skeleton.uuid)),this.material!==void 0)if(Array.isArray(this.material)){const h=[];for(let m=0,p=this.material.length;m<p;m++)h.push(f(e.materials,this.material[m]));c.material=h}else c.material=f(e.materials,this.material);if(this.children.length>0){c.children=[];for(let h=0;h<this.children.length;h++)c.children.push(this.children[h].toJSON(e).object)}if(this.animations.length>0){c.animations=[];for(let h=0;h<this.animations.length;h++){const m=this.animations[h];c.animations.push(f(e.animations,m))}}if(i){const h=d(e.geometries),m=d(e.materials),p=d(e.textures),x=d(e.images),v=d(e.shapes),_=d(e.skeletons),M=d(e.animations),R=d(e.nodes);h.length>0&&(r.geometries=h),m.length>0&&(r.materials=m),p.length>0&&(r.textures=p),x.length>0&&(r.images=x),v.length>0&&(r.shapes=v),_.length>0&&(r.skeletons=_),M.length>0&&(r.animations=M),R.length>0&&(r.nodes=R)}return r.object=c,r;function d(h){const m=[];for(const p in h){const x=h[p];delete x.metadata,m.push(x)}return m}}clone(e){return new this.constructor().copy(this,e)}copy(e,i=!0){if(this.name=e.name,this.up.copy(e.up),this.position.copy(e.position),this.rotation.order=e.rotation.order,this.quaternion.copy(e.quaternion),this.scale.copy(e.scale),this.pivot=e.pivot!==null?e.pivot.clone():null,this.matrix.copy(e.matrix),this.matrixWorld.copy(e.matrixWorld),this.matrixAutoUpdate=e.matrixAutoUpdate,this.matrixWorldAutoUpdate=e.matrixWorldAutoUpdate,this.matrixWorldNeedsUpdate=e.matrixWorldNeedsUpdate,this.layers.mask=e.layers.mask,this.visible=e.visible,this.castShadow=e.castShadow,this.receiveShadow=e.receiveShadow,this.frustumCulled=e.frustumCulled,this.renderOrder=e.renderOrder,this.static=e.static,this.animations=e.animations.slice(),this.userData=JSON.parse(JSON.stringify(e.userData)),i===!0)for(let r=0;r<e.children.length;r++){const c=e.children[r];this.add(c.clone())}return this}dispose(){this.dispatchEvent({type:"dispose"})}}Rn.DEFAULT_UP=new dt(0,1,0);Rn.DEFAULT_MATRIX_AUTO_UPDATE=!0;Rn.DEFAULT_MATRIX_WORLD_AUTO_UPDATE=!0;class lo extends Rn{constructor(){super(),this.isGroup=!0,this.type="Group"}}const gb={type:"move"};class Bh{constructor(){this._targetRay=null,this._grip=null,this._hand=null}getHandSpace(){return this._hand===null&&(this._hand=new lo,this._hand.matrixAutoUpdate=!1,this._hand.visible=!1,this._hand.joints={},this._hand.inputState={pinching:!1}),this._hand}getTargetRaySpace(){return this._targetRay===null&&(this._targetRay=new lo,this._targetRay.matrixAutoUpdate=!1,this._targetRay.visible=!1,this._targetRay.hasLinearVelocity=!1,this._targetRay.linearVelocity=new dt,this._targetRay.hasAngularVelocity=!1,this._targetRay.angularVelocity=new dt),this._targetRay}getGripSpace(){return this._grip===null&&(this._grip=new lo,this._grip.matrixAutoUpdate=!1,this._grip.visible=!1,this._grip.hasLinearVelocity=!1,this._grip.linearVelocity=new dt,this._grip.hasAngularVelocity=!1,this._grip.angularVelocity=new dt,this._grip.eventsEnabled=!1),this._grip}dispatchEvent(e){return this._targetRay!==null&&this._targetRay.dispatchEvent(e),this._grip!==null&&this._grip.dispatchEvent(e),this._hand!==null&&this._hand.dispatchEvent(e),this}connect(e){if(e&&e.hand){const i=this._hand;if(i)for(const r of e.hand.values())this._getHandJoint(i,r)}return this.dispatchEvent({type:"connected",data:e}),this}disconnect(e){return this.dispatchEvent({type:"disconnected",data:e}),this._targetRay!==null&&(this._targetRay.visible=!1),this._grip!==null&&(this._grip.visible=!1),this._hand!==null&&(this._hand.visible=!1),this}update(e,i,r){let c=null,f=null,d=null;const h=this._targetRay,m=this._grip,p=this._hand;if(e&&i.session.visibilityState!=="visible-blurred"){if(p&&e.hand){d=!0;for(const N of e.hand.values()){const y=i.getJointPose(N,r),S=this._getHandJoint(p,N);y!==null&&(S.matrix.fromArray(y.transform.matrix),S.matrix.decompose(S.position,S.rotation,S.scale),S.matrixWorldNeedsUpdate=!0,S.jointRadius=y.radius),S.visible=y!==null}const x=p.joints["index-finger-tip"],v=p.joints["thumb-tip"],_=x.position.distanceTo(v.position),M=.02,R=.005;p.inputState.pinching&&_>M+R?(p.inputState.pinching=!1,this.dispatchEvent({type:"pinchend",handedness:e.handedness,target:this})):!p.inputState.pinching&&_<=M-R&&(p.inputState.pinching=!0,this.dispatchEvent({type:"pinchstart",handedness:e.handedness,target:this}))}else m!==null&&e.gripSpace&&(f=i.getPose(e.gripSpace,r),f!==null&&(m.matrix.fromArray(f.transform.matrix),m.matrix.decompose(m.position,m.rotation,m.scale),m.matrixWorldNeedsUpdate=!0,f.linearVelocity?(m.hasLinearVelocity=!0,m.linearVelocity.copy(f.linearVelocity)):m.hasLinearVelocity=!1,f.angularVelocity?(m.hasAngularVelocity=!0,m.angularVelocity.copy(f.angularVelocity)):m.hasAngularVelocity=!1,m.eventsEnabled&&m.dispatchEvent({type:"gripUpdated",data:e,target:this})));h!==null&&(c=i.getPose(e.targetRaySpace,r),c===null&&f!==null&&(c=f),c!==null&&(h.matrix.fromArray(c.transform.matrix),h.matrix.decompose(h.position,h.rotation,h.scale),h.matrixWorldNeedsUpdate=!0,c.linearVelocity?(h.hasLinearVelocity=!0,h.linearVelocity.copy(c.linearVelocity)):h.hasLinearVelocity=!1,c.angularVelocity?(h.hasAngularVelocity=!0,h.angularVelocity.copy(c.angularVelocity)):h.hasAngularVelocity=!1,this.dispatchEvent(gb)))}return h!==null&&(h.visible=c!==null),m!==null&&(m.visible=f!==null),p!==null&&(p.visible=d!==null),this}_getHandJoint(e,i){if(e.joints[i.jointName]===void 0){const r=new lo;r.matrixAutoUpdate=!1,r.visible=!1,e.joints[i.jointName]=r,e.add(r)}return e.joints[i.jointName]}}const oS={aliceblue:15792383,antiquewhite:16444375,aqua:65535,aquamarine:8388564,azure:15794175,beige:16119260,bisque:16770244,black:0,blanchedalmond:16772045,blue:255,blueviolet:9055202,brown:10824234,burlywood:14596231,cadetblue:6266528,chartreuse:8388352,chocolate:13789470,coral:16744272,cornflowerblue:6591981,cornsilk:16775388,crimson:14423100,cyan:65535,darkblue:139,darkcyan:35723,darkgoldenrod:12092939,darkgray:11119017,darkgreen:25600,darkgrey:11119017,darkkhaki:12433259,darkmagenta:9109643,darkolivegreen:5597999,darkorange:16747520,darkorchid:10040012,darkred:9109504,darksalmon:15308410,darkseagreen:9419919,darkslateblue:4734347,darkslategray:3100495,darkslategrey:3100495,darkturquoise:52945,darkviolet:9699539,deeppink:16716947,deepskyblue:49151,dimgray:6908265,dimgrey:6908265,dodgerblue:2003199,firebrick:11674146,floralwhite:16775920,forestgreen:2263842,fuchsia:16711935,gainsboro:14474460,ghostwhite:16316671,gold:16766720,goldenrod:14329120,gray:8421504,green:32768,greenyellow:11403055,grey:8421504,honeydew:15794160,hotpink:16738740,indianred:13458524,indigo:4915330,ivory:16777200,khaki:15787660,lavender:15132410,lavenderblush:16773365,lawngreen:8190976,lemonchiffon:16775885,lightblue:11393254,lightcoral:15761536,lightcyan:14745599,lightgoldenrodyellow:16448210,lightgray:13882323,lightgreen:9498256,lightgrey:13882323,lightpink:16758465,lightsalmon:16752762,lightseagreen:2142890,lightskyblue:8900346,lightslategray:7833753,lightslategrey:7833753,lightsteelblue:11584734,lightyellow:16777184,lime:65280,limegreen:3329330,linen:16445670,magenta:16711935,maroon:8388608,mediumaquamarine:6737322,mediumblue:205,mediumorchid:12211667,mediumpurple:9662683,mediumseagreen:3978097,mediumslateblue:8087790,mediumspringgreen:64154,mediumturquoise:4772300,mediumvioletred:13047173,midnightblue:1644912,mintcream:16121850,mistyrose:16770273,moccasin:16770229,navajowhite:16768685,navy:128,oldlace:16643558,olive:8421376,olivedrab:7048739,orange:16753920,orangered:16729344,orchid:14315734,palegoldenrod:15657130,palegreen:10025880,paleturquoise:11529966,palevioletred:14381203,papayawhip:16773077,peachpuff:16767673,peru:13468991,pink:16761035,plum:14524637,powderblue:11591910,purple:8388736,rebeccapurple:6697881,red:16711680,rosybrown:12357519,royalblue:4286945,saddlebrown:9127187,salmon:16416882,sandybrown:16032864,seagreen:3050327,seashell:16774638,sienna:10506797,silver:12632256,skyblue:8900331,slateblue:6970061,slategray:7372944,slategrey:7372944,snow:16775930,springgreen:65407,steelblue:4620980,tan:13808780,teal:32896,thistle:14204888,tomato:16737095,turquoise:4251856,violet:15631086,wheat:16113331,white:16777215,whitesmoke:16119285,yellow:16776960,yellowgreen:10145074},ms={h:0,s:0,l:0},uu={h:0,s:0,l:0};function zh(o,e,i){return i<0&&(i+=1),i>1&&(i-=1),i<1/6?o+(e-o)*6*i:i<1/2?e:i<2/3?o+(e-o)*6*(2/3-i):o}class he{constructor(e,i,r){return this.isColor=!0,this.r=1,this.g=1,this.b=1,this.set(e,i,r)}set(e,i,r){if(i===void 0&&r===void 0){const c=e;c&&c.isColor?this.copy(c):typeof c=="number"?this.setHex(c):typeof c=="string"&&this.setStyle(c)}else this.setRGB(e,i,r);return this}setScalar(e){return this.r=e,this.g=e,this.b=e,this}setHex(e,i=ti){return e=Math.floor(e),this.r=(e>>16&255)/255,this.g=(e>>8&255)/255,this.b=(e&255)/255,Pe.colorSpaceToWorking(this,i),this}setRGB(e,i,r,c=Pe.workingColorSpace){return this.r=e,this.g=i,this.b=r,Pe.colorSpaceToWorking(this,c),this}setHSL(e,i,r,c=Pe.workingColorSpace){if(e=ib(e,1),i=De(i,0,1),r=De(r,0,1),i===0)this.r=this.g=this.b=r;else{const f=r<=.5?r*(1+i):r+i-r*i,d=2*r-f;this.r=zh(d,f,e+1/3),this.g=zh(d,f,e),this.b=zh(d,f,e-1/3)}return Pe.colorSpaceToWorking(this,c),this}setStyle(e,i=ti){function r(f){f!==void 0&&parseFloat(f)<1&&oe("Color: Alpha component of "+e+" will be ignored.")}let c;if(c=/^(\w+)\(([^\)]*)\)/.exec(e)){let f;const d=c[1],h=c[2];switch(d){case"rgb":case"rgba":if(f=/^\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*(?:,\s*(\d*\.?\d+)\s*)?$/.exec(h))return r(f[4]),this.setRGB(Math.min(255,parseInt(f[1],10))/255,Math.min(255,parseInt(f[2],10))/255,Math.min(255,parseInt(f[3],10))/255,i);if(f=/^\s*(\d+)\%\s*,\s*(\d+)\%\s*,\s*(\d+)\%\s*(?:,\s*(\d*\.?\d+)\s*)?$/.exec(h))return r(f[4]),this.setRGB(Math.min(100,parseInt(f[1],10))/100,Math.min(100,parseInt(f[2],10))/100,Math.min(100,parseInt(f[3],10))/100,i);break;case"hsl":case"hsla":if(f=/^\s*(\d*\.?\d+)\s*,\s*(\d*\.?\d+)\%\s*,\s*(\d*\.?\d+)\%\s*(?:,\s*(\d*\.?\d+)\s*)?$/.exec(h))return r(f[4]),this.setHSL(parseFloat(f[1])/360,parseFloat(f[2])/100,parseFloat(f[3])/100,i);break;default:oe("Color: Unknown color model "+e)}}else if(c=/^\#([A-Fa-f\d]+)$/.exec(e)){const f=c[1],d=f.length;if(d===3)return this.setRGB(parseInt(f.charAt(0),16)/15,parseInt(f.charAt(1),16)/15,parseInt(f.charAt(2),16)/15,i);if(d===6)return this.setHex(parseInt(f,16),i);oe("Color: Invalid hex color "+e)}else if(e&&e.length>0)return this.setColorName(e,i);return this}setColorName(e,i=ti){const r=oS[e.toLowerCase()];return r!==void 0?this.setHex(r,i):oe("Color: Unknown color "+e),this}clone(){return new this.constructor(this.r,this.g,this.b)}copy(e){return this.r=e.r,this.g=e.g,this.b=e.b,this}copySRGBToLinear(e){return this.r=Ba(e.r),this.g=Ba(e.g),this.b=Ba(e.b),this}copyLinearToSRGB(e){return this.r=ho(e.r),this.g=ho(e.g),this.b=ho(e.b),this}convertSRGBToLinear(){return this.copySRGBToLinear(this),this}convertLinearToSRGB(){return this.copyLinearToSRGB(this),this}getHex(e=ti){return Pe.workingToColorSpace(Bn.copy(this),e),Math.round(De(Bn.r*255,0,255))*65536+Math.round(De(Bn.g*255,0,255))*256+Math.round(De(Bn.b*255,0,255))}getHexString(e=ti){return("000000"+this.getHex(e).toString(16)).slice(-6)}getHSL(e,i=Pe.workingColorSpace){Pe.workingToColorSpace(Bn.copy(this),i);const r=Bn.r,c=Bn.g,f=Bn.b,d=Math.max(r,c,f),h=Math.min(r,c,f);let m,p;const x=(h+d)/2;if(h===d)m=0,p=0;else{const v=d-h;switch(p=x<=.5?v/(d+h):v/(2-d-h),d){case r:m=(c-f)/v+(c<f?6:0);break;case c:m=(f-r)/v+2;break;case f:m=(r-c)/v+4;break}m/=6}return e.h=m,e.s=p,e.l=x,e}getRGB(e,i=Pe.workingColorSpace){return Pe.workingToColorSpace(Bn.copy(this),i),e.r=Bn.r,e.g=Bn.g,e.b=Bn.b,e}getStyle(e=ti){Pe.workingToColorSpace(Bn.copy(this),e);const i=Bn.r,r=Bn.g,c=Bn.b;return e!==ti?`color(${e} ${i.toFixed(3)} ${r.toFixed(3)} ${c.toFixed(3)})`:`rgb(${Math.round(i*255)},${Math.round(r*255)},${Math.round(c*255)})`}offsetHSL(e,i,r){return this.getHSL(ms),this.setHSL(ms.h+e,ms.s+i,ms.l+r)}add(e){return this.r+=e.r,this.g+=e.g,this.b+=e.b,this}addColors(e,i){return this.r=e.r+i.r,this.g=e.g+i.g,this.b=e.b+i.b,this}addScalar(e){return this.r+=e,this.g+=e,this.b+=e,this}sub(e){return this.r=Math.max(0,this.r-e.r),this.g=Math.max(0,this.g-e.g),this.b=Math.max(0,this.b-e.b),this}multiply(e){return this.r*=e.r,this.g*=e.g,this.b*=e.b,this}multiplyScalar(e){return this.r*=e,this.g*=e,this.b*=e,this}lerp(e,i){return this.r+=(e.r-this.r)*i,this.g+=(e.g-this.g)*i,this.b+=(e.b-this.b)*i,this}lerpColors(e,i,r){return this.r=e.r+(i.r-e.r)*r,this.g=e.g+(i.g-e.g)*r,this.b=e.b+(i.b-e.b)*r,this}lerpHSL(e,i){this.getHSL(ms),e.getHSL(uu);const r=Dh(ms.h,uu.h,i),c=Dh(ms.s,uu.s,i),f=Dh(ms.l,uu.l,i);return this.setHSL(r,c,f),this}setFromVector3(e){return this.r=e.x,this.g=e.y,this.b=e.z,this}applyMatrix3(e){const i=this.r,r=this.g,c=this.b,f=e.elements;return this.r=f[0]*i+f[3]*r+f[6]*c,this.g=f[1]*i+f[4]*r+f[7]*c,this.b=f[2]*i+f[5]*r+f[8]*c,this}equals(e){return e.r===this.r&&e.g===this.g&&e.b===this.b}fromArray(e,i=0){return this.r=e[i],this.g=e[i+1],this.b=e[i+2],this}toArray(e=[],i=0){return e[i]=this.r,e[i+1]=this.g,e[i+2]=this.b,e}fromBufferAttribute(e,i){return this.r=e.getX(i),this.g=e.getY(i),this.b=e.getZ(i),this}toJSON(){return this.getHex()}*[Symbol.iterator](){yield this.r,yield this.g,yield this.b}}const Bn=new he;he.NAMES=oS;class fm extends Rn{constructor(){super(),this.isScene=!0,this.type="Scene",this.background=null,this.environment=null,this.fog=null,this.backgroundBlurriness=0,this.backgroundIntensity=1,this.backgroundRotation=new Fa,this.environmentIntensity=1,this.environmentRotation=new Fa,this.overrideMaterial=null,typeof __THREE_DEVTOOLS__<"u"&&__THREE_DEVTOOLS__.dispatchEvent(new CustomEvent("observe",{detail:this}))}copy(e,i){return super.copy(e,i),e.background!==null&&(this.background=e.background.clone()),e.environment!==null&&(this.environment=e.environment.clone()),e.fog!==null&&(this.fog=e.fog.clone()),this.backgroundBlurriness=e.backgroundBlurriness,this.backgroundIntensity=e.backgroundIntensity,this.backgroundRotation.copy(e.backgroundRotation),this.environmentIntensity=e.environmentIntensity,this.environmentRotation.copy(e.environmentRotation),e.overrideMaterial!==null&&(this.overrideMaterial=e.overrideMaterial.clone()),this.matrixAutoUpdate=e.matrixAutoUpdate,this}toJSON(e){const i=super.toJSON(e);return this.fog!==null&&(i.object.fog=this.fog.toJSON()),i.object.backgroundBlurriness=this.backgroundBlurriness,i.object.backgroundIntensity=this.backgroundIntensity,i.object.backgroundRotation=this.backgroundRotation.toArray(),i.object.environmentIntensity=this.environmentIntensity,i.object.environmentRotation=this.environmentRotation.toArray(),i}}const Pi=new dt,Na=new dt,Fh=new dt,Da=new dt,Kr=new dt,Qr=new dt,$_=new dt,Hh=new dt,Gh=new dt,Vh=new dt,kh=new on,Xh=new on,Wh=new on;class zi{constructor(e=new dt,i=new dt,r=new dt){this.a=e,this.b=i,this.c=r}static getNormal(e,i,r,c){c.subVectors(r,i),Pi.subVectors(e,i),c.cross(Pi);const f=c.lengthSq();return f>0?c.multiplyScalar(1/Math.sqrt(f)):c.set(0,0,0)}static getBarycoord(e,i,r,c,f){Pi.subVectors(c,i),Na.subVectors(r,i),Fh.subVectors(e,i);const d=Pi.dot(Pi),h=Pi.dot(Na),m=Pi.dot(Fh),p=Na.dot(Na),x=Na.dot(Fh),v=d*p-h*h;if(v===0)return f.set(0,0,0),null;const _=1/v,M=(p*m-h*x)*_,R=(d*x-h*m)*_;return f.set(1-M-R,R,M)}static containsPoint(e,i,r,c){return this.getBarycoord(e,i,r,c,Da)===null?!1:Da.x>=0&&Da.y>=0&&Da.x+Da.y<=1}static getInterpolation(e,i,r,c,f,d,h,m){return this.getBarycoord(e,i,r,c,Da)===null?(m.x=0,m.y=0,"z"in m&&(m.z=0),"w"in m&&(m.w=0),null):(m.setScalar(0),m.addScaledVector(f,Da.x),m.addScaledVector(d,Da.y),m.addScaledVector(h,Da.z),m)}static getInterpolatedAttribute(e,i,r,c,f,d){return kh.setScalar(0),Xh.setScalar(0),Wh.setScalar(0),kh.fromBufferAttribute(e,i),Xh.fromBufferAttribute(e,r),Wh.fromBufferAttribute(e,c),d.setScalar(0),d.addScaledVector(kh,f.x),d.addScaledVector(Xh,f.y),d.addScaledVector(Wh,f.z),d}static isFrontFacing(e,i,r,c){return Pi.subVectors(r,i),Na.subVectors(e,i),Pi.cross(Na).dot(c)<0}set(e,i,r){return this.a.copy(e),this.b.copy(i),this.c.copy(r),this}setFromPointsAndIndices(e,i,r,c){return this.a.copy(e[i]),this.b.copy(e[r]),this.c.copy(e[c]),this}setFromAttributeAndIndices(e,i,r,c){return this.a.fromBufferAttribute(e,i),this.b.fromBufferAttribute(e,r),this.c.fromBufferAttribute(e,c),this}clone(){return new this.constructor().copy(this)}copy(e){return this.a.copy(e.a),this.b.copy(e.b),this.c.copy(e.c),this}getArea(){return Pi.subVectors(this.c,this.b),Na.subVectors(this.a,this.b),Pi.cross(Na).length()*.5}getMidpoint(e){return e.addVectors(this.a,this.b).add(this.c).multiplyScalar(1/3)}getNormal(e){return zi.getNormal(this.a,this.b,this.c,e)}getPlane(e){return e.setFromCoplanarPoints(this.a,this.b,this.c)}getBarycoord(e,i){return zi.getBarycoord(e,this.a,this.b,this.c,i)}getInterpolation(e,i,r,c,f){return zi.getInterpolation(e,this.a,this.b,this.c,i,r,c,f)}containsPoint(e){return zi.containsPoint(e,this.a,this.b,this.c)}isFrontFacing(e){return zi.isFrontFacing(this.a,this.b,this.c,e)}intersectsBox(e){return e.intersectsTriangle(this)}closestPointToPoint(e,i){const r=this.a,c=this.b,f=this.c;let d,h;Kr.subVectors(c,r),Qr.subVectors(f,r),Hh.subVectors(e,r);const m=Kr.dot(Hh),p=Qr.dot(Hh);if(m<=0&&p<=0)return i.copy(r);Gh.subVectors(e,c);const x=Kr.dot(Gh),v=Qr.dot(Gh);if(x>=0&&v<=x)return i.copy(c);const _=m*v-x*p;if(_<=0&&m>=0&&x<=0)return d=m/(m-x),i.copy(r).addScaledVector(Kr,d);Vh.subVectors(e,f);const M=Kr.dot(Vh),R=Qr.dot(Vh);if(R>=0&&M<=R)return i.copy(f);const N=M*p-m*R;if(N<=0&&p>=0&&R<=0)return h=p/(p-R),i.copy(r).addScaledVector(Qr,h);const y=x*R-M*v;if(y<=0&&v-x>=0&&M-R>=0)return $_.subVectors(f,c),h=(v-x)/(v-x+(M-R)),i.copy(c).addScaledVector($_,h);const S=1/(y+N+_);return d=N*S,h=_*S,i.copy(r).addScaledVector(Kr,d).addScaledVector(Qr,h)}equals(e){return e.a.equals(this.a)&&e.b.equals(this.b)&&e.c.equals(this.c)}}class ir{constructor(e=new dt(1/0,1/0,1/0),i=new dt(-1/0,-1/0,-1/0)){this.isBox3=!0,this.min=e,this.max=i}set(e,i){return this.min.copy(e),this.max.copy(i),this}setFromArray(e){this.makeEmpty();for(let i=0,r=e.length;i<r;i+=3)this.expandByPoint(Ii.fromArray(e,i));return this}setFromBufferAttribute(e){this.makeEmpty();for(let i=0,r=e.count;i<r;i++)this.expandByPoint(Ii.fromBufferAttribute(e,i));return this}setFromPoints(e){this.makeEmpty();for(let i=0,r=e.length;i<r;i++)this.expandByPoint(e[i]);return this}setFromCenterAndSize(e,i){const r=Ii.copy(i).multiplyScalar(.5);return this.min.copy(e).sub(r),this.max.copy(e).add(r),this}setFromObject(e,i=!1){return this.makeEmpty(),this.expandByObject(e,i)}clone(){return new this.constructor().copy(this)}copy(e){return this.min.copy(e.min),this.max.copy(e.max),this}makeEmpty(){return this.min.x=this.min.y=this.min.z=1/0,this.max.x=this.max.y=this.max.z=-1/0,this}isEmpty(){return this.max.x<this.min.x||this.max.y<this.min.y||this.max.z<this.min.z}getCenter(e){return this.isEmpty()?e.set(0,0,0):e.addVectors(this.min,this.max).multiplyScalar(.5)}getSize(e){return this.isEmpty()?e.set(0,0,0):e.subVectors(this.max,this.min)}expandByPoint(e){return this.min.min(e),this.max.max(e),this}expandByVector(e){return this.min.sub(e),this.max.add(e),this}expandByScalar(e){return this.min.addScalar(-e),this.max.addScalar(e),this}expandByObject(e,i=!1){e.updateWorldMatrix(!1,!1);const r=e.geometry;if(r!==void 0){const f=r.getAttribute("position");if(i===!0&&f!==void 0&&e.isInstancedMesh!==!0)for(let d=0,h=f.count;d<h;d++)e.isMesh===!0?e.getVertexPosition(d,Ii):Ii.fromBufferAttribute(f,d),Ii.applyMatrix4(e.matrixWorld),this.expandByPoint(Ii);else e.boundingBox!==void 0?(e.boundingBox===null&&e.computeBoundingBox(),fu.copy(e.boundingBox)):(r.boundingBox===null&&r.computeBoundingBox(),fu.copy(r.boundingBox)),fu.applyMatrix4(e.matrixWorld),this.union(fu)}const c=e.children;for(let f=0,d=c.length;f<d;f++)this.expandByObject(c[f],i);return this}containsPoint(e){return e.x>=this.min.x&&e.x<=this.max.x&&e.y>=this.min.y&&e.y<=this.max.y&&e.z>=this.min.z&&e.z<=this.max.z}containsBox(e){return this.min.x<=e.min.x&&e.max.x<=this.max.x&&this.min.y<=e.min.y&&e.max.y<=this.max.y&&this.min.z<=e.min.z&&e.max.z<=this.max.z}getParameter(e,i){return i.set((e.x-this.min.x)/(this.max.x-this.min.x),(e.y-this.min.y)/(this.max.y-this.min.y),(e.z-this.min.z)/(this.max.z-this.min.z))}intersectsBox(e){return e.max.x>=this.min.x&&e.min.x<=this.max.x&&e.max.y>=this.min.y&&e.min.y<=this.max.y&&e.max.z>=this.min.z&&e.min.z<=this.max.z}intersectsSphere(e){return this.clampPoint(e.center,Ii),Ii.distanceToSquared(e.center)<=e.radius*e.radius}intersectsPlane(e){let i,r;return e.normal.x>0?(i=e.normal.x*this.min.x,r=e.normal.x*this.max.x):(i=e.normal.x*this.max.x,r=e.normal.x*this.min.x),e.normal.y>0?(i+=e.normal.y*this.min.y,r+=e.normal.y*this.max.y):(i+=e.normal.y*this.max.y,r+=e.normal.y*this.min.y),e.normal.z>0?(i+=e.normal.z*this.min.z,r+=e.normal.z*this.max.z):(i+=e.normal.z*this.max.z,r+=e.normal.z*this.min.z),i<=-e.constant&&r>=-e.constant}intersectsTriangle(e){if(this.isEmpty())return!1;this.getCenter(Sl),du.subVectors(this.max,Sl),Jr.subVectors(e.a,Sl),$r.subVectors(e.b,Sl),to.subVectors(e.c,Sl),gs.subVectors($r,Jr),vs.subVectors(to,$r),qs.subVectors(Jr,to);let i=[0,-gs.z,gs.y,0,-vs.z,vs.y,0,-qs.z,qs.y,gs.z,0,-gs.x,vs.z,0,-vs.x,qs.z,0,-qs.x,-gs.y,gs.x,0,-vs.y,vs.x,0,-qs.y,qs.x,0];return!qh(i,Jr,$r,to,du)||(i=[1,0,0,0,1,0,0,0,1],!qh(i,Jr,$r,to,du))?!1:(hu.crossVectors(gs,vs),i=[hu.x,hu.y,hu.z],qh(i,Jr,$r,to,du))}clampPoint(e,i){return i.copy(e).clamp(this.min,this.max)}distanceToPoint(e){return this.clampPoint(e,Ii).distanceTo(e)}getBoundingSphere(e){return this.isEmpty()?e.makeEmpty():(this.getCenter(e.center),e.radius=this.getSize(Ii).length()*.5),e}intersect(e){return this.min.max(e.min),this.max.min(e.max),this.isEmpty()&&this.makeEmpty(),this}union(e){return this.min.min(e.min),this.max.max(e.max),this}applyMatrix4(e){return this.isEmpty()?this:(Ua[0].set(this.min.x,this.min.y,this.min.z).applyMatrix4(e),Ua[1].set(this.min.x,this.min.y,this.max.z).applyMatrix4(e),Ua[2].set(this.min.x,this.max.y,this.min.z).applyMatrix4(e),Ua[3].set(this.min.x,this.max.y,this.max.z).applyMatrix4(e),Ua[4].set(this.max.x,this.min.y,this.min.z).applyMatrix4(e),Ua[5].set(this.max.x,this.min.y,this.max.z).applyMatrix4(e),Ua[6].set(this.max.x,this.max.y,this.min.z).applyMatrix4(e),Ua[7].set(this.max.x,this.max.y,this.max.z).applyMatrix4(e),this.setFromPoints(Ua),this)}translate(e){return this.min.add(e),this.max.add(e),this}equals(e){return e.min.equals(this.min)&&e.max.equals(this.max)}toJSON(){return{min:this.min.toArray(),max:this.max.toArray()}}fromJSON(e){return this.min.fromArray(e.min),this.max.fromArray(e.max),this}}const Ua=[new dt,new dt,new dt,new dt,new dt,new dt,new dt,new dt],Ii=new dt,fu=new ir,Jr=new dt,$r=new dt,to=new dt,gs=new dt,vs=new dt,qs=new dt,Sl=new dt,du=new dt,hu=new dt,Ys=new dt;function qh(o,e,i,r,c){for(let f=0,d=o.length-3;f<=d;f+=3){Ys.fromArray(o,f);const h=c.x*Math.abs(Ys.x)+c.y*Math.abs(Ys.y)+c.z*Math.abs(Ys.z),m=e.dot(Ys),p=i.dot(Ys),x=r.dot(Ys);if(Math.max(-Math.max(m,p,x),Math.min(m,p,x))>h)return!1}return!0}const vn=new dt,pu=new Re;let vb=0;class ua extends nr{constructor(e,i,r=!1){if(super(),Array.isArray(e))throw new TypeError("THREE.BufferAttribute: array should be a Typed Array.");this.isBufferAttribute=!0,Object.defineProperty(this,"id",{value:vb++}),this.name="",this.array=e,this.itemSize=i,this.count=e!==void 0?e.length/i:0,this.normalized=r,this.usage=JE,this.updateRanges=[],this.gpuType=Hi,this.version=0}onUploadCallback(){}set needsUpdate(e){e===!0&&this.version++}setUsage(e){return this.usage=e,this}addUpdateRange(e,i){this.updateRanges.push({start:e,count:i})}clearUpdateRanges(){this.updateRanges.length=0}copy(e){return this.name=e.name,this.array=new e.array.constructor(e.array),this.itemSize=e.itemSize,this.count=e.count,this.normalized=e.normalized,this.usage=e.usage,this.gpuType=e.gpuType,this}copyAt(e,i,r){e*=this.itemSize,r*=i.itemSize;for(let c=0,f=this.itemSize;c<f;c++)this.array[e+c]=i.array[r+c];return this}copyArray(e){return this.array.set(e),this}applyMatrix3(e){if(this.itemSize===2)for(let i=0,r=this.count;i<r;i++)pu.fromBufferAttribute(this,i),pu.applyMatrix3(e),this.setXY(i,pu.x,pu.y);else if(this.itemSize===3)for(let i=0,r=this.count;i<r;i++)vn.fromBufferAttribute(this,i),vn.applyMatrix3(e),this.setXYZ(i,vn.x,vn.y,vn.z);return this}applyMatrix4(e){for(let i=0,r=this.count;i<r;i++)vn.fromBufferAttribute(this,i),vn.applyMatrix4(e),this.setXYZ(i,vn.x,vn.y,vn.z);return this}applyNormalMatrix(e){for(let i=0,r=this.count;i<r;i++)vn.fromBufferAttribute(this,i),vn.applyNormalMatrix(e),this.setXYZ(i,vn.x,vn.y,vn.z);return this}transformDirection(e){for(let i=0,r=this.count;i<r;i++)vn.fromBufferAttribute(this,i),vn.transformDirection(e),this.setXYZ(i,vn.x,vn.y,vn.z);return this}set(e,i=0){return this.array.set(e,i),this}getComponent(e,i){let r=this.array[e*this.itemSize+i];return this.normalized&&(r=_l(r,this.array)),r}setComponent(e,i,r){return this.normalized&&(r=$n(r,this.array)),this.array[e*this.itemSize+i]=r,this}getX(e){let i=this.array[e*this.itemSize];return this.normalized&&(i=_l(i,this.array)),i}setX(e,i){return this.normalized&&(i=$n(i,this.array)),this.array[e*this.itemSize]=i,this}getY(e){let i=this.array[e*this.itemSize+1];return this.normalized&&(i=_l(i,this.array)),i}setY(e,i){return this.normalized&&(i=$n(i,this.array)),this.array[e*this.itemSize+1]=i,this}getZ(e){let i=this.array[e*this.itemSize+2];return this.normalized&&(i=_l(i,this.array)),i}setZ(e,i){return this.normalized&&(i=$n(i,this.array)),this.array[e*this.itemSize+2]=i,this}getW(e){let i=this.array[e*this.itemSize+3];return this.normalized&&(i=_l(i,this.array)),i}setW(e,i){return this.normalized&&(i=$n(i,this.array)),this.array[e*this.itemSize+3]=i,this}setXY(e,i,r){return e*=this.itemSize,this.normalized&&(i=$n(i,this.array),r=$n(r,this.array)),this.array[e+0]=i,this.array[e+1]=r,this}setXYZ(e,i,r,c){return e*=this.itemSize,this.normalized&&(i=$n(i,this.array),r=$n(r,this.array),c=$n(c,this.array)),this.array[e+0]=i,this.array[e+1]=r,this.array[e+2]=c,this}setXYZW(e,i,r,c,f){return e*=this.itemSize,this.normalized&&(i=$n(i,this.array),r=$n(r,this.array),c=$n(c,this.array),f=$n(f,this.array)),this.array[e+0]=i,this.array[e+1]=r,this.array[e+2]=c,this.array[e+3]=f,this}onUpload(e){return this.onUploadCallback=e,this}clone(){return new this.constructor(this.array,this.itemSize).copy(this)}toJSON(){const e={itemSize:this.itemSize,type:this.array.constructor.name,array:Array.from(this.array),normalized:this.normalized};return e.name=this.name,e.usage=this.usage,e.gpuType=this.gpuType,e}dispose(){this.dispatchEvent({type:"dispose"})}}class lS extends ua{constructor(e,i,r){super(new Uint16Array(e),i,r)}}class cS extends ua{constructor(e,i,r){super(new Uint32Array(e),i,r)}}class Yn extends ua{constructor(e,i,r){super(new Float32Array(e),i,r)}}const _b=new ir,yl=new dt,Yh=new dt;class zl{constructor(e=new dt,i=-1){this.isSphere=!0,this.center=e,this.radius=i}set(e,i){return this.center.copy(e),this.radius=i,this}setFromPoints(e,i){const r=this.center;i!==void 0?r.copy(i):_b.setFromPoints(e).getCenter(r);let c=0;for(let f=0,d=e.length;f<d;f++)c=Math.max(c,r.distanceToSquared(e[f]));return this.radius=Math.sqrt(c),this}copy(e){return this.center.copy(e.center),this.radius=e.radius,this}isEmpty(){return this.radius<0}makeEmpty(){return this.center.set(0,0,0),this.radius=-1,this}containsPoint(e){return e.distanceToSquared(this.center)<=this.radius*this.radius}distanceToPoint(e){return e.distanceTo(this.center)-this.radius}intersectsSphere(e){const i=this.radius+e.radius;return e.center.distanceToSquared(this.center)<=i*i}intersectsBox(e){return e.intersectsSphere(this)}intersectsPlane(e){return Math.abs(e.distanceToPoint(this.center))<=this.radius}clampPoint(e,i){const r=this.center.distanceToSquared(e);return i.copy(e),r>this.radius*this.radius&&(i.sub(this.center).normalize(),i.multiplyScalar(this.radius).add(this.center)),i}getBoundingBox(e){return this.isEmpty()?(e.makeEmpty(),e):(e.set(this.center,this.center),e.expandByScalar(this.radius),e)}applyMatrix4(e){return this.center.applyMatrix4(e),this.radius=this.radius*e.getMaxScaleOnAxis(),this}translate(e){return this.center.add(e),this}expandByPoint(e){if(this.isEmpty())return this.center.copy(e),this.radius=0,this;yl.subVectors(e,this.center);const i=yl.lengthSq();if(i>this.radius*this.radius){const r=Math.sqrt(i),c=(r-this.radius)*.5;this.center.addScaledVector(yl,c/r),this.radius+=c}return this}union(e){return e.isEmpty()?this:this.isEmpty()?(this.copy(e),this):(this.center.equals(e.center)===!0?this.radius=Math.max(this.radius,e.radius):(Yh.subVectors(e.center,this.center).setLength(e.radius),this.expandByPoint(yl.copy(e.center).add(Yh)),this.expandByPoint(yl.copy(e.center).sub(Yh))),this)}equals(e){return e.center.equals(this.center)&&e.radius===this.radius}clone(){return new this.constructor().copy(this)}toJSON(){return{radius:this.radius,center:this.center.toArray()}}fromJSON(e){return this.radius=e.radius,this.center.fromArray(e.center),this}}let xb=0;const Ai=new nn,jh=new Rn,eo=new dt,hi=new ir,Ml=new ir,An=new dt;class ki extends nr{constructor(){super(),this.isBufferGeometry=!0,Object.defineProperty(this,"id",{value:xb++}),this.uuid=Bl(),this.name="",this.type="BufferGeometry",this.index=null,this.indirect=null,this.indirectOffset=0,this.attributes={},this.morphAttributes={},this.morphTargetsRelative=!1,this.groups=[],this.boundingBox=null,this.boundingSphere=null,this.drawRange={start:0,count:1/0},this.userData={},this._transformed=!1}getIndex(){return this.index}setIndex(e){return Array.isArray(e)?this.index=new($E(e)?cS:lS)(e,1):this.index=e,this}setIndirect(e,i=0){return this.indirect=e,this.indirectOffset=i,this}getIndirect(){return this.indirect}getAttribute(e){return this.attributes[e]}setAttribute(e,i){return this.attributes[e]=i,this}deleteAttribute(e){return delete this.attributes[e],this}hasAttribute(e){return this.attributes[e]!==void 0}addGroup(e,i,r=0){this.groups.push({start:e,count:i,materialIndex:r})}clearGroups(){this.groups=[]}setDrawRange(e,i){this.drawRange.start=e,this.drawRange.count=i}applyMatrix4(e){const i=this.attributes.position;i!==void 0&&(i.applyMatrix4(e),i.needsUpdate=!0);const r=this.attributes.normal;if(r!==void 0){const f=new de().getNormalMatrix(e);r.applyNormalMatrix(f),r.needsUpdate=!0}const c=this.attributes.tangent;return c!==void 0&&(c.transformDirection(e),c.needsUpdate=!0),this.boundingBox!==null&&this.computeBoundingBox(),this.boundingSphere!==null&&this.computeBoundingSphere(),this._transformed=!0,this}applyQuaternion(e){return Ai.makeRotationFromQuaternion(e),this.applyMatrix4(Ai),this}rotateX(e){return Ai.makeRotationX(e),this.applyMatrix4(Ai),this}rotateY(e){return Ai.makeRotationY(e),this.applyMatrix4(Ai),this}rotateZ(e){return Ai.makeRotationZ(e),this.applyMatrix4(Ai),this}translate(e,i,r){return Ai.makeTranslation(e,i,r),this.applyMatrix4(Ai),this}scale(e,i,r){return Ai.makeScale(e,i,r),this.applyMatrix4(Ai),this}lookAt(e){return jh.lookAt(e),jh.updateMatrix(),this.applyMatrix4(jh.matrix),this}center(){return this.computeBoundingBox(),this.boundingBox.getCenter(eo).negate(),this.translate(eo.x,eo.y,eo.z),this}setFromPoints(e){const i=this.getAttribute("position");if(i===void 0){const r=[];for(let c=0,f=e.length;c<f;c++){const d=e[c];r.push(d.x,d.y,d.z||0)}this.setAttribute("position",new Yn(r,3))}else{const r=Math.min(e.length,i.count);for(let c=0;c<r;c++){const f=e[c];i.setXYZ(c,f.x,f.y,f.z||0)}e.length>i.count&&oe("BufferGeometry: Buffer size too small for points data. Use .dispose() and create a new geometry."),i.needsUpdate=!0}return this}computeBoundingBox(){this.boundingBox===null&&(this.boundingBox=new ir);const e=this.attributes.position,i=this.morphAttributes.position;if(e&&e.isGLBufferAttribute){ze("BufferGeometry.computeBoundingBox(): GLBufferAttribute requires a manual bounding box.",this),this.boundingBox.set(new dt(-1/0,-1/0,-1/0),new dt(1/0,1/0,1/0));return}if(e!==void 0){if(this.boundingBox.setFromBufferAttribute(e),i)for(let r=0,c=i.length;r<c;r++){const f=i[r];hi.setFromBufferAttribute(f),this.morphTargetsRelative?(An.addVectors(this.boundingBox.min,hi.min),this.boundingBox.expandByPoint(An),An.addVectors(this.boundingBox.max,hi.max),this.boundingBox.expandByPoint(An)):(this.boundingBox.expandByPoint(hi.min),this.boundingBox.expandByPoint(hi.max))}}else this.boundingBox.makeEmpty();(isNaN(this.boundingBox.min.x)||isNaN(this.boundingBox.min.y)||isNaN(this.boundingBox.min.z))&&ze('BufferGeometry.computeBoundingBox(): Computed min/max have NaN values. The "position" attribute is likely to have NaN values.',this)}computeBoundingSphere(){this.boundingSphere===null&&(this.boundingSphere=new zl);const e=this.attributes.position,i=this.morphAttributes.position;if(e&&e.isGLBufferAttribute){ze("BufferGeometry.computeBoundingSphere(): GLBufferAttribute requires a manual bounding sphere.",this),this.boundingSphere.set(new dt,1/0);return}if(e){const r=this.boundingSphere.center;if(hi.setFromBufferAttribute(e),i)for(let f=0,d=i.length;f<d;f++){const h=i[f];Ml.setFromBufferAttribute(h),this.morphTargetsRelative?(An.addVectors(hi.min,Ml.min),hi.expandByPoint(An),An.addVectors(hi.max,Ml.max),hi.expandByPoint(An)):(hi.expandByPoint(Ml.min),hi.expandByPoint(Ml.max))}hi.getCenter(r);let c=0;for(let f=0,d=e.count;f<d;f++)An.fromBufferAttribute(e,f),c=Math.max(c,r.distanceToSquared(An));if(i)for(let f=0,d=i.length;f<d;f++){const h=i[f],m=this.morphTargetsRelative;for(let p=0,x=h.count;p<x;p++)An.fromBufferAttribute(h,p),m&&(eo.fromBufferAttribute(e,p),An.add(eo)),c=Math.max(c,r.distanceToSquared(An))}this.boundingSphere.radius=Math.sqrt(c),isNaN(this.boundingSphere.radius)&&ze('BufferGeometry.computeBoundingSphere(): Computed radius is NaN. The "position" attribute is likely to have NaN values.',this)}}computeTangents(){const e=this.index,i=this.attributes;if(e===null||i.position===void 0||i.normal===void 0||i.uv===void 0){ze("BufferGeometry: .computeTangents() failed. Missing required attributes (index, position, normal or uv)");return}const r=i.position,c=i.normal,f=i.uv;let d=this.getAttribute("tangent");(d===void 0||d.count!==r.count)&&(d=new ua(new Float32Array(4*r.count),4),this.setAttribute("tangent",d));const h=[],m=[];for(let b=0;b<r.count;b++)h[b]=new dt,m[b]=new dt;const p=new dt,x=new dt,v=new dt,_=new Re,M=new Re,R=new Re,N=new dt,y=new dt;function S(b,L,X){p.fromBufferAttribute(r,b),x.fromBufferAttribute(r,L),v.fromBufferAttribute(r,X),_.fromBufferAttribute(f,b),M.fromBufferAttribute(f,L),R.fromBufferAttribute(f,X),x.sub(p),v.sub(p),M.sub(_),R.sub(_);const j=1/(M.x*R.y-R.x*M.y);isFinite(j)&&(N.copy(x).multiplyScalar(R.y).addScaledVector(v,-M.y).multiplyScalar(j),y.copy(v).multiplyScalar(M.x).addScaledVector(x,-R.x).multiplyScalar(j),h[b].add(N),h[L].add(N),h[X].add(N),m[b].add(y),m[L].add(y),m[X].add(y))}let O=this.groups;O.length===0&&(O=[{start:0,count:e.count}]);for(let b=0,L=O.length;b<L;++b){const X=O[b],j=X.start,J=X.count;for(let P=j,q=j+J;P<q;P+=3)S(e.getX(P+0),e.getX(P+1),e.getX(P+2))}const k=new dt,C=new dt,D=new dt,U=new dt;function B(b){D.fromBufferAttribute(c,b),U.copy(D);const L=h[b];k.copy(L),k.sub(D.multiplyScalar(D.dot(L))).normalize(),C.crossVectors(U,L);const j=C.dot(m[b])<0?-1:1;d.setXYZW(b,k.x,k.y,k.z,j)}for(let b=0,L=O.length;b<L;++b){const X=O[b],j=X.start,J=X.count;for(let P=j,q=j+J;P<q;P+=3)B(e.getX(P+0)),B(e.getX(P+1)),B(e.getX(P+2))}this._transformed=!0}computeVertexNormals(){const e=this.index,i=this.getAttribute("position");if(i!==void 0){let r=this.getAttribute("normal");if(r===void 0||r.count!==i.count)r=new ua(new Float32Array(i.count*3),3),this.setAttribute("normal",r);else for(let _=0,M=r.count;_<M;_++)r.setXYZ(_,0,0,0);const c=new dt,f=new dt,d=new dt,h=new dt,m=new dt,p=new dt,x=new dt,v=new dt;if(e)for(let _=0,M=e.count;_<M;_+=3){const R=e.getX(_+0),N=e.getX(_+1),y=e.getX(_+2);c.fromBufferAttribute(i,R),f.fromBufferAttribute(i,N),d.fromBufferAttribute(i,y),x.subVectors(d,f),v.subVectors(c,f),x.cross(v),h.fromBufferAttribute(r,R),m.fromBufferAttribute(r,N),p.fromBufferAttribute(r,y),h.add(x),m.add(x),p.add(x),r.setXYZ(R,h.x,h.y,h.z),r.setXYZ(N,m.x,m.y,m.z),r.setXYZ(y,p.x,p.y,p.z)}else for(let _=0,M=i.count;_<M;_+=3)c.fromBufferAttribute(i,_+0),f.fromBufferAttribute(i,_+1),d.fromBufferAttribute(i,_+2),x.subVectors(d,f),v.subVectors(c,f),x.cross(v),r.setXYZ(_+0,x.x,x.y,x.z),r.setXYZ(_+1,x.x,x.y,x.z),r.setXYZ(_+2,x.x,x.y,x.z);this.normalizeNormals(),r.needsUpdate=!0}}normalizeNormals(){const e=this.attributes.normal;for(let i=0,r=e.count;i<r;i++)An.fromBufferAttribute(e,i),An.normalize(),e.setXYZ(i,An.x,An.y,An.z)}toNonIndexed(){function e(h,m){const p=h.array,x=h.itemSize,v=h.normalized,_=new p.constructor(m.length*x);let M=0,R=0;for(let N=0,y=m.length;N<y;N++){h.isInterleavedBufferAttribute?M=m[N]*h.data.stride+h.offset:M=m[N]*x;for(let S=0;S<x;S++)_[R++]=p[M++]}return new ua(_,x,v)}if(this.index===null)return oe("BufferGeometry.toNonIndexed(): BufferGeometry is already non-indexed."),this;const i=new ki,r=this.index.array,c=this.attributes;for(const h in c){const m=c[h],p=e(m,r);i.setAttribute(h,p)}const f=this.morphAttributes;for(const h in f){const m=[],p=f[h];for(let x=0,v=p.length;x<v;x++){const _=p[x],M=e(_,r);m.push(M)}i.morphAttributes[h]=m}i.morphTargetsRelative=this.morphTargetsRelative;const d=this.groups;for(let h=0,m=d.length;h<m;h++){const p=d[h];i.addGroup(p.start,p.count,p.materialIndex)}return i}toJSON(){const e={metadata:{version:4.7,type:"BufferGeometry",generator:"BufferGeometry.toJSON"}};if(e.uuid=this.uuid,e.type=this.parameters!==void 0&&this._transformed===!0?"BufferGeometry":this.type,e.name=this.name,Object.keys(this.userData).length>0&&(e.userData=this.userData),this.parameters!==void 0&&this._transformed!==!0){const m=this.parameters;for(const p in m)m[p]!==void 0&&(e[p]=m[p]);return e}e.data={attributes:{}};const i=this.index;i!==null&&(e.data.index={type:i.array.constructor.name,array:Array.prototype.slice.call(i.array)});const r=this.attributes;for(const m in r){const p=r[m];e.data.attributes[m]=p.toJSON(e.data)}const c={};let f=!1;for(const m in this.morphAttributes){const p=this.morphAttributes[m],x=[];for(let v=0,_=p.length;v<_;v++){const M=p[v];x.push(M.toJSON(e.data))}x.length>0&&(c[m]=x,f=!0)}f&&(e.data.morphAttributes=c,e.data.morphTargetsRelative=this.morphTargetsRelative);const d=this.groups;d.length>0&&(e.data.groups=JSON.parse(JSON.stringify(d)));const h=this.boundingSphere;return h!==null&&(e.data.boundingSphere=h.toJSON()),e}clone(){return new this.constructor().copy(this)}copy(e){this.index=null,this.attributes={},this.morphAttributes={},this.groups=[],this.boundingBox=null,this.boundingSphere=null;const i={};this.name=e.name;const r=e.index;r!==null&&this.setIndex(r.clone());const c=e.attributes;for(const p in c){const x=c[p];this.setAttribute(p,x.clone(i))}const f=e.morphAttributes;for(const p in f){const x=[],v=f[p];for(let _=0,M=v.length;_<M;_++)x.push(v[_].clone(i));this.morphAttributes[p]=x}this.morphTargetsRelative=e.morphTargetsRelative;const d=e.groups;for(let p=0,x=d.length;p<x;p++){const v=d[p];this.addGroup(v.start,v.count,v.materialIndex)}const h=e.boundingBox;h!==null&&(this.boundingBox=h.clone());const m=e.boundingSphere;return m!==null&&(this.boundingSphere=m.clone()),this.drawRange.start=e.drawRange.start,this.drawRange.count=e.drawRange.count,this.userData=e.userData,this._transformed=e._transformed,this}dispose(){this.dispatchEvent({type:"dispose"})}}const Zh=new dt,Sb=new dt,yb=new de;class Ss{constructor(e=new dt(1,0,0),i=0){this.isPlane=!0,this.normal=e,this.constant=i}set(e,i){return this.normal.copy(e),this.constant=i,this}setComponents(e,i,r,c){return this.normal.set(e,i,r),this.constant=c,this}setFromNormalAndCoplanarPoint(e,i){return this.normal.copy(e),this.constant=-i.dot(this.normal),this}setFromCoplanarPoints(e,i,r){const c=Zh.subVectors(r,i).cross(Sb.subVectors(e,i)).normalize();return this.setFromNormalAndCoplanarPoint(c,e),this}copy(e){return this.normal.copy(e.normal),this.constant=e.constant,this}normalize(){const e=1/this.normal.length();return this.normal.multiplyScalar(e),this.constant*=e,this}negate(){return this.constant*=-1,this.normal.negate(),this}distanceToPoint(e){return this.normal.dot(e)+this.constant}distanceToSphere(e){return this.distanceToPoint(e.center)-e.radius}projectPoint(e,i){return i.copy(e).addScaledVector(this.normal,-this.distanceToPoint(e))}intersectLine(e,i,r=!0){const c=e.delta(Zh),f=this.normal.dot(c);if(f===0)return this.distanceToPoint(e.start)===0?i.copy(e.start):null;const d=-(e.start.dot(this.normal)+this.constant)/f;return r===!0&&(d<0||d>1)?null:i.copy(e.start).addScaledVector(c,d)}intersectsLine(e){const i=this.distanceToPoint(e.start),r=this.distanceToPoint(e.end);return i<0&&r>0||r<0&&i>0}intersectsBox(e){return e.intersectsPlane(this)}intersectsSphere(e){return e.intersectsPlane(this)}coplanarPoint(e){return e.copy(this.normal).multiplyScalar(-this.constant)}applyMatrix4(e,i){const r=i||yb.getNormalMatrix(e),c=this.coplanarPoint(Zh).applyMatrix4(e),f=this.normal.applyMatrix3(r).normalize();return this.constant=-c.dot(f),this}translate(e){return this.constant-=e.dot(this.normal),this}equals(e){return e.normal.equals(this.normal)&&e.constant===this.constant}clone(){return new this.constructor().copy(this)}toJSON(){return{normal:this.normal.toArray(),constant:this.constant}}fromJSON(e){return this.normal.fromArray(e.normal),this.constant=e.constant,this}}let Mb=0;class ar extends nr{constructor(){super(),this.isMaterial=!0,Object.defineProperty(this,"id",{value:Mb++}),this.uuid=Bl(),this.name="",this.type="Material",this.blending=Dl,this.side=$s,this.vertexColors=!1,this.opacity=1,this.transparent=!1,this.alphaHash=!1,this.blendSrc=kx,this.blendDst=Xx,this.blendEquation=oo,this.blendSrcAlpha=null,this.blendDstAlpha=null,this.blendEquationAlpha=null,this.blendColor=new he(0,0,0),this.blendAlpha=0,this.depthFunc=Ul,this.depthTest=!0,this.depthWrite=!0,this.stencilWriteMask=255,this.stencilFunc=WE,this.stencilRef=0,this.stencilFuncMask=255,this.stencilFail=Ch,this.stencilZFail=Ch,this.stencilZPass=Ch,this.stencilWrite=!1,this.clippingPlanes=null,this.clipIntersection=!1,this.clipShadows=!1,this.shadowSide=null,this.colorWrite=!0,this.precision=null,this.polygonOffset=!1,this.polygonOffsetFactor=0,this.polygonOffsetUnits=0,this.dithering=!1,this.alphaToCoverage=!1,this.premultipliedAlpha=!1,this.forceSinglePass=!1,this.allowOverride=!0,this.visible=!0,this.toneMapped=!0,this.userData={},this.version=0,this._alphaTest=0}get alphaTest(){return this._alphaTest}set alphaTest(e){this._alphaTest>0!=e>0&&this.version++,this._alphaTest=e}onBeforeRender(){}onBeforeCompile(){}customProgramCacheKey(){return this.onBeforeCompile.toString()}setValues(e){if(e!==void 0)for(const i in e){const r=e[i];if(r===void 0){oe(`Material: parameter '${i}' has value of undefined.`);continue}const c=this[i];if(c===void 0){oe(`Material: '${i}' is not a property of THREE.${this.type}.`);continue}c&&c.isColor?c.set(r):c&&c.isVector2&&r&&r.isVector2||c&&c.isEuler&&r&&r.isEuler||c&&c.isVector3&&r&&r.isVector3?c.copy(r):this[i]=r}}toJSON(e){const i=e===void 0||typeof e=="string";i&&(e={textures:{},images:{}});const r={metadata:{version:4.7,type:"Material",generator:"Material.toJSON"}};r.uuid=this.uuid,r.type=this.type,r.blending=this.blending,r.side=this.side,r.shadowSide=this.shadowSide,r.vertexColors=this.vertexColors,r.opacity=this.opacity,r.transparent=this.transparent,r.blendSrc=this.blendSrc,r.blendDst=this.blendDst,r.blendEquation=this.blendEquation,r.blendSrcAlpha=this.blendSrcAlpha,r.blendDstAlpha=this.blendDstAlpha,r.blendEquationAlpha=this.blendEquationAlpha,r.blendColor=this.blendColor.getHex(),r.blendAlpha=this.blendAlpha,r.depthFunc=this.depthFunc,r.depthTest=this.depthTest,r.depthWrite=this.depthWrite,r.colorWrite=this.colorWrite,r.clipIntersection=this.clipIntersection,r.clipShadows=this.clipShadows,r.stencilWriteMask=this.stencilWriteMask,r.stencilFunc=this.stencilFunc,r.stencilRef=this.stencilRef,r.stencilFuncMask=this.stencilFuncMask,r.stencilFail=this.stencilFail,r.stencilZFail=this.stencilZFail,r.stencilZPass=this.stencilZPass,r.stencilWrite=this.stencilWrite,r.polygonOffset=this.polygonOffset,r.polygonOffsetFactor=this.polygonOffsetFactor,r.polygonOffsetUnits=this.polygonOffsetUnits,r.dithering=this.dithering,r.alphaTest=this.alphaTest,r.alphaHash=this.alphaHash,r.alphaToCoverage=this.alphaToCoverage,r.premultipliedAlpha=this.premultipliedAlpha,r.forceSinglePass=this.forceSinglePass,r.allowOverride=this.allowOverride,r.visible=this.visible,r.toneMapped=this.toneMapped,r.name=this.name,this.color&&this.color.isColor&&(r.color=this.color.getHex()),this.roughness!==void 0&&(r.roughness=this.roughness),this.metalness!==void 0&&(r.metalness=this.metalness),this.sheen!==void 0&&(r.sheen=this.sheen),this.sheenColor&&this.sheenColor.isColor&&(r.sheenColor=this.sheenColor.getHex()),this.sheenRoughness!==void 0&&(r.sheenRoughness=this.sheenRoughness),this.emissive&&this.emissive.isColor&&(r.emissive=this.emissive.getHex()),this.emissiveIntensity!==void 0&&(r.emissiveIntensity=this.emissiveIntensity),this.specular&&this.specular.isColor&&(r.specular=this.specular.getHex()),this.specularIntensity!==void 0&&(r.specularIntensity=this.specularIntensity),this.specularColor&&this.specularColor.isColor&&(r.specularColor=this.specularColor.getHex()),this.shininess!==void 0&&(r.shininess=this.shininess),this.clearcoat!==void 0&&(r.clearcoat=this.clearcoat),this.clearcoatRoughness!==void 0&&(r.clearcoatRoughness=this.clearcoatRoughness),this.clearcoatMap&&this.clearcoatMap.isTexture&&(r.clearcoatMap=this.clearcoatMap.toJSON(e).uuid),this.clearcoatRoughnessMap&&this.clearcoatRoughnessMap.isTexture&&(r.clearcoatRoughnessMap=this.clearcoatRoughnessMap.toJSON(e).uuid),this.clearcoatNormalMap&&this.clearcoatNormalMap.isTexture&&(r.clearcoatNormalMap=this.clearcoatNormalMap.toJSON(e).uuid,r.clearcoatNormalScale=this.clearcoatNormalScale.toArray()),this.sheenColorMap&&this.sheenColorMap.isTexture&&(r.sheenColorMap=this.sheenColorMap.toJSON(e).uuid),this.sheenRoughnessMap&&this.sheenRoughnessMap.isTexture&&(r.sheenRoughnessMap=this.sheenRoughnessMap.toJSON(e).uuid),this.dispersion!==void 0&&(r.dispersion=this.dispersion),this.retroreflectivity!==void 0&&(r.retroreflectivity=this.retroreflectivity),this.iridescence!==void 0&&(r.iridescence=this.iridescence),this.iridescenceIOR!==void 0&&(r.iridescenceIOR=this.iridescenceIOR),this.iridescenceThicknessRange!==void 0&&(r.iridescenceThicknessRange=this.iridescenceThicknessRange),this.iridescenceMap&&this.iridescenceMap.isTexture&&(r.iridescenceMap=this.iridescenceMap.toJSON(e).uuid),this.iridescenceThicknessMap&&this.iridescenceThicknessMap.isTexture&&(r.iridescenceThicknessMap=this.iridescenceThicknessMap.toJSON(e).uuid),this.anisotropy!==void 0&&(r.anisotropy=this.anisotropy),this.anisotropyRotation!==void 0&&(r.anisotropyRotation=this.anisotropyRotation),this.anisotropyMap&&this.anisotropyMap.isTexture&&(r.anisotropyMap=this.anisotropyMap.toJSON(e).uuid),this.map&&this.map.isTexture&&(r.map=this.map.toJSON(e).uuid),this.matcap&&this.matcap.isTexture&&(r.matcap=this.matcap.toJSON(e).uuid),this.alphaMap&&this.alphaMap.isTexture&&(r.alphaMap=this.alphaMap.toJSON(e).uuid),this.lightMap&&this.lightMap.isTexture&&(r.lightMap=this.lightMap.toJSON(e).uuid,r.lightMapIntensity=this.lightMapIntensity),this.aoMap&&this.aoMap.isTexture&&(r.aoMap=this.aoMap.toJSON(e).uuid,r.aoMapIntensity=this.aoMapIntensity),this.bumpMap&&this.bumpMap.isTexture&&(r.bumpMap=this.bumpMap.toJSON(e).uuid,r.bumpScale=this.bumpScale),this.normalMap&&this.normalMap.isTexture&&(r.normalMap=this.normalMap.toJSON(e).uuid,r.normalMapType=this.normalMapType,r.normalScale=this.normalScale.toArray()),this.displacementMap&&this.displacementMap.isTexture&&(r.displacementMap=this.displacementMap.toJSON(e).uuid,r.displacementScale=this.displacementScale,r.displacementBias=this.displacementBias),this.roughnessMap&&this.roughnessMap.isTexture&&(r.roughnessMap=this.roughnessMap.toJSON(e).uuid),this.metalnessMap&&this.metalnessMap.isTexture&&(r.metalnessMap=this.metalnessMap.toJSON(e).uuid),this.emissiveMap&&this.emissiveMap.isTexture&&(r.emissiveMap=this.emissiveMap.toJSON(e).uuid),this.specularMap&&this.specularMap.isTexture&&(r.specularMap=this.specularMap.toJSON(e).uuid),this.specularIntensityMap&&this.specularIntensityMap.isTexture&&(r.specularIntensityMap=this.specularIntensityMap.toJSON(e).uuid),this.specularColorMap&&this.specularColorMap.isTexture&&(r.specularColorMap=this.specularColorMap.toJSON(e).uuid),this.envMap&&this.envMap.isTexture&&(r.envMap=this.envMap.toJSON(e).uuid,this.combine!==void 0&&(r.combine=this.combine)),this.envMapRotation!==void 0&&(r.envMapRotation=this.envMapRotation.toArray()),this.envMapIntensity!==void 0&&(r.envMapIntensity=this.envMapIntensity),this.reflectivity!==void 0&&(r.reflectivity=this.reflectivity),this.refractionRatio!==void 0&&(r.refractionRatio=this.refractionRatio),this.gradientMap&&this.gradientMap.isTexture&&(r.gradientMap=this.gradientMap.toJSON(e).uuid),this.transmission!==void 0&&(r.transmission=this.transmission),this.transmissionMap&&this.transmissionMap.isTexture&&(r.transmissionMap=this.transmissionMap.toJSON(e).uuid),this.thickness!==void 0&&(r.thickness=this.thickness),this.thicknessMap&&this.thicknessMap.isTexture&&(r.thicknessMap=this.thicknessMap.toJSON(e).uuid),this.attenuationDistance!==void 0&&(r.attenuationDistance=this.attenuationDistance),this.attenuationColor!==void 0&&(r.attenuationColor=this.attenuationColor.getHex()),this.size!==void 0&&(r.size=this.size),this.sizeAttenuation!==void 0&&(r.sizeAttenuation=this.sizeAttenuation),Array.isArray(this.clippingPlanes)&&this.clippingPlanes.length>0&&(r.clippingPlanes=this.clippingPlanes.map(f=>f.toJSON())),this.rotation!==void 0&&(r.rotation=this.rotation),this.depthPacking!==void 0&&(r.depthPacking=this.depthPacking),this.linewidth!==void 0&&(r.linewidth=this.linewidth),this.linecap!==void 0&&(r.linecap=this.linecap),this.linejoin!==void 0&&(r.linejoin=this.linejoin),this.dashSize!==void 0&&(r.dashSize=this.dashSize),this.gapSize!==void 0&&(r.gapSize=this.gapSize),this.scale!==void 0&&(r.scale=this.scale),this.wireframe!==void 0&&(r.wireframe=this.wireframe),this.wireframeLinewidth!==void 0&&(r.wireframeLinewidth=this.wireframeLinewidth),this.wireframeLinecap!==void 0&&(r.wireframeLinecap=this.wireframeLinecap),this.wireframeLinejoin!==void 0&&(r.wireframeLinejoin=this.wireframeLinejoin),this.flatShading!==void 0&&(r.flatShading=this.flatShading),this.fog!==void 0&&(r.fog=this.fog),Object.keys(this.userData).length>0&&(r.userData=this.userData);function c(f){const d=[];for(const h in f){const m=f[h];delete m.metadata,d.push(m)}return d}if(i){const f=c(e.textures),d=c(e.images);f.length>0&&(r.textures=f),d.length>0&&(r.images=d)}return r}fromJSON(e,i){if(e.uuid!==void 0&&(this.uuid=e.uuid),e.name!==void 0&&(this.name=e.name),e.color!==void 0&&this.color!==void 0&&this.color.setHex(e.color),e.roughness!==void 0&&(this.roughness=e.roughness),e.metalness!==void 0&&(this.metalness=e.metalness),e.sheen!==void 0&&(this.sheen=e.sheen),e.sheenColor!==void 0&&(this.sheenColor=new he().setHex(e.sheenColor)),e.sheenRoughness!==void 0&&(this.sheenRoughness=e.sheenRoughness),e.emissive!==void 0&&this.emissive!==void 0&&this.emissive.setHex(e.emissive),e.specular!==void 0&&this.specular!==void 0&&this.specular.setHex(e.specular),e.specularIntensity!==void 0&&(this.specularIntensity=e.specularIntensity),e.specularColor!==void 0&&this.specularColor!==void 0&&this.specularColor.setHex(e.specularColor),e.shininess!==void 0&&(this.shininess=e.shininess),e.clearcoat!==void 0&&(this.clearcoat=e.clearcoat),e.clearcoatRoughness!==void 0&&(this.clearcoatRoughness=e.clearcoatRoughness),e.dispersion!==void 0&&(this.dispersion=e.dispersion),e.retroreflectivity!==void 0&&(this.retroreflectivity=e.retroreflectivity),e.iridescence!==void 0&&(this.iridescence=e.iridescence),e.iridescenceIOR!==void 0&&(this.iridescenceIOR=e.iridescenceIOR),e.iridescenceThicknessRange!==void 0&&(this.iridescenceThicknessRange=e.iridescenceThicknessRange),e.transmission!==void 0&&(this.transmission=e.transmission),e.thickness!==void 0&&(this.thickness=e.thickness),e.attenuationDistance!==void 0&&(this.attenuationDistance=e.attenuationDistance),e.attenuationColor!==void 0&&this.attenuationColor!==void 0&&this.attenuationColor.setHex(e.attenuationColor),e.anisotropy!==void 0&&(this.anisotropy=e.anisotropy),e.anisotropyRotation!==void 0&&(this.anisotropyRotation=e.anisotropyRotation),e.fog!==void 0&&(this.fog=e.fog),e.flatShading!==void 0&&(this.flatShading=e.flatShading),e.blending!==void 0&&(this.blending=e.blending),e.combine!==void 0&&(this.combine=e.combine),e.side!==void 0&&(this.side=e.side),e.shadowSide!==void 0&&(this.shadowSide=e.shadowSide),e.opacity!==void 0&&(this.opacity=e.opacity),e.transparent!==void 0&&(this.transparent=e.transparent),e.alphaTest!==void 0&&(this.alphaTest=e.alphaTest),e.alphaHash!==void 0&&(this.alphaHash=e.alphaHash),e.depthFunc!==void 0&&(this.depthFunc=e.depthFunc),e.depthTest!==void 0&&(this.depthTest=e.depthTest),e.depthWrite!==void 0&&(this.depthWrite=e.depthWrite),e.colorWrite!==void 0&&(this.colorWrite=e.colorWrite),e.clippingPlanes!==void 0&&(this.clippingPlanes=e.clippingPlanes.map(r=>new Ss().fromJSON(r))),e.clipIntersection!==void 0&&(this.clipIntersection=e.clipIntersection),e.clipShadows!==void 0&&(this.clipShadows=e.clipShadows),e.depthPacking!==void 0&&(this.depthPacking=e.depthPacking),e.blendSrc!==void 0&&(this.blendSrc=e.blendSrc),e.blendDst!==void 0&&(this.blendDst=e.blendDst),e.blendEquation!==void 0&&(this.blendEquation=e.blendEquation),e.blendSrcAlpha!==void 0&&(this.blendSrcAlpha=e.blendSrcAlpha),e.blendDstAlpha!==void 0&&(this.blendDstAlpha=e.blendDstAlpha),e.blendEquationAlpha!==void 0&&(this.blendEquationAlpha=e.blendEquationAlpha),e.blendColor!==void 0&&this.blendColor!==void 0&&this.blendColor.setHex(e.blendColor),e.blendAlpha!==void 0&&(this.blendAlpha=e.blendAlpha),e.stencilWriteMask!==void 0&&(this.stencilWriteMask=e.stencilWriteMask),e.stencilFunc!==void 0&&(this.stencilFunc=e.stencilFunc),e.stencilRef!==void 0&&(this.stencilRef=e.stencilRef),e.stencilFuncMask!==void 0&&(this.stencilFuncMask=e.stencilFuncMask),e.stencilFail!==void 0&&(this.stencilFail=e.stencilFail),e.stencilZFail!==void 0&&(this.stencilZFail=e.stencilZFail),e.stencilZPass!==void 0&&(this.stencilZPass=e.stencilZPass),e.stencilWrite!==void 0&&(this.stencilWrite=e.stencilWrite),e.wireframe!==void 0&&(this.wireframe=e.wireframe),e.wireframeLinewidth!==void 0&&(this.wireframeLinewidth=e.wireframeLinewidth),e.wireframeLinecap!==void 0&&(this.wireframeLinecap=e.wireframeLinecap),e.wireframeLinejoin!==void 0&&(this.wireframeLinejoin=e.wireframeLinejoin),e.rotation!==void 0&&(this.rotation=e.rotation),e.linewidth!==void 0&&(this.linewidth=e.linewidth),e.linecap!==void 0&&(this.linecap=e.linecap),e.linejoin!==void 0&&(this.linejoin=e.linejoin),e.dashSize!==void 0&&(this.dashSize=e.dashSize),e.gapSize!==void 0&&(this.gapSize=e.gapSize),e.scale!==void 0&&(this.scale=e.scale),e.polygonOffset!==void 0&&(this.polygonOffset=e.polygonOffset),e.polygonOffsetFactor!==void 0&&(this.polygonOffsetFactor=e.polygonOffsetFactor),e.polygonOffsetUnits!==void 0&&(this.polygonOffsetUnits=e.polygonOffsetUnits),e.dithering!==void 0&&(this.dithering=e.dithering),e.alphaToCoverage!==void 0&&(this.alphaToCoverage=e.alphaToCoverage),e.premultipliedAlpha!==void 0&&(this.premultipliedAlpha=e.premultipliedAlpha),e.forceSinglePass!==void 0&&(this.forceSinglePass=e.forceSinglePass),e.allowOverride!==void 0&&(this.allowOverride=e.allowOverride),e.visible!==void 0&&(this.visible=e.visible),e.toneMapped!==void 0&&(this.toneMapped=e.toneMapped),e.userData!==void 0&&(this.userData=e.userData),e.vertexColors!==void 0&&(typeof e.vertexColors=="number"?this.vertexColors=e.vertexColors>0:this.vertexColors=e.vertexColors),e.size!==void 0&&(this.size=e.size),e.sizeAttenuation!==void 0&&(this.sizeAttenuation=e.sizeAttenuation),e.map!==void 0&&(this.map=i[e.map]||null),e.matcap!==void 0&&(this.matcap=i[e.matcap]||null),e.alphaMap!==void 0&&(this.alphaMap=i[e.alphaMap]||null),e.bumpMap!==void 0&&(this.bumpMap=i[e.bumpMap]||null),e.bumpScale!==void 0&&(this.bumpScale=e.bumpScale),e.normalMap!==void 0&&(this.normalMap=i[e.normalMap]||null),e.normalMapType!==void 0&&(this.normalMapType=e.normalMapType),e.normalScale!==void 0){let r=e.normalScale;Array.isArray(r)===!1&&(r=[r,r]),this.normalScale=new Re().fromArray(r)}return e.displacementMap!==void 0&&(this.displacementMap=i[e.displacementMap]||null),e.displacementScale!==void 0&&(this.displacementScale=e.displacementScale),e.displacementBias!==void 0&&(this.displacementBias=e.displacementBias),e.roughnessMap!==void 0&&(this.roughnessMap=i[e.roughnessMap]||null),e.metalnessMap!==void 0&&(this.metalnessMap=i[e.metalnessMap]||null),e.emissiveMap!==void 0&&(this.emissiveMap=i[e.emissiveMap]||null),e.emissiveIntensity!==void 0&&(this.emissiveIntensity=e.emissiveIntensity),e.specularMap!==void 0&&(this.specularMap=i[e.specularMap]||null),e.specularIntensityMap!==void 0&&(this.specularIntensityMap=i[e.specularIntensityMap]||null),e.specularColorMap!==void 0&&(this.specularColorMap=i[e.specularColorMap]||null),e.envMap!==void 0&&(this.envMap=i[e.envMap]||null),e.envMapRotation!==void 0&&this.envMapRotation.fromArray(e.envMapRotation),e.envMapIntensity!==void 0&&(this.envMapIntensity=e.envMapIntensity),e.reflectivity!==void 0&&(this.reflectivity=e.reflectivity),e.refractionRatio!==void 0&&(this.refractionRatio=e.refractionRatio),e.lightMap!==void 0&&(this.lightMap=i[e.lightMap]||null),e.lightMapIntensity!==void 0&&(this.lightMapIntensity=e.lightMapIntensity),e.aoMap!==void 0&&(this.aoMap=i[e.aoMap]||null),e.aoMapIntensity!==void 0&&(this.aoMapIntensity=e.aoMapIntensity),e.gradientMap!==void 0&&(this.gradientMap=i[e.gradientMap]||null),e.clearcoatMap!==void 0&&(this.clearcoatMap=i[e.clearcoatMap]||null),e.clearcoatRoughnessMap!==void 0&&(this.clearcoatRoughnessMap=i[e.clearcoatRoughnessMap]||null),e.clearcoatNormalMap!==void 0&&(this.clearcoatNormalMap=i[e.clearcoatNormalMap]||null),e.clearcoatNormalScale!==void 0&&(this.clearcoatNormalScale=new Re().fromArray(e.clearcoatNormalScale)),e.iridescenceMap!==void 0&&(this.iridescenceMap=i[e.iridescenceMap]||null),e.iridescenceThicknessMap!==void 0&&(this.iridescenceThicknessMap=i[e.iridescenceThicknessMap]||null),e.transmissionMap!==void 0&&(this.transmissionMap=i[e.transmissionMap]||null),e.thicknessMap!==void 0&&(this.thicknessMap=i[e.thicknessMap]||null),e.anisotropyMap!==void 0&&(this.anisotropyMap=i[e.anisotropyMap]||null),e.sheenColorMap!==void 0&&(this.sheenColorMap=i[e.sheenColorMap]||null),e.sheenRoughnessMap!==void 0&&(this.sheenRoughnessMap=i[e.sheenRoughnessMap]||null),this}clone(){return new this.constructor().copy(this)}copy(e){this.name=e.name,this.blending=e.blending,this.side=e.side,this.vertexColors=e.vertexColors,this.opacity=e.opacity,this.transparent=e.transparent,this.blendSrc=e.blendSrc,this.blendDst=e.blendDst,this.blendEquation=e.blendEquation,this.blendSrcAlpha=e.blendSrcAlpha,this.blendDstAlpha=e.blendDstAlpha,this.blendEquationAlpha=e.blendEquationAlpha,this.blendColor.copy(e.blendColor),this.blendAlpha=e.blendAlpha,this.depthFunc=e.depthFunc,this.depthTest=e.depthTest,this.depthWrite=e.depthWrite,this.stencilWriteMask=e.stencilWriteMask,this.stencilFunc=e.stencilFunc,this.stencilRef=e.stencilRef,this.stencilFuncMask=e.stencilFuncMask,this.stencilFail=e.stencilFail,this.stencilZFail=e.stencilZFail,this.stencilZPass=e.stencilZPass,this.stencilWrite=e.stencilWrite;const i=e.clippingPlanes;let r=null;if(i!==null){const c=i.length;r=new Array(c);for(let f=0;f!==c;++f)r[f]=i[f].clone()}return this.clippingPlanes=r,this.clipIntersection=e.clipIntersection,this.clipShadows=e.clipShadows,this.shadowSide=e.shadowSide,this.colorWrite=e.colorWrite,this.precision=e.precision,this.polygonOffset=e.polygonOffset,this.polygonOffsetFactor=e.polygonOffsetFactor,this.polygonOffsetUnits=e.polygonOffsetUnits,this.dithering=e.dithering,this.alphaTest=e.alphaTest,this.alphaHash=e.alphaHash,this.alphaToCoverage=e.alphaToCoverage,this.premultipliedAlpha=e.premultipliedAlpha,this.forceSinglePass=e.forceSinglePass,this.allowOverride=e.allowOverride,this.visible=e.visible,this.toneMapped=e.toneMapped,this.userData=JSON.parse(JSON.stringify(e.userData)),this}dispose(){this.dispatchEvent({type:"dispose"})}set needsUpdate(e){e===!0&&this.version++}}const La=new dt,Kh=new dt,mu=new dt,gu=new dt;class Eb{constructor(e=new dt,i=new dt(0,0,-1)){this.origin=e,this.direction=i}set(e,i){return this.origin.copy(e),this.direction.copy(i),this}copy(e){return this.origin.copy(e.origin),this.direction.copy(e.direction),this}at(e,i){return i.copy(this.origin).addScaledVector(this.direction,e)}lookAt(e){return this.direction.copy(e).sub(this.origin).normalize(),this}recast(e){return this.origin.copy(this.at(e,La)),this}closestPointToPoint(e,i){i.subVectors(e,this.origin);const r=i.dot(this.direction);return r<0?i.copy(this.origin):i.copy(this.origin).addScaledVector(this.direction,r)}distanceToPoint(e){return Math.sqrt(this.distanceSqToPoint(e))}distanceSqToPoint(e){const i=La.subVectors(e,this.origin).dot(this.direction);return i<0?this.origin.distanceToSquared(e):(La.copy(this.origin).addScaledVector(this.direction,i),La.distanceToSquared(e))}distanceSqToSegment(e,i,r,c){Kh.copy(e).add(i).multiplyScalar(.5),mu.copy(i).sub(e).normalize(),gu.copy(this.origin).sub(Kh);const f=e.distanceTo(i)*.5,d=-this.direction.dot(mu),h=gu.dot(this.direction),m=-gu.dot(mu),p=gu.lengthSq(),x=Math.abs(1-d*d);let v,_,M,R;if(x>0)if(v=d*m-h,_=d*h-m,R=f*x,v>=0)if(_>=-R)if(_<=R){const N=1/x;v*=N,_*=N,M=v*(v+d*_+2*h)+_*(d*v+_+2*m)+p}else _=f,v=Math.max(0,-(d*_+h)),M=-v*v+_*(_+2*m)+p;else _=-f,v=Math.max(0,-(d*_+h)),M=-v*v+_*(_+2*m)+p;else _<=-R?(v=Math.max(0,-(-d*f+h)),_=v>0?-f:Math.min(Math.max(-f,-m),f),M=-v*v+_*(_+2*m)+p):_<=R?(v=0,_=Math.min(Math.max(-f,-m),f),M=_*(_+2*m)+p):(v=Math.max(0,-(d*f+h)),_=v>0?f:Math.min(Math.max(-f,-m),f),M=-v*v+_*(_+2*m)+p);else _=d>0?-f:f,v=Math.max(0,-(d*_+h)),M=-v*v+_*(_+2*m)+p;return r&&r.copy(this.origin).addScaledVector(this.direction,v),c&&c.copy(Kh).addScaledVector(mu,_),M}intersectSphere(e,i){if(e.radius<0)return null;La.subVectors(e.center,this.origin);const r=La.dot(this.direction),c=La.dot(La)-r*r,f=e.radius*e.radius;if(c>f)return null;const d=Math.sqrt(f-c),h=r-d,m=r+d;return m<0?null:h<0?this.at(m,i):this.at(h,i)}intersectsSphere(e){return e.radius<0?!1:this.distanceSqToPoint(e.center)<=e.radius*e.radius}distanceToPlane(e){const i=e.normal.dot(this.direction);if(i===0)return e.distanceToPoint(this.origin)===0?0:null;const r=-(this.origin.dot(e.normal)+e.constant)/i;return r>=0?r:null}intersectPlane(e,i){const r=this.distanceToPlane(e);return r===null?null:this.at(r,i)}intersectsPlane(e){const i=e.distanceToPoint(this.origin);return i===0||e.normal.dot(this.direction)*i<0}intersectBox(e,i){let r,c,f,d,h,m;const p=1/this.direction.x,x=1/this.direction.y,v=1/this.direction.z,_=this.origin;return p>=0?(r=(e.min.x-_.x)*p,c=(e.max.x-_.x)*p):(r=(e.max.x-_.x)*p,c=(e.min.x-_.x)*p),x>=0?(f=(e.min.y-_.y)*x,d=(e.max.y-_.y)*x):(f=(e.max.y-_.y)*x,d=(e.min.y-_.y)*x),r>d||f>c||((f>r||isNaN(r))&&(r=f),(d<c||isNaN(c))&&(c=d),v>=0?(h=(e.min.z-_.z)*v,m=(e.max.z-_.z)*v):(h=(e.max.z-_.z)*v,m=(e.min.z-_.z)*v),r>m||h>c)||((h>r||r!==r)&&(r=h),(m<c||c!==c)&&(c=m),c<0)?null:this.at(r>=0?r:c,i)}intersectsBox(e){return this.intersectBox(e,La)!==null}intersectTriangle(e,i,r,c,f){const d=this.origin,h=this.direction,m=h.x,p=h.y,x=h.z,v=e.x-d.x,_=e.y-d.y,M=e.z-d.z,R=i.x-d.x,N=i.y-d.y,y=i.z-d.z,S=r.x-d.x,O=r.y-d.y,k=r.z-d.z,C=Math.abs(m),D=Math.abs(p),U=Math.abs(x);let B,b,L,X,j,J,P,q,$,Y,Q,pt;if(C>=D&&C>=U?(L=m,J=v,$=R,pt=S,m>=0?(B=p,b=x,X=_,j=M,P=N,q=y,Y=O,Q=k):(B=x,b=p,X=M,j=_,P=y,q=N,Y=k,Q=O)):D>=U?(L=p,J=_,$=N,pt=O,p>=0?(B=x,b=m,X=M,j=v,P=y,q=R,Y=k,Q=S):(B=m,b=x,X=v,j=M,P=R,q=y,Y=S,Q=k)):(L=x,J=M,$=y,pt=k,x>=0?(B=m,b=p,X=v,j=_,P=R,q=N,Y=S,Q=O):(B=p,b=m,X=_,j=v,P=N,q=R,Y=O,Q=S)),L===0)return null;const ot=B/L,vt=b/L,Tt=1/L,Zt=X-ot*J,Yt=j-vt*J,H=P-ot*$,F=q-vt*$,gt=Y-ot*pt,G=Q-vt*pt,st=gt*F-G*H,At=Zt*G-Yt*gt,Dt=H*Yt-F*Zt;if(c){if(st<0||At<0||Dt<0)return null}else if((st<0||At<0||Dt<0)&&(st>0||At>0||Dt>0))return null;const xt=st+At+Dt;if(xt===0)return null;const Ct=Tt*(st*J+At*$+Dt*pt);return(xt>0?Ct<0:Ct>0)?null:this.at(Ct/xt,f)}applyMatrix4(e){return this.origin.applyMatrix4(e),this.direction.transformDirection(e),this}equals(e){return e.origin.equals(this.origin)&&e.direction.equals(this.direction)}clone(){return new this.constructor().copy(this)}}class dm extends ar{constructor(e){super(),this.isMeshBasicMaterial=!0,this.type="MeshBasicMaterial",this.color=new he(16777215),this.map=null,this.lightMap=null,this.lightMapIntensity=1,this.aoMap=null,this.aoMapIntensity=1,this.specularMap=null,this.alphaMap=null,this.envMap=null,this.envMapRotation=new Fa,this.combine=$p,this.reflectivity=1,this.refractionRatio=.98,this.wireframe=!1,this.wireframeLinewidth=1,this.wireframeLinecap="round",this.wireframeLinejoin="round",this.fog=!0,this.setValues(e)}copy(e){return super.copy(e),this.color.copy(e.color),this.map=e.map,this.lightMap=e.lightMap,this.lightMapIntensity=e.lightMapIntensity,this.aoMap=e.aoMap,this.aoMapIntensity=e.aoMapIntensity,this.specularMap=e.specularMap,this.alphaMap=e.alphaMap,this.envMap=e.envMap,this.envMapRotation.copy(e.envMapRotation),this.combine=e.combine,this.reflectivity=e.reflectivity,this.refractionRatio=e.refractionRatio,this.wireframe=e.wireframe,this.wireframeLinewidth=e.wireframeLinewidth,this.wireframeLinecap=e.wireframeLinecap,this.wireframeLinejoin=e.wireframeLinejoin,this.fog=e.fog,this}}const tx=new nn,js=new Eb,vu=new zl,ex=new dt,_u=new dt,xu=new dt,Su=new dt,Qh=new dt,yu=new dt,nx=new dt,Mu=new dt;class _n extends Rn{constructor(e=new ki,i=new dm){super(),this.isMesh=!0,this.type="Mesh",this.geometry=e,this.material=i,this.morphTargetDictionary=void 0,this.morphTargetInfluences=void 0,this.count=1,this.updateMorphTargets()}copy(e,i){return super.copy(e,i),e.morphTargetInfluences!==void 0&&(this.morphTargetInfluences=e.morphTargetInfluences.slice()),e.morphTargetDictionary!==void 0&&(this.morphTargetDictionary=Object.assign({},e.morphTargetDictionary)),this.material=Array.isArray(e.material)?e.material.slice():e.material,this.geometry=e.geometry,this}updateMorphTargets(){const i=this.geometry.morphAttributes,r=Object.keys(i);if(r.length>0){const c=i[r[0]];if(c!==void 0){this.morphTargetInfluences=[],this.morphTargetDictionary={};for(let f=0,d=c.length;f<d;f++){const h=c[f].name||String(f);this.morphTargetInfluences.push(0),this.morphTargetDictionary[h]=f}}}}getVertexPosition(e,i){const r=this.geometry,c=r.attributes.position,f=r.morphAttributes.position,d=r.morphTargetsRelative;i.fromBufferAttribute(c,e);const h=this.morphTargetInfluences;if(f&&h){yu.set(0,0,0);for(let m=0,p=f.length;m<p;m++){const x=h[m],v=f[m];x!==0&&(Qh.fromBufferAttribute(v,e),d?yu.addScaledVector(Qh,x):yu.addScaledVector(Qh.sub(i),x))}i.add(yu)}return i}intersectsFrustum(e){return e.intersectsObject(this)}raycast(e,i){const r=this.geometry,c=this.material,f=this.matrixWorld;c!==void 0&&(r.boundingSphere===null&&r.computeBoundingSphere(),vu.copy(r.boundingSphere),vu.applyMatrix4(f),js.copy(e.ray).recast(e.near),!(vu.containsPoint(js.origin)===!1&&(js.intersectSphere(vu,ex)===null||js.origin.distanceToSquared(ex)>(e.far-e.near)**2))&&(tx.copy(f).invert(),js.copy(e.ray).applyMatrix4(tx),!(r.boundingBox!==null&&js.intersectsBox(r.boundingBox)===!1)&&this._computeIntersections(e,i,js)))}_computeIntersections(e,i,r){let c;const f=this.geometry,d=this.material,h=f.index,m=f.attributes.position,p=f.attributes.uv,x=f.attributes.uv1,v=f.attributes.normal,_=f.groups,M=f.drawRange;if(h!==null)if(Array.isArray(d))for(let R=0,N=_.length;R<N;R++){const y=_[R],S=d[y.materialIndex],O=Math.max(y.start,M.start),k=Math.min(h.count,Math.min(y.start+y.count,M.start+M.count));for(let C=O,D=k;C<D;C+=3){const U=h.getX(C),B=h.getX(C+1),b=h.getX(C+2);c=Eu(this,S,e,r,p,x,v,U,B,b),c&&(c.faceIndex=Math.floor(C/3),c.face.materialIndex=y.materialIndex,i.push(c))}}else{const R=Math.max(0,M.start),N=Math.min(h.count,M.start+M.count);for(let y=R,S=N;y<S;y+=3){const O=h.getX(y),k=h.getX(y+1),C=h.getX(y+2);c=Eu(this,d,e,r,p,x,v,O,k,C),c&&(c.faceIndex=Math.floor(y/3),i.push(c))}}else if(m!==void 0)if(Array.isArray(d))for(let R=0,N=_.length;R<N;R++){const y=_[R],S=d[y.materialIndex],O=Math.max(y.start,M.start),k=Math.min(m.count,Math.min(y.start+y.count,M.start+M.count));for(let C=O,D=k;C<D;C+=3){const U=C,B=C+1,b=C+2;c=Eu(this,S,e,r,p,x,v,U,B,b),c&&(c.faceIndex=Math.floor(C/3),c.face.materialIndex=y.materialIndex,i.push(c))}}else{const R=Math.max(0,M.start),N=Math.min(m.count,M.start+M.count);for(let y=R,S=N;y<S;y+=3){const O=y,k=y+1,C=y+2;c=Eu(this,d,e,r,p,x,v,O,k,C),c&&(c.faceIndex=Math.floor(y/3),i.push(c))}}}}function bb(o,e,i,r,c,f,d,h){let m;if(e.side===qn?m=r.intersectTriangle(d,f,c,!0,h):m=r.intersectTriangle(c,f,d,e.side===$s,h),m===null)return null;Mu.copy(h),Mu.applyMatrix4(o.matrixWorld);const p=i.ray.origin.distanceTo(Mu);return p<i.near||p>i.far?null:{distance:p,point:Mu.clone(),object:o}}function Eu(o,e,i,r,c,f,d,h,m,p){o.getVertexPosition(h,_u),o.getVertexPosition(m,xu),o.getVertexPosition(p,Su);const x=bb(o,e,i,r,_u,xu,Su,nx);if(x){const v=new dt;zi.getBarycoord(nx,_u,xu,Su,v),c&&(x.uv=zi.getInterpolatedAttribute(c,h,m,p,v,new Re)),f&&(x.uv1=zi.getInterpolatedAttribute(f,h,m,p,v,new Re)),d&&(x.normal=zi.getInterpolatedAttribute(d,h,m,p,v,new dt),x.normal.dot(r.direction)>0&&x.normal.multiplyScalar(-1));const _={a:h,b:m,c:p,normal:new dt,materialIndex:0};zi.getNormal(_u,xu,Su,_.normal),x.face=_,x.barycoord=v}return x}class uS extends Fn{constructor(e=null,i=1,r=1,c,f,d,h,m,p=On,x=On,v,_){super(null,d,h,m,p,x,c,f,v,_),this.isDataTexture=!0,this.image={data:e,width:i,height:r},this.generateMipmaps=!1,this.flipY=!1,this.unpackAlignment=1}}class ix extends ua{constructor(e,i,r,c=1){super(e,i,r),this.isInstancedBufferAttribute=!0,this.meshPerAttribute=c}copy(e){return super.copy(e),this.meshPerAttribute=e.meshPerAttribute,this}toJSON(){const e=super.toJSON();return e.meshPerAttribute=this.meshPerAttribute,e.isInstancedBufferAttribute=!0,e}}const no=new nn,ax=new nn,bu=[],sx=new ir,Tb=new nn,El=new _n,bl=new zl;class Ab extends _n{constructor(e,i,r){super(e,i),this.isInstancedMesh=!0,this.instanceMatrix=new ix(new Float32Array(r*16),16),this.instanceColor=null,this.morphTexture=null,this.count=r,this.boundingBox=null,this.boundingSphere=null;for(let c=0;c<r;c++)this.setMatrixAt(c,Tb)}computeBoundingBox(){const e=this.geometry,i=this.count;this.boundingBox===null&&(this.boundingBox=new ir),e.boundingBox===null&&e.computeBoundingBox(),this.boundingBox.makeEmpty();for(let r=0;r<i;r++)this.getMatrixAt(r,no),sx.copy(e.boundingBox).applyMatrix4(no),this.boundingBox.union(sx)}computeBoundingSphere(){const e=this.geometry,i=this.count;this.boundingSphere===null&&(this.boundingSphere=new zl),e.boundingSphere===null&&e.computeBoundingSphere(),this.boundingSphere.makeEmpty();for(let r=0;r<i;r++)this.getMatrixAt(r,no),bl.copy(e.boundingSphere).applyMatrix4(no),this.boundingSphere.union(bl)}copy(e,i){return super.copy(e,i),this.instanceMatrix.copy(e.instanceMatrix),e.morphTexture!==null&&(this.morphTexture=e.morphTexture.clone()),e.instanceColor!==null&&(this.instanceColor=e.instanceColor.clone()),this.count=e.count,e.boundingBox!==null&&(this.boundingBox=e.boundingBox.clone()),e.boundingSphere!==null&&(this.boundingSphere=e.boundingSphere.clone()),this}getColorAt(e,i){return this.instanceColor===null?i.setRGB(1,1,1):i.fromArray(this.instanceColor.array,e*3)}getMatrixAt(e,i){return i.fromArray(this.instanceMatrix.array,e*16)}getMorphAt(e,i){const r=i.morphTargetInfluences,c=this.morphTexture.source.data.data,f=r.length+1,d=e*f+1;for(let h=0;h<r.length;h++)r[h]=c[d+h]}raycast(e,i){const r=this.matrixWorld,c=this.count;if(El.geometry=this.geometry,El.material=this.material,El.material!==void 0&&(this.boundingSphere===null&&this.computeBoundingSphere(),bl.copy(this.boundingSphere),bl.applyMatrix4(r),e.ray.intersectsSphere(bl)!==!1))for(let f=0;f<c;f++){this.getMatrixAt(f,no),ax.multiplyMatrices(r,no),El.matrixWorld=ax,El.raycast(e,bu);for(let d=0,h=bu.length;d<h;d++){const m=bu[d];m.instanceId=f,m.object=this,i.push(m)}bu.length=0}}setColorAt(e,i){return this.instanceColor===null&&(this.instanceColor=new ix(new Float32Array(this.instanceMatrix.count*3).fill(1),3)),i.toArray(this.instanceColor.array,e*3),this}setMatrixAt(e,i){return i.toArray(this.instanceMatrix.array,e*16),this}setMorphAt(e,i){const r=i.morphTargetInfluences,c=r.length+1;this.morphTexture===null&&(this.morphTexture=new uS(new Float32Array(c*this.count),c,this.count,am,Hi));const f=this.morphTexture.source.data.data;let d=0;for(let p=0;p<r.length;p++)d+=r[p];const h=this.geometry.morphTargetsRelative?1:1-d,m=c*e;return f[m]=h,f.set(r,m+1),this}updateMorphTargets(){}dispose(){super.dispose(),this.morphTexture!==null&&(this.morphTexture.dispose(),this.morphTexture=null)}}const Zs=new zl,Rb=new Re(.5,.5),Tu=new dt;class hm{constructor(e=new Ss,i=new Ss,r=new Ss,c=new Ss,f=new Ss,d=new Ss){this.planes=[e,i,r,c,f,d]}set(e,i,r,c,f,d){const h=this.planes;return h[0].copy(e),h[1].copy(i),h[2].copy(r),h[3].copy(c),h[4].copy(f),h[5].copy(d),this}copy(e){const i=this.planes;for(let r=0;r<6;r++)i[r].copy(e.planes[r]);return this}setFromProjectionMatrix(e,i=la,r=!1){const c=this.planes,f=e.elements,d=f[0],h=f[1],m=f[2],p=f[3],x=f[4],v=f[5],_=f[6],M=f[7],R=f[8],N=f[9],y=f[10],S=f[11],O=f[12],k=f[13],C=f[14],D=f[15];if(c[0].setComponents(p-d,M-x,S-R,D-O).normalize(),c[1].setComponents(p+d,M+x,S+R,D+O).normalize(),c[2].setComponents(p+h,M+v,S+N,D+k).normalize(),c[3].setComponents(p-h,M-v,S-N,D-k).normalize(),r)c[4].setComponents(m,_,y,C).normalize(),c[5].setComponents(p-m,M-_,S-y,D-C).normalize();else if(c[4].setComponents(p-m,M-_,S-y,D-C).normalize(),i===la)c[5].setComponents(p+m,M+_,S+y,D+C).normalize();else if(i===Pl)c[5].setComponents(m,_,y,C).normalize();else throw new Error("THREE.Frustum.setFromProjectionMatrix(): Invalid coordinate system: "+i);return this}intersectsObject(e){if(e.boundingSphere!==void 0)e.boundingSphere===null&&e.computeBoundingSphere(),Zs.copy(e.boundingSphere).applyMatrix4(e.matrixWorld);else{const i=e.geometry;i.boundingSphere===null&&i.computeBoundingSphere(),Zs.copy(i.boundingSphere).applyMatrix4(e.matrixWorld)}return this.intersectsSphere(Zs)}intersectsSprite(e){Zs.center.set(0,0,0);const i=Rb.distanceTo(e.center);return Zs.radius=.7071067811865476+i,Zs.applyMatrix4(e.matrixWorld),this.intersectsSphere(Zs)}intersectsSphere(e){const i=this.planes,r=e.center,c=-e.radius;for(let f=0;f<6;f++)if(i[f].distanceToPoint(r)<c)return!1;return!0}intersectsBox(e){const i=this.planes;for(let r=0;r<6;r++){const c=i[r];if(Tu.x=c.normal.x>0?e.max.x:e.min.x,Tu.y=c.normal.y>0?e.max.y:e.min.y,Tu.z=c.normal.z>0?e.max.z:e.min.z,c.distanceToPoint(Tu)<0)return!1}return!0}containsPoint(e){const i=this.planes;for(let r=0;r<6;r++)if(i[r].distanceToPoint(e)<0)return!1;return!0}clone(){return new this.constructor().copy(this)}}class fS extends Fn{constructor(e=[],i=tr,r,c,f,d,h,m,p,x){super(e,i,r,c,f,d,h,m,p,x),this.isCubeTexture=!0,this.flipY=!1}get images(){return this.image}set images(e){this.image=e}}class rx extends Fn{constructor(e,i,r,c,f,d,h,m,p){super(e,i,r,c,f,d,h,m,p),this.isCanvasTexture=!0,this.needsUpdate=!0}}class Il extends Fn{constructor(e,i,r=fa,c,f,d,h=On,m=On,p,x=za,v=1){if(x!==za&&x!==Js)throw new Error("THREE.DepthTexture: format must be either THREE.DepthFormat or THREE.DepthStencilFormat");const _={width:e,height:i,depth:v};super(_,c,f,d,h,m,x,r,p),this.isDepthTexture=!0,this.flipY=!1,this.generateMipmaps=!1,this.compareFunction=null}copy(e){return super.copy(e),this.source=new um(Object.assign({},e.image)),this.compareFunction=e.compareFunction,this}toJSON(e){const i=super.toJSON(e);return i.compareFunction=this.compareFunction,i}}class wb extends Il{constructor(e,i=fa,r=tr,c,f,d=On,h=On,m,p=za){const x={width:e,height:e,depth:1},v=[x,x,x,x,x,x];super(e,e,i,r,c,f,d,h,m,p),this.image=v,this.isCubeDepthTexture=!0,this.isCubeTexture=!0}get images(){return this.image}set images(e){this.image=e}}class dS extends Fn{constructor(e=null){super(),this.sourceTexture=e,this.isExternalTexture=!0}copy(e){return super.copy(e),this.sourceTexture=e.sourceTexture,this}}class Ms extends ki{constructor(e=1,i=1,r=1,c=1,f=1,d=1){super(),this.type="BoxGeometry",this.parameters={width:e,height:i,depth:r,widthSegments:c,heightSegments:f,depthSegments:d};const h=this;c=Math.floor(c),f=Math.floor(f),d=Math.floor(d);const m=[],p=[],x=[],v=[];let _=0,M=0;R("z","y","x",-1,-1,r,i,e,d,f,0),R("z","y","x",1,-1,r,i,-e,d,f,1),R("x","z","y",1,1,e,r,i,c,d,2),R("x","z","y",1,-1,e,r,-i,c,d,3),R("x","y","z",1,-1,e,i,r,c,f,4),R("x","y","z",-1,-1,e,i,-r,c,f,5),this.setIndex(m),this.setAttribute("position",new Yn(p,3)),this.setAttribute("normal",new Yn(x,3)),this.setAttribute("uv",new Yn(v,2));function R(N,y,S,O,k,C,D,U,B,b,L){const X=C/B,j=D/b,J=C/2,P=D/2,q=U/2,$=B+1,Y=b+1;let Q=0,pt=0;const ot=new dt;for(let vt=0;vt<Y;vt++){const Tt=vt*j-P;for(let Zt=0;Zt<$;Zt++){const Yt=Zt*X-J;ot[N]=Yt*O,ot[y]=Tt*k,ot[S]=q,p.push(ot.x,ot.y,ot.z),ot[N]=0,ot[y]=0,ot[S]=U>0?1:-1,x.push(ot.x,ot.y,ot.z),v.push(Zt/B),v.push(1-vt/b),Q+=1}}for(let vt=0;vt<b;vt++)for(let Tt=0;Tt<B;Tt++){const Zt=_+Tt+$*vt,Yt=_+Tt+$*(vt+1),H=_+(Tt+1)+$*(vt+1),F=_+(Tt+1)+$*vt;m.push(Zt,Yt,F),m.push(Yt,H,F),pt+=6}h.addGroup(M,pt,L),M+=pt,_+=Q}}copy(e){return super.copy(e),this.parameters=Object.assign({},e.parameters),this}static fromJSON(e){return new Ms(e.width,e.height,e.depth,e.widthSegments,e.heightSegments,e.depthSegments)}}class mo extends ki{constructor(e=1,i=1,r=1,c=32,f=1,d=!1,h=0,m=Math.PI*2){super(),this.type="CylinderGeometry",this.parameters={radiusTop:e,radiusBottom:i,height:r,radialSegments:c,heightSegments:f,openEnded:d,thetaStart:h,thetaLength:m};const p=this;c=Math.floor(c),f=Math.floor(f);const x=[],v=[],_=[],M=[];let R=0;const N=[],y=r/2;let S=0;O(),d===!1&&(e>0&&k(!0),i>0&&k(!1)),this.setIndex(x),this.setAttribute("position",new Yn(v,3)),this.setAttribute("normal",new Yn(_,3)),this.setAttribute("uv",new Yn(M,2));function O(){const C=new dt,D=new dt;let U=0;const B=(i-e)/r;for(let b=0;b<=f;b++){const L=[],X=b/f,j=X*(i-e)+e;for(let J=0;J<=c;J++){const P=J/c,q=P*m+h,$=Math.sin(q),Y=Math.cos(q);D.x=j*$,D.y=-X*r+y,D.z=j*Y,v.push(D.x,D.y,D.z),C.set($,B,Y).normalize(),_.push(C.x,C.y,C.z),M.push(P,1-X),L.push(R++)}N.push(L)}for(let b=0;b<c;b++)for(let L=0;L<f;L++){const X=N[L][b],j=N[L+1][b],J=N[L+1][b+1],P=N[L][b+1];(e>0||L!==0)&&(x.push(X,j,P),U+=3),(i>0||L!==f-1)&&(x.push(j,J,P),U+=3)}p.addGroup(S,U,0),S+=U}function k(C){const D=R,U=new Re,B=new dt;let b=0;const L=C===!0?e:i,X=C===!0?1:-1;for(let J=1;J<=c;J++)v.push(0,y*X,0),_.push(0,X,0),M.push(.5,.5),R++;const j=R;for(let J=0;J<=c;J++){const q=J/c*m+h,$=Math.cos(q),Y=Math.sin(q);B.x=L*Y,B.y=y*X,B.z=L*$,v.push(B.x,B.y,B.z),_.push(0,X,0),U.x=$*.5+.5,U.y=Y*.5*X+.5,M.push(U.x,U.y),R++}for(let J=0;J<c;J++){const P=D+J,q=j+J;C===!0?x.push(q,q+1,P):x.push(q+1,q,P),b+=3}p.addGroup(S,b,C===!0?1:2),S+=b}}copy(e){return super.copy(e),this.parameters=Object.assign({},e.parameters),this}static fromJSON(e){return new mo(e.radiusTop,e.radiusBottom,e.height,e.radialSegments,e.heightSegments,e.openEnded,e.thetaStart,e.thetaLength)}}class go extends ki{constructor(e=1,i=1,r=1,c=1){super(),this.type="PlaneGeometry",this.parameters={width:e,height:i,widthSegments:r,heightSegments:c};const f=e/2,d=i/2,h=Math.floor(r),m=Math.floor(c),p=h+1,x=m+1,v=e/h,_=i/m,M=[],R=[],N=[],y=[];for(let S=0;S<x;S++){const O=S*_-d;for(let k=0;k<p;k++){const C=k*v-f;R.push(C,-O,0),N.push(0,0,1),y.push(k/h),y.push(1-S/m)}}for(let S=0;S<m;S++)for(let O=0;O<h;O++){const k=O+p*S,C=O+p*(S+1),D=O+1+p*(S+1),U=O+1+p*S;M.push(k,C,U),M.push(C,D,U)}this.setIndex(M),this.setAttribute("position",new Yn(R,3)),this.setAttribute("normal",new Yn(N,3)),this.setAttribute("uv",new Yn(y,2))}copy(e){return super.copy(e),this.parameters=Object.assign({},e.parameters),this}static fromJSON(e){return new go(e.width,e.height,e.widthSegments,e.heightSegments)}}class pm extends ki{constructor(e=1,i=32,r=16,c=0,f=Math.PI*2,d=0,h=Math.PI){super(),this.type="SphereGeometry",this.parameters={radius:e,widthSegments:i,heightSegments:r,phiStart:c,phiLength:f,thetaStart:d,thetaLength:h},i=Math.max(3,Math.floor(i)),r=Math.max(2,Math.floor(r));const m=Math.min(d+h,Math.PI);let p=0;const x=[],v=new dt,_=new dt,M=[],R=[],N=[],y=[];for(let S=0;S<=r;S++){const O=[],k=S/r,C=d+k*h,D=e*Math.cos(C),U=Math.sqrt(e*e-D*D);let B=0;S===0&&d===0?B=.5/i:S===r&&m===Math.PI&&(B=-.5/i);for(let b=0;b<=i;b++){const L=b/i,X=c+L*f;v.x=-U*Math.cos(X),v.y=D,v.z=U*Math.sin(X),R.push(v.x,v.y,v.z),_.copy(v).normalize(),N.push(_.x,_.y,_.z),y.push(L+B,1-k),O.push(p++)}x.push(O)}for(let S=0;S<r;S++)for(let O=0;O<i;O++){const k=x[S][O+1],C=x[S][O],D=x[S+1][O],U=x[S+1][O+1];(S!==0||d>0)&&M.push(k,C,U),(S!==r-1||m<Math.PI)&&M.push(C,D,U)}this.setIndex(M),this.setAttribute("position",new Yn(R,3)),this.setAttribute("normal",new Yn(N,3)),this.setAttribute("uv",new Yn(y,2))}copy(e){return super.copy(e),this.parameters=Object.assign({},e.parameters),this}static fromJSON(e){return new pm(e.radius,e.widthSegments,e.heightSegments,e.phiStart,e.phiLength,e.thetaStart,e.thetaLength)}}class Cb extends ar{constructor(e){super(),this.isShadowMaterial=!0,this.type="ShadowMaterial",this.color=new he(0),this.transparent=!0,this.fog=!0,this.setValues(e)}copy(e){return super.copy(e),this.color.copy(e.color),this.fog=e.fog,this}}function vo(o){const e={};for(const i in o){e[i]={};for(const r in o[i]){const c=o[i][r];if(ox(c))c.isRenderTargetTexture?(oe("UniformsUtils: Textures of render targets cannot be cloned via cloneUniforms() or mergeUniforms()."),e[i][r]=null):e[i][r]=c.clone();else if(Array.isArray(c))if(ox(c[0])){const f=[];for(let d=0,h=c.length;d<h;d++)f[d]=c[d].clone();e[i][r]=f}else e[i][r]=c.slice();else e[i][r]=c}}return e}function Wn(o){const e={};for(let i=0;i<o.length;i++){const r=vo(o[i]);for(const c in r)e[c]=r[c]}return e}function ox(o){return o&&(o.isColor||o.isMatrix3||o.isMatrix4||o.isVector2||o.isVector3||o.isVector4||o.isTexture||o.isQuaternion)}function Nb(o){const e=[];for(let i=0;i<o.length;i++)e.push(o[i].clone());return e}function hS(o){const e=o.getRenderTarget();return e===null?o.outputColorSpace:e.isXRRenderTarget===!0?e.texture.colorSpace:Pe.workingColorSpace}const Db={clone:vo,merge:Wn};var Ub=`void main() {
	gl_Position = projectionMatrix * modelViewMatrix * vec4( position, 1.0 );
}`,Lb=`void main() {
	gl_FragColor = vec4( 1.0, 0.0, 0.0, 1.0 );
}`;class ha extends ar{constructor(e){super(),this.isShaderMaterial=!0,this.type="ShaderMaterial",this.defines={},this.uniforms={},this.uniformsGroups=[],this.vertexShader=Ub,this.fragmentShader=Lb,this.linewidth=1,this.wireframe=!1,this.wireframeLinewidth=1,this.fog=!1,this.lights=!1,this.clipping=!1,this.forceSinglePass=!0,this.extensions={clipCullDistance:!1,multiDraw:!1},this.defaultAttributeValues={color:[1,1,1],uv:[0,0],uv1:[0,0]},this.index0AttributeName=void 0,this.uniformsNeedUpdate=!1,this.glslVersion=null,e!==void 0&&this.setValues(e)}copy(e){return super.copy(e),this.fragmentShader=e.fragmentShader,this.vertexShader=e.vertexShader,this.uniforms=vo(e.uniforms),this.uniformsGroups=Nb(e.uniformsGroups),this.defines=Object.assign({},e.defines),this.wireframe=e.wireframe,this.wireframeLinewidth=e.wireframeLinewidth,this.fog=e.fog,this.lights=e.lights,this.clipping=e.clipping,this.extensions=Object.assign({},e.extensions),this.glslVersion=e.glslVersion,this.defaultAttributeValues=Object.assign({},e.defaultAttributeValues),this.index0AttributeName=e.index0AttributeName,this.uniformsNeedUpdate=e.uniformsNeedUpdate,this}toJSON(e){const i=super.toJSON(e);i.glslVersion=this.glslVersion,i.uniforms={};for(const c in this.uniforms){const d=this.uniforms[c].value;d&&d.isTexture?i.uniforms[c]={type:"t",value:d.toJSON(e).uuid}:d&&d.isColor?i.uniforms[c]={type:"c",value:d.getHex()}:d&&d.isVector2?i.uniforms[c]={type:"v2",value:d.toArray()}:d&&d.isVector3?i.uniforms[c]={type:"v3",value:d.toArray()}:d&&d.isVector4?i.uniforms[c]={type:"v4",value:d.toArray()}:d&&d.isMatrix3?i.uniforms[c]={type:"m3",value:d.toArray()}:d&&d.isMatrix4?i.uniforms[c]={type:"m4",value:d.toArray()}:i.uniforms[c]={value:d}}Object.keys(this.defines).length>0&&(i.defines=this.defines),i.vertexShader=this.vertexShader,i.fragmentShader=this.fragmentShader,i.lights=this.lights,i.clipping=this.clipping;const r={};for(const c in this.extensions)this.extensions[c]===!0&&(r[c]=!0);return Object.keys(r).length>0&&(i.extensions=r),i}fromJSON(e,i){if(super.fromJSON(e,i),e.uniforms!==void 0)for(const r in e.uniforms){const c=e.uniforms[r];switch(this.uniforms[r]={},c.type){case"t":this.uniforms[r].value=i[c.value]||null;break;case"c":this.uniforms[r].value=new he().setHex(c.value);break;case"v2":this.uniforms[r].value=new Re().fromArray(c.value);break;case"v3":this.uniforms[r].value=new dt().fromArray(c.value);break;case"v4":this.uniforms[r].value=new on().fromArray(c.value);break;case"m3":this.uniforms[r].value=new de().fromArray(c.value);break;case"m4":this.uniforms[r].value=new nn().fromArray(c.value);break;default:this.uniforms[r].value=c.value}}if(e.defines!==void 0&&(this.defines=e.defines),e.vertexShader!==void 0&&(this.vertexShader=e.vertexShader),e.fragmentShader!==void 0&&(this.fragmentShader=e.fragmentShader),e.glslVersion!==void 0&&(this.glslVersion=e.glslVersion),e.extensions!==void 0)for(const r in e.extensions)this.extensions[r]=e.extensions[r];return e.lights!==void 0&&(this.lights=e.lights),e.clipping!==void 0&&(this.clipping=e.clipping),this}}class Ob extends ha{constructor(e){super(e),this.isRawShaderMaterial=!0,this.type="RawShaderMaterial"}}class Fi extends ar{constructor(e){super(),this.isMeshStandardMaterial=!0,this.type="MeshStandardMaterial",this.defines={STANDARD:""},this.color=new he(16777215),this.roughness=1,this.metalness=0,this.map=null,this.lightMap=null,this.lightMapIntensity=1,this.aoMap=null,this.aoMapIntensity=1,this.emissive=new he(0),this.emissiveIntensity=1,this.emissiveMap=null,this.bumpMap=null,this.bumpScale=1,this.normalMap=null,this.normalMapType=zu,this.normalScale=new Re(1,1),this.displacementMap=null,this.displacementScale=1,this.displacementBias=0,this.roughnessMap=null,this.metalnessMap=null,this.alphaMap=null,this.envMap=null,this.envMapRotation=new Fa,this.envMapIntensity=1,this.wireframe=!1,this.wireframeLinewidth=1,this.wireframeLinecap="round",this.wireframeLinejoin="round",this.flatShading=!1,this.fog=!0,this.setValues(e)}copy(e){return super.copy(e),this.defines={STANDARD:""},this.color.copy(e.color),this.roughness=e.roughness,this.metalness=e.metalness,this.map=e.map,this.lightMap=e.lightMap,this.lightMapIntensity=e.lightMapIntensity,this.aoMap=e.aoMap,this.aoMapIntensity=e.aoMapIntensity,this.emissive.copy(e.emissive),this.emissiveMap=e.emissiveMap,this.emissiveIntensity=e.emissiveIntensity,this.bumpMap=e.bumpMap,this.bumpScale=e.bumpScale,this.normalMap=e.normalMap,this.normalMapType=e.normalMapType,this.normalScale.copy(e.normalScale),this.displacementMap=e.displacementMap,this.displacementScale=e.displacementScale,this.displacementBias=e.displacementBias,this.roughnessMap=e.roughnessMap,this.metalnessMap=e.metalnessMap,this.alphaMap=e.alphaMap,this.envMap=e.envMap,this.envMapRotation.copy(e.envMapRotation),this.envMapIntensity=e.envMapIntensity,this.wireframe=e.wireframe,this.wireframeLinewidth=e.wireframeLinewidth,this.wireframeLinecap=e.wireframeLinecap,this.wireframeLinejoin=e.wireframeLinejoin,this.flatShading=e.flatShading,this.fog=e.fog,this}}class lx extends Fi{constructor(e){super(),this.isMeshPhysicalMaterial=!0,this.defines={STANDARD:"",PHYSICAL:""},this.type="MeshPhysicalMaterial",this.anisotropyRotation=0,this.anisotropyMap=null,this.clearcoatMap=null,this.clearcoatRoughness=0,this.clearcoatRoughnessMap=null,this.clearcoatNormalScale=new Re(1,1),this.clearcoatNormalMap=null,this.ior=1.5,Object.defineProperty(this,"reflectivity",{get:function(){return De(2.5*(this.ior-1)/(this.ior+1),0,1)},set:function(i){this.ior=(1+.4*i)/(1-.4*i)}}),this.iridescenceMap=null,this.iridescenceIOR=1.3,this.iridescenceThicknessRange=[100,400],this.iridescenceThicknessMap=null,this.sheenColor=new he(0),this.sheenColorMap=null,this.sheenRoughness=1,this.sheenRoughnessMap=null,this.transmissionMap=null,this.thickness=0,this.thicknessMap=null,this.attenuationDistance=1/0,this.attenuationColor=new he(1,1,1),this.specularIntensity=1,this.specularIntensityMap=null,this.specularColor=new he(1,1,1),this.specularColorMap=null,this._anisotropy=0,this._clearcoat=0,this._dispersion=0,this._iridescence=0,this._retroreflectivity=0,this._sheen=0,this._transmission=0,this.setValues(e)}get anisotropy(){return this._anisotropy}set anisotropy(e){this._anisotropy>0!=e>0&&this.version++,this._anisotropy=e}get clearcoat(){return this._clearcoat}set clearcoat(e){this._clearcoat>0!=e>0&&this.version++,this._clearcoat=e}get iridescence(){return this._iridescence}set iridescence(e){this._iridescence>0!=e>0&&this.version++,this._iridescence=e}get dispersion(){return this._dispersion}set dispersion(e){this._dispersion>0!=e>0&&this.version++,this._dispersion=e}get retroreflectivity(){return this._retroreflectivity}set retroreflectivity(e){this._retroreflectivity>0!=e>0&&this.version++,this._retroreflectivity=e}get sheen(){return this._sheen}set sheen(e){this._sheen>0!=e>0&&this.version++,this._sheen=e}get transmission(){return this._transmission}set transmission(e){this._transmission>0!=e>0&&this.version++,this._transmission=e}copy(e){return super.copy(e),this.defines={STANDARD:"",PHYSICAL:""},this.anisotropy=e.anisotropy,this.anisotropyRotation=e.anisotropyRotation,this.anisotropyMap=e.anisotropyMap,this.clearcoat=e.clearcoat,this.clearcoatMap=e.clearcoatMap,this.clearcoatRoughness=e.clearcoatRoughness,this.clearcoatRoughnessMap=e.clearcoatRoughnessMap,this.clearcoatNormalMap=e.clearcoatNormalMap,this.clearcoatNormalScale.copy(e.clearcoatNormalScale),this.dispersion=e.dispersion,this.ior=e.ior,this.iridescence=e.iridescence,this.iridescenceMap=e.iridescenceMap,this.iridescenceIOR=e.iridescenceIOR,this.iridescenceThicknessRange=[...e.iridescenceThicknessRange],this.iridescenceThicknessMap=e.iridescenceThicknessMap,this.retroreflectivity=e.retroreflectivity,this.sheen=e.sheen,this.sheenColor.copy(e.sheenColor),this.sheenColorMap=e.sheenColorMap,this.sheenRoughness=e.sheenRoughness,this.sheenRoughnessMap=e.sheenRoughnessMap,this.transmission=e.transmission,this.transmissionMap=e.transmissionMap,this.thickness=e.thickness,this.thicknessMap=e.thicknessMap,this.attenuationDistance=e.attenuationDistance,this.attenuationColor.copy(e.attenuationColor),this.specularIntensity=e.specularIntensity,this.specularIntensityMap=e.specularIntensityMap,this.specularColor.copy(e.specularColor),this.specularColorMap=e.specularColorMap,this}}class Pb extends ar{constructor(e){super(),this.isMeshLambertMaterial=!0,this.type="MeshLambertMaterial",this.color=new he(16777215),this.map=null,this.lightMap=null,this.lightMapIntensity=1,this.aoMap=null,this.aoMapIntensity=1,this.emissive=new he(0),this.emissiveIntensity=1,this.emissiveMap=null,this.bumpMap=null,this.bumpScale=1,this.normalMap=null,this.normalMapType=zu,this.normalScale=new Re(1,1),this.displacementMap=null,this.displacementScale=1,this.displacementBias=0,this.specularMap=null,this.alphaMap=null,this.envMap=null,this.envMapRotation=new Fa,this.combine=$p,this.reflectivity=1,this.envMapIntensity=1,this.refractionRatio=.98,this.wireframe=!1,this.wireframeLinewidth=1,this.wireframeLinecap="round",this.wireframeLinejoin="round",this.flatShading=!1,this.fog=!0,this.setValues(e)}copy(e){return super.copy(e),this.color.copy(e.color),this.map=e.map,this.lightMap=e.lightMap,this.lightMapIntensity=e.lightMapIntensity,this.aoMap=e.aoMap,this.aoMapIntensity=e.aoMapIntensity,this.emissive.copy(e.emissive),this.emissiveMap=e.emissiveMap,this.emissiveIntensity=e.emissiveIntensity,this.bumpMap=e.bumpMap,this.bumpScale=e.bumpScale,this.normalMap=e.normalMap,this.normalMapType=e.normalMapType,this.normalScale.copy(e.normalScale),this.displacementMap=e.displacementMap,this.displacementScale=e.displacementScale,this.displacementBias=e.displacementBias,this.specularMap=e.specularMap,this.alphaMap=e.alphaMap,this.envMap=e.envMap,this.envMapRotation.copy(e.envMapRotation),this.combine=e.combine,this.reflectivity=e.reflectivity,this.envMapIntensity=e.envMapIntensity,this.refractionRatio=e.refractionRatio,this.wireframe=e.wireframe,this.wireframeLinewidth=e.wireframeLinewidth,this.wireframeLinecap=e.wireframeLinecap,this.wireframeLinejoin=e.wireframeLinejoin,this.flatShading=e.flatShading,this.fog=e.fog,this}}class Ib extends ar{constructor(e){super(),this.isMeshDepthMaterial=!0,this.type="MeshDepthMaterial",this.depthPacking=kE,this.map=null,this.alphaMap=null,this.displacementMap=null,this.displacementScale=1,this.displacementBias=0,this.wireframe=!1,this.wireframeLinewidth=1,this.setValues(e)}copy(e){return super.copy(e),this.depthPacking=e.depthPacking,this.map=e.map,this.alphaMap=e.alphaMap,this.displacementMap=e.displacementMap,this.displacementScale=e.displacementScale,this.displacementBias=e.displacementBias,this.wireframe=e.wireframe,this.wireframeLinewidth=e.wireframeLinewidth,this}}class Bb extends ar{constructor(e){super(),this.isMeshDistanceMaterial=!0,this.type="MeshDistanceMaterial",this.map=null,this.alphaMap=null,this.displacementMap=null,this.displacementScale=1,this.displacementBias=0,this.setValues(e)}copy(e){return super.copy(e),this.map=e.map,this.alphaMap=e.alphaMap,this.displacementMap=e.displacementMap,this.displacementScale=e.displacementScale,this.displacementBias=e.displacementBias,this}}class Xu extends Rn{constructor(e,i=1){super(),this.isLight=!0,this.type="Light",this.color=new he(e),this.intensity=i}copy(e,i){return super.copy(e,i),this.color.copy(e.color),this.intensity=e.intensity,this}toJSON(e){const i=super.toJSON(e);return i.object.color=this.color.getHex(),i.object.intensity=this.intensity,i}}class zb extends Xu{constructor(e,i,r){super(e,r),this.isHemisphereLight=!0,this.type="HemisphereLight",this.position.copy(Rn.DEFAULT_UP),this.updateMatrix(),this.groundColor=new he(i)}copy(e,i){return super.copy(e,i),this.groundColor.copy(e.groundColor),this}toJSON(e){const i=super.toJSON(e);return i.object.groundColor=this.groundColor.getHex(),i}}const Jh=new nn,cx=new dt,ux=new dt;class pS{constructor(e){this.camera=e,this.intensity=1,this.bias=0,this.biasNode=null,this.normalBias=0,this.radius=1,this.blurSamples=8,this.mapSize=new Re(512,512),this.mapType=pi,this.map=null,this.mapPass=null,this.matrix=new nn,this.autoUpdate=!0,this.needsUpdate=!1,this._frustum=new hm,this._frameExtents=new Re(1,1),this._viewportCount=1,this._viewports=[new on(0,0,1,1)]}getViewportCount(){return this._viewportCount}getCamera(){return this.camera}getFrustum(){return this._frustum}updateMatrices(e){const i=this.camera;cx.setFromMatrixPosition(e.matrixWorld),i.position.copy(cx),ux.setFromMatrixPosition(e.target.matrixWorld),i.lookAt(ux),i.updateMatrixWorld(),this._updateMatrix(i,this.matrix,this._frustum)}_updateMatrix(e,i,r,c){Jh.multiplyMatrices(e.projectionMatrix,e.matrixWorldInverse),r.setFromProjectionMatrix(Jh,e.coordinateSystem,e.reversedDepth);const f=this._frameExtents,d=c?c.z/f.x:1,h=c?c.w/f.y:1,m=c?c.x/f.x:0,p=c?c.y/f.y:0;e.coordinateSystem===Pl||e.reversedDepth?i.set(.5*d,0,0,.5*d+m,0,.5*h,0,.5*h+p,0,0,1,0,0,0,0,1):i.set(.5*d,0,0,.5*d+m,0,.5*h,0,.5*h+p,0,0,.5,.5,0,0,0,1),i.multiply(Jh)}getViewport(e){return this._viewports[e]}getFrameExtents(){return this._frameExtents}dispose(){this.map&&this.map.dispose(),this.mapPass&&this.mapPass.dispose()}copy(e){return this.camera=e.camera.clone(),this.intensity=e.intensity,this.bias=e.bias,this.radius=e.radius,this.autoUpdate=e.autoUpdate,this.needsUpdate=e.needsUpdate,this.normalBias=e.normalBias,this.blurSamples=e.blurSamples,this.mapSize.copy(e.mapSize),this.biasNode=e.biasNode,this}clone(){return new this.constructor().copy(this)}toJSON(){const e={};return e.intensity=this.intensity,e.bias=this.bias,e.normalBias=this.normalBias,e.radius=this.radius,e.blurSamples=this.blurSamples,e.mapSize=this.mapSize.toArray(),e.camera=this.camera.toJSON(!1).object,delete e.camera.matrix,e}}const Au=new dt,Ru=new _o,aa=new dt;class mS extends Rn{constructor(){super(),this.isCamera=!0,this.type="Camera",this.matrixWorldInverse=new nn,this.projectionMatrix=new nn,this.projectionMatrixInverse=new nn,this.coordinateSystem=la,this._reversedDepth=!1}get reversedDepth(){return this._reversedDepth}copy(e,i){return super.copy(e,i),this.matrixWorldInverse.copy(e.matrixWorldInverse),this.projectionMatrix.copy(e.projectionMatrix),this.projectionMatrixInverse.copy(e.projectionMatrixInverse),this.coordinateSystem=e.coordinateSystem,this}getWorldDirection(e){return super.getWorldDirection(e).negate()}updateMatrixWorld(e){super.updateMatrixWorld(e),this.matrixWorld.decompose(Au,Ru,aa),aa.x===1&&aa.y===1&&aa.z===1?this.matrixWorldInverse.copy(this.matrixWorld).invert():this.matrixWorldInverse.compose(Au,Ru,aa.set(1,1,1)).invert()}updateWorldMatrix(e,i,r=!1){super.updateWorldMatrix(e,i,r),this.matrixWorld.decompose(Au,Ru,aa),aa.x===1&&aa.y===1&&aa.z===1?this.matrixWorldInverse.copy(this.matrixWorld).invert():this.matrixWorldInverse.compose(Au,Ru,aa.set(1,1,1)).invert()}clone(){return new this.constructor().copy(this)}}const _s=new dt,fx=new Re,dx=new Re;class ei extends mS{constructor(e=50,i=1,r=.1,c=2e3){super(),this.isPerspectiveCamera=!0,this.type="PerspectiveCamera",this.fov=e,this.zoom=1,this.near=r,this.far=c,this.focus=10,this.aspect=i,this.view=null,this.filmGauge=35,this.filmOffset=0,this.updateProjectionMatrix()}copy(e,i){return super.copy(e,i),this.fov=e.fov,this.zoom=e.zoom,this.near=e.near,this.far=e.far,this.focus=e.focus,this.aspect=e.aspect,this.view=e.view===null?null:Object.assign({},e.view),this.filmGauge=e.filmGauge,this.filmOffset=e.filmOffset,this}setFocalLength(e){const i=.5*this.getFilmHeight()/e;this.fov=qp*2*Math.atan(i),this.updateProjectionMatrix()}getFocalLength(){const e=Math.tan(Nh*.5*this.fov);return .5*this.getFilmHeight()/e}getEffectiveFOV(){return qp*2*Math.atan(Math.tan(Nh*.5*this.fov)/this.zoom)}getFilmWidth(){return this.filmGauge*Math.min(this.aspect,1)}getFilmHeight(){return this.filmGauge/Math.max(this.aspect,1)}getViewBounds(e,i,r){_s.set(-1,-1,.5).applyMatrix4(this.projectionMatrixInverse),i.set(_s.x,_s.y).multiplyScalar(-e/_s.z),_s.set(1,1,.5).applyMatrix4(this.projectionMatrixInverse),r.set(_s.x,_s.y).multiplyScalar(-e/_s.z)}getViewSize(e,i){return this.getViewBounds(e,fx,dx),i.subVectors(dx,fx)}setViewOffset(e,i,r,c,f,d){this.aspect=e/i,this.view===null&&(this.view={enabled:!0,fullWidth:1,fullHeight:1,offsetX:0,offsetY:0,width:1,height:1}),this.view.enabled=!0,this.view.fullWidth=e,this.view.fullHeight=i,this.view.offsetX=r,this.view.offsetY=c,this.view.width=f,this.view.height=d,this.updateProjectionMatrix()}clearViewOffset(){this.view!==null&&(this.view.enabled=!1),this.updateProjectionMatrix()}updateProjectionMatrix(){const e=this.near;let i=e*Math.tan(Nh*.5*this.fov)/this.zoom,r=2*i,c=this.aspect*r,f=-.5*c;const d=this.view;if(this.view!==null&&this.view.enabled){const m=d.fullWidth,p=d.fullHeight;f+=d.offsetX*c/m,i-=d.offsetY*r/p,c*=d.width/m,r*=d.height/p}const h=this.filmOffset;h!==0&&(f+=e*h/this.getFilmWidth()),this.projectionMatrix.makePerspective(f,f+c,i,i-r,e,this.far,this.coordinateSystem,this.reversedDepth),this.projectionMatrixInverse.copy(this.projectionMatrix).invert()}toJSON(e){const i=super.toJSON(e);return i.object.fov=this.fov,i.object.zoom=this.zoom,i.object.near=this.near,i.object.far=this.far,i.object.focus=this.focus,i.object.aspect=this.aspect,this.view!==null&&(i.object.view=Object.assign({},this.view)),i.object.filmGauge=this.filmGauge,i.object.filmOffset=this.filmOffset,i}}class Fb extends pS{constructor(){super(new ei(90,1,.5,500)),this.isPointLightShadow=!0}}class mm extends Xu{constructor(e,i,r=0,c=2){super(e,i),this.isPointLight=!0,this.type="PointLight",this.distance=r,this.decay=c,this.shadow=new Fb}get power(){return this.intensity*4*Math.PI}set power(e){this.intensity=e/(4*Math.PI)}dispose(){super.dispose(),this.shadow.dispose()}copy(e,i){return super.copy(e,i),this.distance=e.distance,this.decay=e.decay,this.shadow=e.shadow.clone(),this}toJSON(e){const i=super.toJSON(e);return i.object.distance=this.distance,i.object.decay=this.decay,i.object.shadow=this.shadow.toJSON(),i}}class gm extends mS{constructor(e=-1,i=1,r=1,c=-1,f=.1,d=2e3){super(),this.isOrthographicCamera=!0,this.type="OrthographicCamera",this.zoom=1,this.view=null,this.left=e,this.right=i,this.top=r,this.bottom=c,this.near=f,this.far=d,this.updateProjectionMatrix()}copy(e,i){return super.copy(e,i),this.left=e.left,this.right=e.right,this.top=e.top,this.bottom=e.bottom,this.near=e.near,this.far=e.far,this.zoom=e.zoom,this.view=e.view===null?null:Object.assign({},e.view),this}setViewOffset(e,i,r,c,f,d){this.view===null&&(this.view={enabled:!0,fullWidth:1,fullHeight:1,offsetX:0,offsetY:0,width:1,height:1}),this.view.enabled=!0,this.view.fullWidth=e,this.view.fullHeight=i,this.view.offsetX=r,this.view.offsetY=c,this.view.width=f,this.view.height=d,this.updateProjectionMatrix()}clearViewOffset(){this.view!==null&&(this.view.enabled=!1),this.updateProjectionMatrix()}updateProjectionMatrix(){const e=(this.right-this.left)/(2*this.zoom),i=(this.top-this.bottom)/(2*this.zoom),r=(this.right+this.left)/2,c=(this.top+this.bottom)/2;let f=r-e,d=r+e,h=c+i,m=c-i;if(this.view!==null&&this.view.enabled){const p=(this.right-this.left)/this.view.fullWidth/this.zoom,x=(this.top-this.bottom)/this.view.fullHeight/this.zoom;f+=p*this.view.offsetX,d=f+p*this.view.width,h-=x*this.view.offsetY,m=h-x*this.view.height}this.projectionMatrix.makeOrthographic(f,d,h,m,this.near,this.far,this.coordinateSystem,this.reversedDepth),this.projectionMatrixInverse.copy(this.projectionMatrix).invert()}toJSON(e){const i=super.toJSON(e);return i.object.zoom=this.zoom,i.object.left=this.left,i.object.right=this.right,i.object.top=this.top,i.object.bottom=this.bottom,i.object.near=this.near,i.object.far=this.far,this.view!==null&&(i.object.view=Object.assign({},this.view)),i}}class Hb extends pS{constructor(){super(new gm(-5,5,5,-5,.5,500)),this.isDirectionalLightShadow=!0}}class $h extends Xu{constructor(e,i){super(e,i),this.isDirectionalLight=!0,this.type="DirectionalLight",this.position.copy(Rn.DEFAULT_UP),this.updateMatrix(),this.target=new Rn,this.shadow=new Hb}dispose(){super.dispose(),this.shadow.dispose()}copy(e){return super.copy(e),this.target=e.target.clone(),this.shadow=e.shadow.clone(),this}toJSON(e){const i=super.toJSON(e);return i.object.shadow=this.shadow.toJSON(),i.object.target=this.target.uuid,i}}class Gb extends Xu{constructor(e,i){super(e,i),this.isAmbientLight=!0,this.type="AmbientLight"}}const io=-90,ao=1;class Vb extends Rn{constructor(e,i,r){super(),this.type="CubeCamera",this.renderTarget=r,this.coordinateSystem=null,this.activeMipmapLevel=0;const c=new ei(io,ao,e,i);c.layers=this.layers,this.add(c);const f=new ei(io,ao,e,i);f.layers=this.layers,this.add(f);const d=new ei(io,ao,e,i);d.layers=this.layers,this.add(d);const h=new ei(io,ao,e,i);h.layers=this.layers,this.add(h);const m=new ei(io,ao,e,i);m.layers=this.layers,this.add(m);const p=new ei(io,ao,e,i);p.layers=this.layers,this.add(p)}updateCoordinateSystem(){const e=this.coordinateSystem,i=this.children.concat(),[r,c,f,d,h,m]=i;for(const p of i)this.remove(p);if(e===la)r.up.set(0,1,0),r.lookAt(1,0,0),c.up.set(0,1,0),c.lookAt(-1,0,0),f.up.set(0,0,-1),f.lookAt(0,1,0),d.up.set(0,0,1),d.lookAt(0,-1,0),h.up.set(0,1,0),h.lookAt(0,0,1),m.up.set(0,1,0),m.lookAt(0,0,-1);else if(e===Pl)r.up.set(0,-1,0),r.lookAt(-1,0,0),c.up.set(0,-1,0),c.lookAt(1,0,0),f.up.set(0,0,1),f.lookAt(0,1,0),d.up.set(0,0,-1),d.lookAt(0,-1,0),h.up.set(0,-1,0),h.lookAt(0,0,1),m.up.set(0,-1,0),m.lookAt(0,0,-1);else throw new Error("THREE.CubeCamera.updateCoordinateSystem(): Invalid coordinate system: "+e);for(const p of i)this.add(p),p.updateMatrixWorld()}update(e,i){this.parent===null&&this.updateMatrixWorld();const{renderTarget:r,activeMipmapLevel:c}=this;this.coordinateSystem!==e.coordinateSystem&&(this.coordinateSystem=e.coordinateSystem,this.updateCoordinateSystem());const[f,d,h,m,p,x]=this.children,v=e.getRenderTarget(),_=e.getActiveCubeFace(),M=e.getActiveMipmapLevel(),R=e.xr.enabled;e.xr.enabled=!1;const N=r.texture.generateMipmaps;r.texture.generateMipmaps=!1;let y=!1;e.isWebGLRenderer===!0?y=e.state.buffers.depth.getReversed():y=e.reversedDepthBuffer,e.setRenderTarget(r,0,c),y&&e.autoClear===!1&&e.clearDepth(),e.render(i,f),e.setRenderTarget(r,1,c),y&&e.autoClear===!1&&e.clearDepth(),e.render(i,d),e.setRenderTarget(r,2,c),y&&e.autoClear===!1&&e.clearDepth(),e.render(i,h),e.setRenderTarget(r,3,c),y&&e.autoClear===!1&&e.clearDepth(),e.render(i,m),e.setRenderTarget(r,4,c),y&&e.autoClear===!1&&e.clearDepth(),e.render(i,p),r.texture.generateMipmaps=N,e.setRenderTarget(r,5,c),y&&e.autoClear===!1&&e.clearDepth(),e.render(i,x),e.setRenderTarget(v,_,M),e.xr.enabled=R,r.texture.needsPMREMUpdate=!0}}class kb extends ei{constructor(e=[]){super(),this.isArrayCamera=!0,this.isMultiViewCamera=!1,this.cameras=e}}const ym=class ym{constructor(e,i,r,c){this.elements=[1,0,0,1],e!==void 0&&this.set(e,i,r,c)}identity(){return this.set(1,0,0,1),this}fromArray(e,i=0){for(let r=0;r<4;r++)this.elements[r]=e[r+i];return this}set(e,i,r,c){const f=this.elements;return f[0]=e,f[2]=i,f[1]=r,f[3]=c,this}};ym.prototype.isMatrix2=!0;let hx=ym;function px(o,e,i,r){const c=Xb(r);switch(i){case nS:return o*e;case am:return o*e/c.components*c.byteLength;case sm:return o*e/c.components*c.byteLength;case er:return o*e*2/c.components*c.byteLength;case rm:return o*e*2/c.components*c.byteLength;case iS:return o*e*3/c.components*c.byteLength;case Gi:return o*e*4/c.components*c.byteLength;case om:return o*e*4/c.components*c.byteLength;case Du:case Uu:return Math.floor((o+3)/4)*Math.floor((e+3)/4)*8;case Lu:case Ou:return Math.floor((o+3)/4)*Math.floor((e+3)/4)*16;case vp:case xp:return Math.max(o,16)*Math.max(e,8)/4;case gp:case _p:return Math.max(o,8)*Math.max(e,8)/2;case Sp:case yp:case Ep:case bp:return Math.floor((o+3)/4)*Math.floor((e+3)/4)*8;case Mp:case Iu:case Tp:return Math.floor((o+3)/4)*Math.floor((e+3)/4)*16;case Ap:return Math.floor((o+3)/4)*Math.floor((e+3)/4)*16;case Rp:return Math.floor((o+4)/5)*Math.floor((e+3)/4)*16;case wp:return Math.floor((o+4)/5)*Math.floor((e+4)/5)*16;case Cp:return Math.floor((o+5)/6)*Math.floor((e+4)/5)*16;case Np:return Math.floor((o+5)/6)*Math.floor((e+5)/6)*16;case Dp:return Math.floor((o+7)/8)*Math.floor((e+4)/5)*16;case Up:return Math.floor((o+7)/8)*Math.floor((e+5)/6)*16;case Lp:return Math.floor((o+7)/8)*Math.floor((e+7)/8)*16;case Op:return Math.floor((o+9)/10)*Math.floor((e+4)/5)*16;case Pp:return Math.floor((o+9)/10)*Math.floor((e+5)/6)*16;case Ip:return Math.floor((o+9)/10)*Math.floor((e+7)/8)*16;case Bp:return Math.floor((o+9)/10)*Math.floor((e+9)/10)*16;case zp:return Math.floor((o+11)/12)*Math.floor((e+9)/10)*16;case Fp:return Math.floor((o+11)/12)*Math.floor((e+11)/12)*16;case Hp:case Gp:case Vp:return Math.ceil(o/4)*Math.ceil(e/4)*16;case kp:case Xp:return Math.ceil(o/4)*Math.ceil(e/4)*8;case Bu:case Wp:return Math.ceil(o/4)*Math.ceil(e/4)*16}throw new Error(`Unable to determine texture byte length for ${i} format.`)}function Xb(o){switch(o){case pi:case Jx:return{byteLength:1,components:1};case Ll:case $x:case da:return{byteLength:2,components:1};case nm:case im:return{byteLength:2,components:4};case fa:case em:case Hi:return{byteLength:4,components:1};case tS:case eS:return{byteLength:4,components:3}}throw new Error(`THREE.TextureUtils: Unknown texture type ${o}.`)}typeof __THREE_DEVTOOLS__<"u"&&__THREE_DEVTOOLS__.dispatchEvent(new CustomEvent("register",{detail:{revision:Jp}}));typeof window<"u"&&(window.__THREE__?oe("WARNING: Multiple instances of Three.js being imported."):window.__THREE__=Jp);/**
 * @license
 * Copyright 2010-2026 Three.js Authors
 * SPDX-License-Identifier: MIT
 */function gS(){let o=null,e=!1,i=null,r=null;function c(f,d){r=o.requestAnimationFrame(c),i(f,d)}return{start:function(){e!==!0&&i!==null&&o!==null&&(r=o.requestAnimationFrame(c),e=!0)},stop:function(){o!==null&&o.cancelAnimationFrame(r),e=!1},setAnimationLoop:function(f){i=f},setContext:function(f){o=f}}}function Wb(o){const e=new WeakMap;function i(h,m){const p=h.array,x=h.usage,v=p.byteLength,_=o.createBuffer();o.bindBuffer(m,_),o.bufferData(m,p,x),h.onUploadCallback();let M;if(p instanceof Float32Array)M=o.FLOAT;else if(typeof Float16Array<"u"&&p instanceof Float16Array)M=o.HALF_FLOAT;else if(p instanceof Uint16Array)h.isFloat16BufferAttribute?M=o.HALF_FLOAT:M=o.UNSIGNED_SHORT;else if(p instanceof Int16Array)M=o.SHORT;else if(p instanceof Uint32Array)M=o.UNSIGNED_INT;else if(p instanceof Int32Array)M=o.INT;else if(p instanceof Int8Array)M=o.BYTE;else if(p instanceof Uint8Array)M=o.UNSIGNED_BYTE;else if(p instanceof Uint8ClampedArray)M=o.UNSIGNED_BYTE;else throw new Error("THREE.WebGLAttributes: Unsupported buffer data format: "+p);return{buffer:_,type:M,bytesPerElement:p.BYTES_PER_ELEMENT,version:h.version,size:v}}function r(h,m,p){const x=m.array,v=m.updateRanges;if(o.bindBuffer(p,h),v.length===0)o.bufferSubData(p,0,x);else{v.sort((M,R)=>M.start-R.start);let _=0;for(let M=1;M<v.length;M++){const R=v[_],N=v[M];N.start<=R.start+R.count+1?R.count=Math.max(R.count,N.start+N.count-R.start):(++_,v[_]=N)}v.length=_+1;for(let M=0,R=v.length;M<R;M++){const N=v[M];o.bufferSubData(p,N.start*x.BYTES_PER_ELEMENT,x,N.start,N.count)}m.clearUpdateRanges()}m.onUploadCallback()}function c(h){return h.isInterleavedBufferAttribute&&(h=h.data),e.get(h)}function f(h){h.isInterleavedBufferAttribute&&(h=h.data);const m=e.get(h);m&&(o.deleteBuffer(m.buffer),e.delete(h))}function d(h,m){if(h.isInterleavedBufferAttribute&&(h=h.data),h.isGLBufferAttribute){const x=e.get(h);(!x||x.version<h.version)&&e.set(h,{buffer:h.buffer,type:h.type,bytesPerElement:h.elementSize,version:h.version});return}const p=e.get(h);if(p===void 0)e.set(h,i(h,m));else if(p.version<h.version){if(p.size!==h.array.byteLength)throw new Error("THREE.WebGLAttributes: The size of the buffer attribute's array buffer does not match the original size. Resizing buffer attributes is not supported.");r(p.buffer,h,m),p.version=h.version}}return{get:c,remove:f,update:d}}var qb=`#ifdef USE_ALPHAHASH
	if ( diffuseColor.a < getAlphaHashThreshold( vPosition ) ) discard;
#endif`,Yb=`#ifdef USE_ALPHAHASH
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
#endif`,jb=`#ifdef USE_ALPHAMAP
	diffuseColor.a *= texture2D( alphaMap, vAlphaMapUv ).g;
#endif`,Zb=`#ifdef USE_ALPHAMAP
	uniform sampler2D alphaMap;
#endif`,Kb=`#ifdef USE_ALPHATEST
	#ifdef ALPHA_TO_COVERAGE
	diffuseColor.a = smoothstep( alphaTest, alphaTest + fwidth( diffuseColor.a ), diffuseColor.a );
	if ( diffuseColor.a == 0.0 ) discard;
	#else
	if ( diffuseColor.a < alphaTest ) discard;
	#endif
#endif`,Qb=`#ifdef USE_ALPHATEST
	uniform float alphaTest;
#endif`,Jb=`#ifdef USE_AOMAP
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
#endif`,$b=`#ifdef USE_AOMAP
	uniform sampler2D aoMap;
	uniform float aoMapIntensity;
#endif`,tT=`#ifdef USE_BATCHING
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
#endif`,eT=`#ifdef USE_BATCHING
	mat4 batchingMatrix = getBatchingMatrix( getIndirectIndex( gl_DrawID ) );
#endif`,nT=`vec3 transformed = vec3( position );
#ifdef USE_ALPHAHASH
	vPosition = vec3( position );
#endif`,iT=`vec3 objectNormal = vec3( normal );
#ifdef USE_TANGENT
	vec3 objectTangent = vec3( tangent.xyz );
#endif`,aT=`float G_BlinnPhong_Implicit( ) {
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
} // validated`,sT=`#ifdef USE_IRIDESCENCE
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
#endif`,rT=`#ifdef USE_BUMPMAP
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
#endif`,oT=`#if NUM_CLIPPING_PLANES > 0
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
#endif`,lT=`#if NUM_CLIPPING_PLANES > 0
	varying vec3 vClipPosition;
	uniform vec4 clippingPlanes[ NUM_CLIPPING_PLANES ];
#endif`,cT=`#if NUM_CLIPPING_PLANES > 0
	varying vec3 vClipPosition;
#endif`,uT=`#if NUM_CLIPPING_PLANES > 0
	vClipPosition = - mvPosition.xyz;
#endif`,fT=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA )
	diffuseColor *= vColor;
#endif`,dT=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA )
	varying vec4 vColor;
#endif`,hT=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA ) || defined( USE_INSTANCING_COLOR ) || defined( USE_BATCHING_COLOR )
	varying vec4 vColor;
#endif`,pT=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA ) || defined( USE_INSTANCING_COLOR ) || defined( USE_BATCHING_COLOR )
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
#endif`,mT=`#define PI 3.141592653589793
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
} // validated`,gT=`#ifdef ENVMAP_TYPE_CUBE_UV
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
#endif`,vT=`vec3 transformedNormal = objectNormal;
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
#endif`,_T=`#ifdef USE_DISPLACEMENTMAP
	uniform sampler2D displacementMap;
	uniform float displacementScale;
	uniform float displacementBias;
#endif`,xT=`#ifdef USE_DISPLACEMENTMAP
	transformed += normalize( objectNormal ) * ( texture2D( displacementMap, vDisplacementMapUv ).x * displacementScale + displacementBias );
#endif`,ST=`#ifdef USE_EMISSIVEMAP
	vec4 emissiveColor = texture2D( emissiveMap, vEmissiveMapUv );
	#ifdef DECODE_VIDEO_TEXTURE_EMISSIVE
		emissiveColor = sRGBTransferEOTF( emissiveColor );
	#endif
	totalEmissiveRadiance *= emissiveColor.rgb;
#endif`,yT=`#ifdef USE_EMISSIVEMAP
	uniform sampler2D emissiveMap;
#endif`,MT="gl_FragColor = linearToOutputTexel( gl_FragColor );",ET=`vec4 LinearTransferOETF( in vec4 value ) {
	return value;
}
vec4 sRGBTransferEOTF( in vec4 value ) {
	return vec4( mix( pow( value.rgb * 0.9478672986 + vec3( 0.0521327014 ), vec3( 2.4 ) ), value.rgb * 0.0773993808, vec3( lessThanEqual( value.rgb, vec3( 0.04045 ) ) ) ), value.a );
}
vec4 sRGBTransferOETF( in vec4 value ) {
	return vec4( mix( pow( value.rgb, vec3( 0.41666 ) ) * 1.055 - vec3( 0.055 ), value.rgb * 12.92, vec3( lessThanEqual( value.rgb, vec3( 0.0031308 ) ) ) ), value.a );
}`,bT=`#ifdef USE_ENVMAP
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
#endif`,TT=`#ifdef USE_ENVMAP
	uniform float envMapIntensity;
	uniform mat3 envMapRotation;
	#ifdef ENVMAP_TYPE_CUBE
		uniform samplerCube envMap;
	#else
		uniform sampler2D envMap;
	#endif
#endif`,AT=`#ifdef USE_ENVMAP
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
#endif`,RT=`#ifdef USE_ENVMAP
	#if defined( USE_BUMPMAP ) || defined( USE_NORMALMAP ) || defined( PHONG ) || defined( LAMBERT )
		#define ENV_WORLDPOS
	#endif
	#ifdef ENV_WORLDPOS
		
		varying vec3 vWorldPosition;
	#else
		varying vec3 vReflect;
		uniform float refractionRatio;
	#endif
#endif`,wT=`#ifdef USE_ENVMAP
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
#endif`,CT=`#ifdef USE_FOG
	vFogDepth = - mvPosition.z;
#endif`,NT=`#ifdef USE_FOG
	varying float vFogDepth;
#endif`,DT=`#ifdef USE_FOG
	#ifdef FOG_EXP2
		float fogFactor = 1.0 - exp( - fogDensity * fogDensity * vFogDepth * vFogDepth );
	#else
		float fogFactor = smoothstep( fogNear, fogFar, vFogDepth );
	#endif
	gl_FragColor.rgb = mix( gl_FragColor.rgb, fogColor, fogFactor );
#endif`,UT=`#ifdef USE_FOG
	uniform vec3 fogColor;
	varying float vFogDepth;
	#ifdef FOG_EXP2
		uniform float fogDensity;
	#else
		uniform float fogNear;
		uniform float fogFar;
	#endif
#endif`,LT=`#ifdef USE_GRADIENTMAP
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
}`,OT=`#ifdef USE_LIGHTMAP
	uniform sampler2D lightMap;
	uniform float lightMapIntensity;
#endif`,PT=`LambertMaterial material;
material.diffuseColor = diffuseColor.rgb;
material.specularStrength = specularStrength;`,IT=`varying vec3 vViewPosition;
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
#define RE_IndirectDiffuse		RE_IndirectDiffuse_Lambert`,BT=`uniform bool receiveShadow;
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
#include <lightprobes_pars_fragment>`,zT=`#ifdef USE_ENVMAP
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
#endif`,FT=`ToonMaterial material;
material.diffuseColor = diffuseColor.rgb;`,HT=`varying vec3 vViewPosition;
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
#define RE_IndirectDiffuse		RE_IndirectDiffuse_Toon`,GT=`BlinnPhongMaterial material;
material.diffuseColor = diffuseColor.rgb;
material.specularColor = specular;
material.specularShininess = shininess;
material.specularStrength = specularStrength;`,VT=`varying vec3 vViewPosition;
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
#define RE_IndirectDiffuse		RE_IndirectDiffuse_BlinnPhong`,kT=`PhysicalMaterial material;
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
#endif`,XT=`uniform sampler2D dfgLUT;
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
}`,WT=`
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
#endif`,qT=`#if defined( RE_IndirectDiffuse )
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
#endif`,YT=`#if defined( RE_IndirectDiffuse )
	#if defined( LAMBERT ) || defined( PHONG )
		irradiance += iblIrradiance;
	#endif
	RE_IndirectDiffuse( irradiance, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
#endif
#if defined( RE_IndirectSpecular )
	RE_IndirectSpecular( radiance, iblIrradiance, clearcoatRadiance, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
#endif`,jT=`#ifdef USE_LIGHT_PROBES_GRID
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
#endif`,ZT=`#if defined( USE_LOGARITHMIC_DEPTH_BUFFER )
	gl_FragDepth = vIsPerspective == 0.0 ? gl_FragCoord.z : log2( vFragDepth ) * logDepthBufFC * 0.5;
#endif`,KT=`#if defined( USE_LOGARITHMIC_DEPTH_BUFFER )
	uniform float logDepthBufFC;
	varying float vFragDepth;
	varying float vIsPerspective;
#endif`,QT=`#ifdef USE_LOGARITHMIC_DEPTH_BUFFER
	varying float vFragDepth;
	varying float vIsPerspective;
#endif`,JT=`#ifdef USE_LOGARITHMIC_DEPTH_BUFFER
	vFragDepth = 1.0 + gl_Position.w;
	vIsPerspective = float( isPerspectiveMatrix( projectionMatrix ) );
#endif`,$T=`#ifdef USE_MAP
	vec4 sampledDiffuseColor = texture2D( map, vMapUv );
	#ifdef DECODE_VIDEO_TEXTURE
		sampledDiffuseColor = sRGBTransferEOTF( sampledDiffuseColor );
	#endif
	diffuseColor *= sampledDiffuseColor;
#endif`,t1=`#ifdef USE_MAP
	uniform sampler2D map;
#endif`,e1=`#if defined( USE_MAP ) || defined( USE_ALPHAMAP )
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
#endif`,n1=`#if defined( USE_POINTS_UV )
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
#endif`,i1=`float metalnessFactor = metalness;
#ifdef USE_METALNESSMAP
	vec4 texelMetalness = texture2D( metalnessMap, vMetalnessMapUv );
	metalnessFactor *= texelMetalness.b;
#endif`,a1=`#ifdef USE_METALNESSMAP
	uniform sampler2D metalnessMap;
#endif`,s1=`#ifdef USE_INSTANCING_MORPH
	float morphTargetInfluences[ MORPHTARGETS_COUNT ];
	float morphTargetBaseInfluence = texelFetch( morphTexture, ivec2( 0, gl_InstanceID ), 0 ).r;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		morphTargetInfluences[i] =  texelFetch( morphTexture, ivec2( i + 1, gl_InstanceID ), 0 ).r;
	}
#endif`,r1=`#if defined( USE_MORPHCOLORS )
	vColor *= morphTargetBaseInfluence;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		#if defined( USE_COLOR_ALPHA )
			if ( morphTargetInfluences[ i ] != 0.0 ) vColor += getMorph( gl_VertexID, i, 2 ) * morphTargetInfluences[ i ];
		#elif defined( USE_COLOR )
			if ( morphTargetInfluences[ i ] != 0.0 ) vColor += getMorph( gl_VertexID, i, 2 ).rgb * morphTargetInfluences[ i ];
		#endif
	}
#endif`,o1=`#ifdef USE_MORPHNORMALS
	objectNormal *= morphTargetBaseInfluence;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		if ( morphTargetInfluences[ i ] != 0.0 ) objectNormal += getMorph( gl_VertexID, i, 1 ).xyz * morphTargetInfluences[ i ];
	}
#endif`,l1=`#ifdef USE_MORPHTARGETS
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
#endif`,c1=`#ifdef USE_MORPHTARGETS
	transformed *= morphTargetBaseInfluence;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		if ( morphTargetInfluences[ i ] != 0.0 ) transformed += getMorph( gl_VertexID, i, 0 ).xyz * morphTargetInfluences[ i ];
	}
#endif`,u1=`float faceDirection = gl_FrontFacing ? 1.0 : - 1.0;
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
vec3 nonPerturbedNormal = normal;`,f1=`#ifdef USE_NORMALMAP_OBJECTSPACE
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
#endif`,d1=`#ifndef FLAT_SHADED
	varying vec3 vNormal;
	#ifdef USE_TANGENT
		varying vec3 vTangent;
		varying vec3 vBitangent;
	#endif
#endif`,h1=`#ifndef FLAT_SHADED
	varying vec3 vNormal;
	#ifdef USE_TANGENT
		varying vec3 vTangent;
		varying vec3 vBitangent;
	#endif
#endif`,p1=`#ifndef FLAT_SHADED
	vNormal = normalize( transformedNormal );
	#ifdef USE_TANGENT
		vTangent = normalize( transformedTangent );
		vBitangent = normalize( cross( vNormal, vTangent ) * tangent.w );
		#ifdef FLIP_SIDED
			vBitangent = - vBitangent;
		#endif
	#endif
#endif`,m1=`#ifdef USE_NORMALMAP
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
#endif`,g1=`#ifdef USE_CLEARCOAT
	vec3 clearcoatNormal = nonPerturbedNormal;
#endif`,v1=`#ifdef USE_CLEARCOAT_NORMALMAP
	vec3 clearcoatMapN = texture2D( clearcoatNormalMap, vClearcoatNormalMapUv ).xyz * 2.0 - 1.0;
	clearcoatMapN.xy *= clearcoatNormalScale;
	clearcoatNormal = normalize( tbn2 * clearcoatMapN );
#endif`,_1=`#ifdef USE_CLEARCOATMAP
	uniform sampler2D clearcoatMap;
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	uniform sampler2D clearcoatNormalMap;
	uniform vec2 clearcoatNormalScale;
#endif
#ifdef USE_CLEARCOAT_ROUGHNESSMAP
	uniform sampler2D clearcoatRoughnessMap;
#endif`,x1=`#ifdef USE_IRIDESCENCEMAP
	uniform sampler2D iridescenceMap;
#endif
#ifdef USE_IRIDESCENCE_THICKNESSMAP
	uniform sampler2D iridescenceThicknessMap;
#endif`,S1=`#ifdef OPAQUE
diffuseColor.a = 1.0;
#endif
#ifdef USE_TRANSMISSION
diffuseColor.a *= material.transmissionAlpha;
#endif
gl_FragColor = vec4( outgoingLight, diffuseColor.a );`,y1=`vec3 packNormalToRGB( const in vec3 normal ) {
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
}`,M1=`#ifdef PREMULTIPLIED_ALPHA
	gl_FragColor.rgb *= gl_FragColor.a;
#endif`,E1=`vec4 mvPosition = vec4( transformed, 1.0 );
#ifdef USE_BATCHING
	mvPosition = batchingMatrix * mvPosition;
#endif
#ifdef USE_INSTANCING
	mvPosition = instanceMatrix * mvPosition;
#endif
mvPosition = modelViewMatrix * mvPosition;
gl_Position = projectionMatrix * mvPosition;`,b1=`#ifdef DITHERING
	gl_FragColor.rgb = dithering( gl_FragColor.rgb );
#endif`,T1=`#ifdef DITHERING
	vec3 dithering( vec3 color ) {
		float grid_position = rand( gl_FragCoord.xy );
		vec3 dither_shift_RGB = vec3( 0.25 / 255.0, -0.25 / 255.0, 0.25 / 255.0 );
		dither_shift_RGB = mix( 2.0 * dither_shift_RGB, -2.0 * dither_shift_RGB, grid_position );
		return color + dither_shift_RGB;
	}
#endif`,A1=`float roughnessFactor = roughness;
#ifdef USE_ROUGHNESSMAP
	vec4 texelRoughness = texture2D( roughnessMap, vRoughnessMapUv );
	roughnessFactor *= texelRoughness.g;
#endif`,R1=`#ifdef USE_ROUGHNESSMAP
	uniform sampler2D roughnessMap;
#endif`,w1=`#if NUM_SPOT_LIGHT_COORDS > 0
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
#endif`,C1=`#if NUM_SPOT_LIGHT_COORDS > 0
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
#endif`,N1=`#if ( defined( USE_SHADOWMAP ) && ( NUM_DIR_LIGHT_SHADOWS > 0 || NUM_SUN_LIGHT_SHADOWS > 0 || NUM_POINT_LIGHT_SHADOWS > 0 ) ) || ( NUM_SPOT_LIGHT_COORDS > 0 )
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
#endif`,D1=`float getShadowMask() {
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
}`,U1=`#ifdef USE_SKINNING
	mat4 boneMatX = getBoneMatrix( skinIndex.x );
	mat4 boneMatY = getBoneMatrix( skinIndex.y );
	mat4 boneMatZ = getBoneMatrix( skinIndex.z );
	mat4 boneMatW = getBoneMatrix( skinIndex.w );
#endif`,L1=`#ifdef USE_SKINNING
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
#endif`,O1=`#ifdef USE_SKINNING
	vec4 skinVertex = bindMatrix * vec4( transformed, 1.0 );
	vec4 skinned = vec4( 0.0 );
	skinned += boneMatX * skinVertex * skinWeight.x;
	skinned += boneMatY * skinVertex * skinWeight.y;
	skinned += boneMatZ * skinVertex * skinWeight.z;
	skinned += boneMatW * skinVertex * skinWeight.w;
	transformed = ( bindMatrixInverse * skinned ).xyz;
#endif`,P1=`#ifdef USE_SKINNING
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
#endif`,I1=`float specularStrength;
#ifdef USE_SPECULARMAP
	vec4 texelSpecular = texture2D( specularMap, vSpecularMapUv );
	specularStrength = texelSpecular.r;
#else
	specularStrength = 1.0;
#endif`,B1=`#ifdef USE_SPECULARMAP
	uniform sampler2D specularMap;
#endif`,z1=`#if defined( TONE_MAPPING )
	gl_FragColor.rgb = toneMapping( gl_FragColor.rgb );
#endif`,F1=`#ifndef saturate
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
vec3 CustomToneMapping( vec3 color ) { return color; }`,H1=`#ifdef USE_TRANSMISSION
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
#endif`,G1=`#ifdef USE_TRANSMISSION
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
#endif`,V1=`#if defined( USE_UV ) || defined( USE_ANISOTROPY )
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
#endif`,k1=`#if defined( USE_UV ) || defined( USE_ANISOTROPY )
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
#endif`,X1=`#if defined( USE_UV ) || defined( USE_ANISOTROPY )
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
#endif`,W1=`#if defined( USE_ENVMAP ) || defined( DISTANCE ) || defined ( USE_SHADOWMAP ) || defined ( USE_TRANSMISSION ) || NUM_SPOT_LIGHT_COORDS > 0
	vec4 worldPosition = vec4( transformed, 1.0 );
	#ifdef USE_BATCHING
		worldPosition = batchingMatrix * worldPosition;
	#endif
	#ifdef USE_INSTANCING
		worldPosition = instanceMatrix * worldPosition;
	#endif
	worldPosition = modelMatrix * worldPosition;
#endif`;const q1=`varying vec2 vUv;
uniform mat3 uvTransform;
void main() {
	vUv = ( uvTransform * vec3( uv, 1 ) ).xy;
	gl_Position = vec4( position.xy, 1.0, 1.0 );
}`,Y1=`uniform sampler2D t2D;
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
}`,j1=`varying vec3 vWorldDirection;
#include <common>
void main() {
	vWorldDirection = transformDirection( position, modelMatrix );
	#include <begin_vertex>
	#include <project_vertex>
	gl_Position.z = gl_Position.w;
}`,Z1=`#ifdef ENVMAP_TYPE_CUBE
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
}`,K1=`varying vec3 vWorldDirection;
#include <common>
void main() {
	vWorldDirection = transformDirection( position, modelMatrix );
	#include <begin_vertex>
	#include <project_vertex>
	gl_Position.z = gl_Position.w;
}`,Q1=`uniform samplerCube tCube;
uniform float tFlip;
uniform float opacity;
varying vec3 vWorldDirection;
void main() {
	vec4 texColor = textureCube( tCube, vec3( tFlip * vWorldDirection.x, vWorldDirection.yz ) );
	gl_FragColor = texColor;
	gl_FragColor.a *= opacity;
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,J1=`#include <common>
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
}`,$1=`#if DEPTH_PACKING == 3200
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
}`,tA=`#define DISTANCE
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
}`,eA=`#define DISTANCE
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
}`,nA=`varying vec3 vWorldDirection;
#include <common>
void main() {
	vWorldDirection = transformDirection( position, modelMatrix );
	#include <begin_vertex>
	#include <project_vertex>
}`,iA=`uniform sampler2D tEquirect;
varying vec3 vWorldDirection;
#include <common>
void main() {
	vec3 direction = normalize( vWorldDirection );
	vec2 sampleUV = equirectUv( direction );
	gl_FragColor = texture2D( tEquirect, sampleUV );
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,aA=`uniform float scale;
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
}`,sA=`uniform vec3 diffuse;
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
}`,rA=`#include <common>
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
}`,oA=`uniform vec3 diffuse;
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
}`,lA=`#define LAMBERT
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
}`,cA=`#define LAMBERT
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
}`,uA=`#define MATCAP
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
}`,fA=`#define MATCAP
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
}`,dA=`#define NORMAL
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
}`,hA=`#define NORMAL
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
}`,pA=`#define PHONG
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
}`,mA=`#define PHONG
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
}`,gA=`#define STANDARD
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
}`,vA=`#define STANDARD
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
}`,_A=`#define TOON
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
}`,xA=`#define TOON
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
}`,SA=`uniform float size;
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
}`,yA=`uniform vec3 diffuse;
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
}`,MA=`#include <common>
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
}`,EA=`uniform vec3 color;
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
}`,bA=`uniform float rotation;
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
}`,TA=`uniform vec3 diffuse;
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
}`,_e={alphahash_fragment:qb,alphahash_pars_fragment:Yb,alphamap_fragment:jb,alphamap_pars_fragment:Zb,alphatest_fragment:Kb,alphatest_pars_fragment:Qb,aomap_fragment:Jb,aomap_pars_fragment:$b,batching_pars_vertex:tT,batching_vertex:eT,begin_vertex:nT,beginnormal_vertex:iT,bsdfs:aT,iridescence_fragment:sT,bumpmap_pars_fragment:rT,clipping_planes_fragment:oT,clipping_planes_pars_fragment:lT,clipping_planes_pars_vertex:cT,clipping_planes_vertex:uT,color_fragment:fT,color_pars_fragment:dT,color_pars_vertex:hT,color_vertex:pT,common:mT,cube_uv_reflection_fragment:gT,defaultnormal_vertex:vT,displacementmap_pars_vertex:_T,displacementmap_vertex:xT,emissivemap_fragment:ST,emissivemap_pars_fragment:yT,colorspace_fragment:MT,colorspace_pars_fragment:ET,envmap_fragment:bT,envmap_common_pars_fragment:TT,envmap_pars_fragment:AT,envmap_pars_vertex:RT,envmap_physical_pars_fragment:zT,envmap_vertex:wT,fog_vertex:CT,fog_pars_vertex:NT,fog_fragment:DT,fog_pars_fragment:UT,gradientmap_pars_fragment:LT,lightmap_pars_fragment:OT,lights_lambert_fragment:PT,lights_lambert_pars_fragment:IT,lights_pars_begin:BT,lights_toon_fragment:FT,lights_toon_pars_fragment:HT,lights_phong_fragment:GT,lights_phong_pars_fragment:VT,lights_physical_fragment:kT,lights_physical_pars_fragment:XT,lights_fragment_begin:WT,lights_fragment_maps:qT,lights_fragment_end:YT,lightprobes_pars_fragment:jT,logdepthbuf_fragment:ZT,logdepthbuf_pars_fragment:KT,logdepthbuf_pars_vertex:QT,logdepthbuf_vertex:JT,map_fragment:$T,map_pars_fragment:t1,map_particle_fragment:e1,map_particle_pars_fragment:n1,metalnessmap_fragment:i1,metalnessmap_pars_fragment:a1,morphinstance_vertex:s1,morphcolor_vertex:r1,morphnormal_vertex:o1,morphtarget_pars_vertex:l1,morphtarget_vertex:c1,normal_fragment_begin:u1,normal_fragment_maps:f1,normal_pars_fragment:d1,normal_pars_vertex:h1,normal_vertex:p1,normalmap_pars_fragment:m1,clearcoat_normal_fragment_begin:g1,clearcoat_normal_fragment_maps:v1,clearcoat_pars_fragment:_1,iridescence_pars_fragment:x1,opaque_fragment:S1,packing:y1,premultiplied_alpha_fragment:M1,project_vertex:E1,dithering_fragment:b1,dithering_pars_fragment:T1,roughnessmap_fragment:A1,roughnessmap_pars_fragment:R1,shadowmap_pars_fragment:w1,shadowmap_pars_vertex:C1,shadowmap_vertex:N1,shadowmask_pars_fragment:D1,skinbase_vertex:U1,skinning_pars_vertex:L1,skinning_vertex:O1,skinnormal_vertex:P1,specularmap_fragment:I1,specularmap_pars_fragment:B1,tonemapping_fragment:z1,tonemapping_pars_fragment:F1,transmission_fragment:H1,transmission_pars_fragment:G1,uv_pars_fragment:V1,uv_pars_vertex:k1,uv_vertex:X1,worldpos_vertex:W1,background_vert:q1,background_frag:Y1,backgroundCube_vert:j1,backgroundCube_frag:Z1,cube_vert:K1,cube_frag:Q1,depth_vert:J1,depth_frag:$1,distance_vert:tA,distance_frag:eA,equirect_vert:nA,equirect_frag:iA,linedashed_vert:aA,linedashed_frag:sA,meshbasic_vert:rA,meshbasic_frag:oA,meshlambert_vert:lA,meshlambert_frag:cA,meshmatcap_vert:uA,meshmatcap_frag:fA,meshnormal_vert:dA,meshnormal_frag:hA,meshphong_vert:pA,meshphong_frag:mA,meshphysical_vert:gA,meshphysical_frag:vA,meshtoon_vert:_A,meshtoon_frag:xA,points_vert:SA,points_frag:yA,shadow_vert:MA,shadow_frag:EA,sprite_vert:bA,sprite_frag:TA},Ht={common:{diffuse:{value:new he(16777215)},opacity:{value:1},map:{value:null},mapTransform:{value:new de},alphaMap:{value:null},alphaMapTransform:{value:new de},alphaTest:{value:0}},specularmap:{specularMap:{value:null},specularMapTransform:{value:new de}},envmap:{envMap:{value:null},envMapRotation:{value:new de},reflectivity:{value:1},ior:{value:1.5},refractionRatio:{value:.98},dfgLUT:{value:null}},aomap:{aoMap:{value:null},aoMapIntensity:{value:1},aoMapTransform:{value:new de}},lightmap:{lightMap:{value:null},lightMapIntensity:{value:1},lightMapTransform:{value:new de}},bumpmap:{bumpMap:{value:null},bumpMapTransform:{value:new de},bumpScale:{value:1}},normalmap:{normalMap:{value:null},normalMapTransform:{value:new de},normalScale:{value:new Re(1,1)}},displacementmap:{displacementMap:{value:null},displacementMapTransform:{value:new de},displacementScale:{value:1},displacementBias:{value:0}},emissivemap:{emissiveMap:{value:null},emissiveMapTransform:{value:new de}},metalnessmap:{metalnessMap:{value:null},metalnessMapTransform:{value:new de}},roughnessmap:{roughnessMap:{value:null},roughnessMapTransform:{value:new de}},gradientmap:{gradientMap:{value:null}},fog:{fogDensity:{value:25e-5},fogNear:{value:1},fogFar:{value:2e3},fogColor:{value:new he(16777215)}},lights:{ambientLightColor:{value:[]},lightProbe:{value:[]},sunLights:{value:[],properties:{direction:{},color:{}}},sunLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{}}},sunShadowMatrix:{value:[]},sunShadowCascade:{value:[]},directionalLights:{value:[],properties:{direction:{},color:{}}},directionalLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{}}},directionalShadowMatrix:{value:[]},spotLights:{value:[],properties:{color:{},position:{},direction:{},distance:{},coneCos:{},penumbraCos:{},decay:{}}},spotLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{}}},spotLightMap:{value:[]},spotLightMatrix:{value:[]},pointLights:{value:[],properties:{color:{},position:{},decay:{},distance:{}}},pointLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{},shadowCameraNear:{},shadowCameraFar:{}}},pointShadowMatrix:{value:[]},hemisphereLights:{value:[],properties:{direction:{},skyColor:{},groundColor:{}}},rectAreaLights:{value:[],properties:{color:{},position:{},width:{},height:{}}},ltc_1:{value:null},ltc_2:{value:null},probesSH:{value:null},probesMin:{value:new dt},probesMax:{value:new dt},probesResolution:{value:new dt}},points:{diffuse:{value:new he(16777215)},opacity:{value:1},size:{value:1},scale:{value:1},map:{value:null},alphaMap:{value:null},alphaMapTransform:{value:new de},alphaTest:{value:0},uvTransform:{value:new de}},sprite:{diffuse:{value:new he(16777215)},opacity:{value:1},center:{value:new Re(.5,.5)},rotation:{value:0},map:{value:null},mapTransform:{value:new de},alphaMap:{value:null},alphaMapTransform:{value:new de},alphaTest:{value:0}}},oa={basic:{uniforms:Wn([Ht.common,Ht.specularmap,Ht.envmap,Ht.aomap,Ht.lightmap,Ht.fog]),vertexShader:_e.meshbasic_vert,fragmentShader:_e.meshbasic_frag},lambert:{uniforms:Wn([Ht.common,Ht.specularmap,Ht.envmap,Ht.aomap,Ht.lightmap,Ht.emissivemap,Ht.bumpmap,Ht.normalmap,Ht.displacementmap,Ht.fog,Ht.lights,{emissive:{value:new he(0)},envMapIntensity:{value:1}}]),vertexShader:_e.meshlambert_vert,fragmentShader:_e.meshlambert_frag},phong:{uniforms:Wn([Ht.common,Ht.specularmap,Ht.envmap,Ht.aomap,Ht.lightmap,Ht.emissivemap,Ht.bumpmap,Ht.normalmap,Ht.displacementmap,Ht.fog,Ht.lights,{emissive:{value:new he(0)},specular:{value:new he(1118481)},shininess:{value:30},envMapIntensity:{value:1}}]),vertexShader:_e.meshphong_vert,fragmentShader:_e.meshphong_frag},standard:{uniforms:Wn([Ht.common,Ht.envmap,Ht.aomap,Ht.lightmap,Ht.emissivemap,Ht.bumpmap,Ht.normalmap,Ht.displacementmap,Ht.roughnessmap,Ht.metalnessmap,Ht.fog,Ht.lights,{emissive:{value:new he(0)},roughness:{value:1},metalness:{value:0},envMapIntensity:{value:1}}]),vertexShader:_e.meshphysical_vert,fragmentShader:_e.meshphysical_frag},toon:{uniforms:Wn([Ht.common,Ht.aomap,Ht.lightmap,Ht.emissivemap,Ht.bumpmap,Ht.normalmap,Ht.displacementmap,Ht.gradientmap,Ht.fog,Ht.lights,{emissive:{value:new he(0)}}]),vertexShader:_e.meshtoon_vert,fragmentShader:_e.meshtoon_frag},matcap:{uniforms:Wn([Ht.common,Ht.bumpmap,Ht.normalmap,Ht.displacementmap,Ht.fog,{matcap:{value:null}}]),vertexShader:_e.meshmatcap_vert,fragmentShader:_e.meshmatcap_frag},points:{uniforms:Wn([Ht.points,Ht.fog]),vertexShader:_e.points_vert,fragmentShader:_e.points_frag},dashed:{uniforms:Wn([Ht.common,Ht.fog,{scale:{value:1},dashSize:{value:1},totalSize:{value:2}}]),vertexShader:_e.linedashed_vert,fragmentShader:_e.linedashed_frag},depth:{uniforms:Wn([Ht.common,Ht.displacementmap]),vertexShader:_e.depth_vert,fragmentShader:_e.depth_frag},normal:{uniforms:Wn([Ht.common,Ht.bumpmap,Ht.normalmap,Ht.displacementmap,{opacity:{value:1}}]),vertexShader:_e.meshnormal_vert,fragmentShader:_e.meshnormal_frag},sprite:{uniforms:Wn([Ht.sprite,Ht.fog]),vertexShader:_e.sprite_vert,fragmentShader:_e.sprite_frag},background:{uniforms:{uvTransform:{value:new de},t2D:{value:null},backgroundIntensity:{value:1}},vertexShader:_e.background_vert,fragmentShader:_e.background_frag},backgroundCube:{uniforms:{envMap:{value:null},backgroundBlurriness:{value:0},backgroundIntensity:{value:1},backgroundRotation:{value:new de}},vertexShader:_e.backgroundCube_vert,fragmentShader:_e.backgroundCube_frag},cube:{uniforms:{tCube:{value:null},tFlip:{value:-1},opacity:{value:1}},vertexShader:_e.cube_vert,fragmentShader:_e.cube_frag},equirect:{uniforms:{tEquirect:{value:null}},vertexShader:_e.equirect_vert,fragmentShader:_e.equirect_frag},distance:{uniforms:Wn([Ht.common,Ht.displacementmap,{referencePosition:{value:new dt},nearDistance:{value:1},farDistance:{value:1e3}}]),vertexShader:_e.distance_vert,fragmentShader:_e.distance_frag},shadow:{uniforms:Wn([Ht.lights,Ht.fog,{color:{value:new he(0)},opacity:{value:1}}]),vertexShader:_e.shadow_vert,fragmentShader:_e.shadow_frag}};oa.physical={uniforms:Wn([oa.standard.uniforms,{clearcoat:{value:0},clearcoatMap:{value:null},clearcoatMapTransform:{value:new de},clearcoatNormalMap:{value:null},clearcoatNormalMapTransform:{value:new de},clearcoatNormalScale:{value:new Re(1,1)},clearcoatRoughness:{value:0},clearcoatRoughnessMap:{value:null},clearcoatRoughnessMapTransform:{value:new de},dispersion:{value:0},retroreflectivity:{value:0},iridescence:{value:0},iridescenceMap:{value:null},iridescenceMapTransform:{value:new de},iridescenceIOR:{value:1.3},iridescenceThicknessMinimum:{value:100},iridescenceThicknessMaximum:{value:400},iridescenceThicknessMap:{value:null},iridescenceThicknessMapTransform:{value:new de},sheen:{value:0},sheenColor:{value:new he(0)},sheenColorMap:{value:null},sheenColorMapTransform:{value:new de},sheenRoughness:{value:1},sheenRoughnessMap:{value:null},sheenRoughnessMapTransform:{value:new de},transmission:{value:0},transmissionMap:{value:null},transmissionMapTransform:{value:new de},transmissionSamplerSize:{value:new Re},transmissionSamplerMap:{value:null},thickness:{value:0},thicknessMap:{value:null},thicknessMapTransform:{value:new de},attenuationDistance:{value:0},attenuationColor:{value:new he(0)},specularColor:{value:new he(1,1,1)},specularColorMap:{value:null},specularColorMapTransform:{value:new de},specularIntensity:{value:1},specularIntensityMap:{value:null},specularIntensityMapTransform:{value:new de},anisotropyVector:{value:new Re},anisotropyMap:{value:null},anisotropyMapTransform:{value:new de}}]),vertexShader:_e.meshphysical_vert,fragmentShader:_e.meshphysical_frag};const wu={r:0,b:0,g:0},AA=new nn,vS=new de;vS.set(-1,0,0,0,1,0,0,0,1);function RA(o,e,i,r,c,f){const d=new he(0);let h=c===!0?0:1,m,p,x=null,v=0,_=null;function M(O){let k=O.isScene===!0?O.background:null;if(k&&k.isTexture){const C=O.backgroundBlurriness>0;k=e.get(k,C)}return k}function R(O){let k=!1;const C=M(O);C===null?y(d,h):C&&C.isColor&&(y(C,1),k=!0);const D=o.xr.getEnvironmentBlendMode();D==="additive"?i.buffers.color.setClear(0,0,0,1,f):D==="alpha-blend"&&i.buffers.color.setClear(0,0,0,0,f),(o.autoClear||k)&&(i.buffers.depth.setTest(!0),i.buffers.depth.setMask(!0),i.buffers.color.setMask(!0),o.clear(o.autoClearColor,o.autoClearDepth,o.autoClearStencil))}function N(O,k){const C=M(k);C&&(C.isCubeTexture||C.mapping===ku)?(p===void 0&&(p=new _n(new Ms(1,1,1),new ha({name:"BackgroundCubeMaterial",uniforms:vo(oa.backgroundCube.uniforms),vertexShader:oa.backgroundCube.vertexShader,fragmentShader:oa.backgroundCube.fragmentShader,side:qn,depthTest:!1,depthWrite:!1,fog:!1,allowOverride:!1})),p.geometry.deleteAttribute("normal"),p.geometry.deleteAttribute("uv"),p.onBeforeRender=function(D,U,B){this.matrixWorld.copyPosition(B.matrixWorld)},Object.defineProperty(p.material,"envMap",{get:function(){return this.uniforms.envMap.value}}),r.update(p)),p.material.uniforms.envMap.value=C,p.material.uniforms.backgroundBlurriness.value=k.backgroundBlurriness,p.material.uniforms.backgroundIntensity.value=k.backgroundIntensity,p.material.uniforms.backgroundRotation.value.setFromMatrix4(AA.makeRotationFromEuler(k.backgroundRotation)).transpose(),C.isCubeTexture&&C.isRenderTargetTexture===!1&&p.material.uniforms.backgroundRotation.value.premultiply(vS),p.material.toneMapped=Pe.getTransfer(C.colorSpace)!==Ye,(x!==C||v!==C.version||_!==o.toneMapping)&&(p.material.needsUpdate=!0,x=C,v=C.version,_=o.toneMapping),p.layers.enableAll(),O.unshift(p,p.geometry,p.material,0,0,null)):C&&C.isTexture&&(m===void 0&&(m=new _n(new go(2,2),new ha({name:"BackgroundMaterial",uniforms:vo(oa.background.uniforms),vertexShader:oa.background.vertexShader,fragmentShader:oa.background.fragmentShader,side:$s,depthTest:!1,depthWrite:!1,fog:!1,allowOverride:!1})),m.geometry.deleteAttribute("normal"),Object.defineProperty(m.material,"map",{get:function(){return this.uniforms.t2D.value}}),r.update(m)),m.material.uniforms.t2D.value=C,m.material.uniforms.backgroundIntensity.value=k.backgroundIntensity,m.material.toneMapped=Pe.getTransfer(C.colorSpace)!==Ye,C.matrixAutoUpdate===!0&&C.updateMatrix(),m.material.uniforms.uvTransform.value.copy(C.matrix),(x!==C||v!==C.version||_!==o.toneMapping)&&(m.material.needsUpdate=!0,x=C,v=C.version,_=o.toneMapping),m.layers.enableAll(),O.unshift(m,m.geometry,m.material,0,0,null))}function y(O,k){O.getRGB(wu,hS(o)),i.buffers.color.setClear(wu.r,wu.g,wu.b,k,f)}function S(){p!==void 0&&(p.geometry.dispose(),p.material.dispose(),p=void 0),m!==void 0&&(m.geometry.dispose(),m.material.dispose(),m=void 0)}return{getClearColor:function(){return d},setClearColor:function(O,k=1){d.set(O),h=k,y(d,h)},getClearAlpha:function(){return h},setClearAlpha:function(O){h=O,y(d,h)},render:R,addToRenderList:N,dispose:S}}function wA(o,e){const i=o.getParameter(o.MAX_VERTEX_ATTRIBS),r={},c=_(null);let f=c,d=!1;function h(j,J,P,q,$){let Y=!1;const Q=v(j,q,P,J);f!==Q&&(f=Q,p(f.object)),Y=M(j,q,P,$),Y&&R(j,q,P,$),$!==null&&e.update($,o.ELEMENT_ARRAY_BUFFER),(Y||d)&&(d=!1,C(j,J,P,q),$!==null&&o.bindBuffer(o.ELEMENT_ARRAY_BUFFER,e.get($).buffer))}function m(){return o.createVertexArray()}function p(j){return o.bindVertexArray(j)}function x(j){return o.deleteVertexArray(j)}function v(j,J,P,q){const $=q.wireframe===!0;let Y=r[J.id];Y===void 0&&(Y={},r[J.id]=Y);const Q=j.isInstancedMesh===!0?j.id:0;let pt=Y[Q];pt===void 0&&(pt={},Y[Q]=pt);let ot=pt[P.id];ot===void 0&&(ot={},pt[P.id]=ot);let vt=ot[$];return vt===void 0&&(vt=_(m()),ot[$]=vt),vt}function _(j){const J=[],P=[],q=[];for(let $=0;$<i;$++)J[$]=0,P[$]=0,q[$]=0;return{geometry:null,program:null,wireframe:!1,newAttributes:J,enabledAttributes:P,attributeDivisors:q,object:j,attributes:{},index:null}}function M(j,J,P,q){const $=f.attributes,Y=J.attributes;let Q=0;const pt=P.getAttributes();for(const ot in pt)if(pt[ot].location>=0){const Tt=$[ot];let Zt=Y[ot];if(Zt===void 0&&(ot==="instanceMatrix"&&j.instanceMatrix&&(Zt=j.instanceMatrix),ot==="instanceColor"&&j.instanceColor&&(Zt=j.instanceColor)),Tt===void 0||Tt.attribute!==Zt||Zt&&Tt.data!==Zt.data)return!0;Q++}return f.attributesNum!==Q||f.index!==q}function R(j,J,P,q){const $={},Y=J.attributes;let Q=0;const pt=P.getAttributes();for(const ot in pt)if(pt[ot].location>=0){let Tt=Y[ot];Tt===void 0&&(ot==="instanceMatrix"&&j.instanceMatrix&&(Tt=j.instanceMatrix),ot==="instanceColor"&&j.instanceColor&&(Tt=j.instanceColor));const Zt={};Zt.attribute=Tt,Tt&&Tt.data&&(Zt.data=Tt.data),$[ot]=Zt,Q++}f.attributes=$,f.attributesNum=Q,f.index=q}function N(){const j=f.newAttributes;for(let J=0,P=j.length;J<P;J++)j[J]=0}function y(j){S(j,0)}function S(j,J){const P=f.newAttributes,q=f.enabledAttributes,$=f.attributeDivisors;P[j]=1,q[j]===0&&(o.enableVertexAttribArray(j),q[j]=1),$[j]!==J&&(o.vertexAttribDivisor(j,J),$[j]=J)}function O(){const j=f.newAttributes,J=f.enabledAttributes;for(let P=0,q=J.length;P<q;P++)J[P]!==j[P]&&(o.disableVertexAttribArray(P),J[P]=0)}function k(j,J,P,q,$,Y,Q){Q===!0?o.vertexAttribIPointer(j,J,P,$,Y):o.vertexAttribPointer(j,J,P,q,$,Y)}function C(j,J,P,q){N();const $=q.attributes,Y=P.getAttributes(),Q=J.defaultAttributeValues;for(const pt in Y){const ot=Y[pt];if(ot.location>=0){let vt=$[pt];if(vt===void 0&&(pt==="instanceMatrix"&&j.instanceMatrix&&(vt=j.instanceMatrix),pt==="instanceColor"&&j.instanceColor&&(vt=j.instanceColor)),vt!==void 0){const Tt=vt.normalized,Zt=vt.itemSize,Yt=e.get(vt);if(Yt===void 0)continue;const H=Yt.buffer,F=Yt.type,gt=Yt.bytesPerElement,G=F===o.INT||F===o.UNSIGNED_INT||vt.gpuType===em;if(vt.isInterleavedBufferAttribute){const st=vt.data,At=st.stride,Dt=vt.offset;if(st.isInstancedInterleavedBuffer){for(let xt=0;xt<ot.locationSize;xt++)S(ot.location+xt,st.meshPerAttribute);j.isInstancedMesh!==!0&&q._maxInstanceCount===void 0&&(q._maxInstanceCount=st.meshPerAttribute*st.count)}else for(let xt=0;xt<ot.locationSize;xt++)y(ot.location+xt);o.bindBuffer(o.ARRAY_BUFFER,H);for(let xt=0;xt<ot.locationSize;xt++)k(ot.location+xt,Zt/ot.locationSize,F,Tt,At*gt,(Dt+Zt/ot.locationSize*xt)*gt,G)}else{if(vt.isInstancedBufferAttribute){for(let st=0;st<ot.locationSize;st++)S(ot.location+st,vt.meshPerAttribute);j.isInstancedMesh!==!0&&q._maxInstanceCount===void 0&&(q._maxInstanceCount=vt.meshPerAttribute*vt.count)}else for(let st=0;st<ot.locationSize;st++)y(ot.location+st);o.bindBuffer(o.ARRAY_BUFFER,H);for(let st=0;st<ot.locationSize;st++)k(ot.location+st,Zt/ot.locationSize,F,Tt,Zt*gt,Zt/ot.locationSize*st*gt,G)}}else if(Q!==void 0){const Tt=Q[pt];if(Tt!==void 0)switch(Tt.length){case 2:o.vertexAttrib2fv(ot.location,Tt);break;case 3:o.vertexAttrib3fv(ot.location,Tt);break;case 4:o.vertexAttrib4fv(ot.location,Tt);break;default:o.vertexAttrib1fv(ot.location,Tt)}}}}O()}function D(){L();for(const j in r){const J=r[j];for(const P in J){const q=J[P];for(const $ in q){const Y=q[$];for(const Q in Y)x(Y[Q].object),delete Y[Q];delete q[$]}}delete r[j]}}function U(j){if(r[j.id]===void 0)return;const J=r[j.id];for(const P in J){const q=J[P];for(const $ in q){const Y=q[$];for(const Q in Y)x(Y[Q].object),delete Y[Q];delete q[$]}}delete r[j.id]}function B(j){for(const J in r){const P=r[J];for(const q in P){const $=P[q];if($[j.id]===void 0)continue;const Y=$[j.id];for(const Q in Y)x(Y[Q].object),delete Y[Q];delete $[j.id]}}}function b(j){for(const J in r){const P=r[J],q=j.isInstancedMesh===!0?j.id:0,$=P[q];if($!==void 0){for(const Y in $){const Q=$[Y];for(const pt in Q)x(Q[pt].object),delete Q[pt];delete $[Y]}delete P[q],Object.keys(P).length===0&&delete r[J]}}}function L(){X(),d=!0,f!==c&&(f=c,p(f.object))}function X(){c.geometry=null,c.program=null,c.wireframe=!1}return{setup:h,reset:L,resetDefaultState:X,dispose:D,releaseStatesOfGeometry:U,releaseStatesOfObject:b,releaseStatesOfProgram:B,initAttributes:N,enableAttribute:y,disableUnusedAttributes:O}}function CA(o,e,i){let r;function c(m){r=m}function f(m,p){o.drawArrays(r,m,p),i.update(p,r,1)}function d(m,p,x){x!==0&&(o.drawArraysInstanced(r,m,p,x),i.update(p,r,x))}function h(m,p,x){if(x===0)return;e.get("WEBGL_multi_draw").multiDrawArraysWEBGL(r,m,0,p,0,x);let _=0;for(let M=0;M<x;M++)_+=p[M];i.update(_,r,1)}this.setMode=c,this.render=f,this.renderInstances=d,this.renderMultiDraw=h}function NA(o,e,i,r){let c;function f(){if(c!==void 0)return c;if(e.has("EXT_texture_filter_anisotropic")===!0){const B=e.get("EXT_texture_filter_anisotropic");c=o.getParameter(B.MAX_TEXTURE_MAX_ANISOTROPY_EXT)}else c=0;return c}function d(B){return!(B!==Gi&&r.convert(B)!==o.getParameter(o.IMPLEMENTATION_COLOR_READ_FORMAT))}function h(B){const b=B===da&&(e.has("EXT_color_buffer_half_float")||e.has("EXT_color_buffer_float"));return!(B!==pi&&B!==Hi&&!b&&r.convert(B)!==o.getParameter(o.IMPLEMENTATION_COLOR_READ_TYPE))}function m(B){if(B==="highp"){if(o.getShaderPrecisionFormat(o.VERTEX_SHADER,o.HIGH_FLOAT).precision>0&&o.getShaderPrecisionFormat(o.FRAGMENT_SHADER,o.HIGH_FLOAT).precision>0)return"highp";B="mediump"}return B==="mediump"&&o.getShaderPrecisionFormat(o.VERTEX_SHADER,o.MEDIUM_FLOAT).precision>0&&o.getShaderPrecisionFormat(o.FRAGMENT_SHADER,o.MEDIUM_FLOAT).precision>0?"mediump":"lowp"}let p=i.precision!==void 0?i.precision:"highp";const x=m(p);x!==p&&(oe("WebGLRenderer:",p,"not supported, using",x,"instead."),p=x);const v=i.logarithmicDepthBuffer===!0,_=i.reversedDepthBuffer===!0&&e.has("EXT_clip_control");i.reversedDepthBuffer===!0&&_===!1&&oe("WebGLRenderer: Unable to use reversed depth buffer due to missing EXT_clip_control extension. Fallback to default depth buffer.");const M=o.getParameter(o.MAX_TEXTURE_IMAGE_UNITS),R=o.getParameter(o.MAX_VERTEX_TEXTURE_IMAGE_UNITS),N=o.getParameter(o.MAX_TEXTURE_SIZE),y=o.getParameter(o.MAX_CUBE_MAP_TEXTURE_SIZE),S=o.getParameter(o.MAX_VERTEX_ATTRIBS),O=o.getParameter(o.MAX_VERTEX_UNIFORM_VECTORS),k=o.getParameter(o.MAX_VARYING_VECTORS),C=o.getParameter(o.MAX_FRAGMENT_UNIFORM_VECTORS),D=o.getParameter(o.MAX_SAMPLES),U=o.getParameter(o.SAMPLES);return{isWebGL2:!0,getMaxAnisotropy:f,getMaxPrecision:m,textureFormatReadable:d,textureTypeReadable:h,precision:p,logarithmicDepthBuffer:v,reversedDepthBuffer:_,maxTextures:M,maxVertexTextures:R,maxTextureSize:N,maxCubemapSize:y,maxAttributes:S,maxVertexUniforms:O,maxVaryings:k,maxFragmentUniforms:C,maxSamples:D,samples:U}}function DA(o){const e=this;let i=null,r=0,c=!1,f=!1;const d=new Ss,h=new de,m={value:null,needsUpdate:!1};this.uniform=m,this.numPlanes=0,this.numIntersection=0,this.init=function(v,_){const M=v.length!==0||_||r!==0||c;return c=_,r=v.length,M},this.beginShadows=function(){f=!0,x(null)},this.endShadows=function(){f=!1},this.setGlobalState=function(v,_){i=x(v,_,0)},this.setState=function(v,_,M){const R=v.clippingPlanes,N=v.clipIntersection,y=v.clipShadows,S=o.get(v);if(!c||R===null||R.length===0||f&&!y)f?x(null):p();else{const O=f?0:r,k=O*4;let C=S.clippingState||null;m.value=C,C=x(R,_,k,M);for(let D=0;D!==k;++D)C[D]=i[D];S.clippingState=C,this.numIntersection=N?this.numPlanes:0,this.numPlanes+=O}};function p(){m.value!==i&&(m.value=i,m.needsUpdate=r>0),e.numPlanes=r,e.numIntersection=0}function x(v,_,M,R){const N=v!==null?v.length:0;let y=null;if(N!==0){if(y=m.value,R!==!0||y===null){const S=M+N*4,O=_.matrixWorldInverse;h.getNormalMatrix(O),(y===null||y.length<S)&&(y=new Float32Array(S));for(let k=0,C=M;k!==N;++k,C+=4)d.copy(v[k]).applyMatrix4(O,h),d.normal.toArray(y,C),y[C+3]=d.constant}m.value=y,m.needsUpdate=!0}return e.numPlanes=N,e.numIntersection=0,y}}const co=4,UA=6,LA=20,OA=256,Tl=new gm,mx=new he;let tp=null,ep=0,np=0,ip=!1;const PA=new dt,Ks=new dt;class Yp{constructor(e){this._renderer=e,this._pingPongRenderTarget=null,this._lodMax=0,this._cubeSize=0,this._sizeLods=[],this._lodMeshes=[],this._backgroundBox=null,this._cubemapMaterial=null,this._equirectMaterial=null,this._blurMaterial=null,this._ggxMaterial=null}fromScene(e,i=0,r=.1,c=100,f={}){const{size:d=256,position:h=PA}=f;tp=this._renderer.getRenderTarget(),ep=this._renderer.getActiveCubeFace(),np=this._renderer.getActiveMipmapLevel(),ip=this._renderer.xr.enabled,this._renderer.xr.enabled=!1,this._setSize(d);const m=this._allocateTargets();return m.depthBuffer=!0,this._sceneToCubeUV(e,r,c,m,h),i>0&&this._blur(m,0,0,i),this._applyPMREM(m),this._cleanup(m),m}fromEquirectangular(e,i=null){return this._fromTexture(e,i)}fromCubemap(e,i=null){return this._fromTexture(e,i)}compileCubemapShader(){this._cubemapMaterial===null&&(this._cubemapMaterial=_x(),this._compileMaterial(this._cubemapMaterial))}compileEquirectangularShader(){this._equirectMaterial===null&&(this._equirectMaterial=vx(),this._compileMaterial(this._equirectMaterial))}dispose(){this._dispose(),this._cubemapMaterial!==null&&this._cubemapMaterial.dispose(),this._equirectMaterial!==null&&this._equirectMaterial.dispose(),this._backgroundBox!==null&&(this._backgroundBox.geometry.dispose(),this._backgroundBox.material.dispose())}_setSize(e){this._lodMax=Math.floor(Math.log2(e)),this._cubeSize=Math.pow(2,this._lodMax)}_dispose(){this._blurMaterial!==null&&this._blurMaterial.dispose(),this._ggxMaterial!==null&&this._ggxMaterial.dispose(),this._pingPongRenderTarget!==null&&this._pingPongRenderTarget.dispose();for(let e=0;e<this._lodMeshes.length;e++)this._lodMeshes[e].geometry.dispose()}_cleanup(e){this._renderer.setRenderTarget(tp,ep,np),this._renderer.xr.enabled=ip,e.scissorTest=!1,so(e,0,0,e.width,e.height)}_fromTexture(e,i){e.mapping===tr||e.mapping===po?this._setSize(e.image.length===0?16:e.image[0].width||e.image[0].image.width):this._setSize(e.image.width/4),tp=this._renderer.getRenderTarget(),ep=this._renderer.getActiveCubeFace(),np=this._renderer.getActiveMipmapLevel(),ip=this._renderer.xr.enabled,this._renderer.xr.enabled=!1;const r=i||this._allocateTargets();return this._textureToCubeUV(e,r),this._applyPMREM(r),this._cleanup(r),r}_allocateTargets(){const e=3*Math.max(this._cubeSize,112),i=4*this._cubeSize,r={magFilter:zn,minFilter:zn,generateMipmaps:!1,type:da,format:Gi,colorSpace:Fu,depthBuffer:!1},c=gx(e,i,r);if(this._pingPongRenderTarget===null||this._pingPongRenderTarget.width!==e||this._pingPongRenderTarget.height!==i){this._pingPongRenderTarget!==null&&this._dispose(),this._pingPongRenderTarget=gx(e,i,r);const{_lodMax:f}=this;({lodMeshes:this._lodMeshes,sizeLods:this._sizeLods}=IA(f)),this._blurMaterial=zA(f,e,i),this._ggxMaterial=BA(f,e,i)}return c}_compileMaterial(e){const i=new _n(new ki,e);this._renderer.compile(i,Tl)}_sceneToCubeUV(e,i,r,c,f){const m=new ei(90,1,i,r),p=[1,-1,1,1,1,1],x=[1,1,1,-1,-1,-1],v=this._renderer,_=v.autoClear,M=v.toneMapping;v.getClearColor(mx),v.toneMapping=ca,v.autoClear=!1,v.state.buffers.depth.getReversed()&&(v.setRenderTarget(c),v.clearDepth(),v.setRenderTarget(null)),this._backgroundBox===null&&(this._backgroundBox=new _n(new Ms,new dm({name:"PMREM.Background",side:qn,depthWrite:!1,depthTest:!1})));const N=this._backgroundBox,y=N.material;let S=!1;const O=e.background;O?O.isColor&&(y.color.copy(O),e.background=null,S=!0):(y.color.copy(mx),S=!0);for(let k=0;k<6;k++){const C=k%3;C===0?(m.up.set(0,p[k],0),m.position.set(f.x,f.y,f.z),m.lookAt(f.x+x[k],f.y,f.z)):C===1?(m.up.set(0,0,p[k]),m.position.set(f.x,f.y,f.z),m.lookAt(f.x,f.y+x[k],f.z)):(m.up.set(0,p[k],0),m.position.set(f.x,f.y,f.z),m.lookAt(f.x,f.y,f.z+x[k]));const D=this._cubeSize;so(c,C*D,k>2?D:0,D,D),v.setRenderTarget(c),S&&v.render(N,m),v.render(e,m)}v.toneMapping=M,v.autoClear=_,e.background=O}_textureToCubeUV(e,i){const r=this._renderer,c=e.mapping===tr||e.mapping===po;c?(this._cubemapMaterial===null&&(this._cubemapMaterial=_x()),this._cubemapMaterial.uniforms.flipEnvMap.value=e.isRenderTargetTexture===!1?-1:1):this._equirectMaterial===null&&(this._equirectMaterial=vx());const f=c?this._cubemapMaterial:this._equirectMaterial,d=this._lodMeshes[0];d.material=f;const h=f.uniforms;h.envMap.value=e;const m=this._cubeSize;so(i,0,0,3*m,2*m),r.setRenderTarget(i),r.render(d,Tl)}_applyPMREM(e){const i=this._renderer,r=i.autoClear;i.autoClear=!1;const c=this._lodMeshes.length;for(let f=1;f<c;f++)this._applyGGXFilter(e,f-1,f);i.autoClear=r}_applyGGXFilter(e,i,r){const c=this._renderer,f=this._pingPongRenderTarget,d=this._ggxMaterial,h=this._lodMeshes[r];h.material=d;const m=d.uniforms,p=r/(this._lodMeshes.length-1),x=i/(this._lodMeshes.length-1),v=Math.sqrt(p*p-x*x),_=p*1.25,M=v*_,{_lodMax:R}=this,N=this._sizeLods[r],y=3*N*(r>R-co?r-R+co:0),S=4*(this._cubeSize-N);m.envMap.value=e.texture,m.roughness.value=M,m.mipInt.value=R-i,so(f,y,S,3*N,2*N),c.setRenderTarget(f),c.render(h,Tl),m.envMap.value=f.texture,m.roughness.value=0,m.mipInt.value=R-r,so(e,y,S,3*N,2*N),c.setRenderTarget(e),c.render(h,Tl)}_blur(e,i,r,c){const f=this._pingPongRenderTarget,d=Math.min(c,Math.PI)/Math.SQRT2;this._blurPass(e,f,i,r,d),this._blurPass(f,e,r,r,d)}_blurPass(e,i,r,c,f){const d=this._renderer,h=this._blurMaterial,m=this._lodMeshes[c];m.material=h;const p=h.uniforms;p.envMap.value=e.texture,p.sigma.value=f,p.mipInt.value=this._lodMax-r;const x=this._sizeLods[c],v=3*x*(c>this._lodMax-co?c-this._lodMax+co:0),_=4*(this._cubeSize-x);so(i,v,_,3*x,2*x),d.setRenderTarget(i),d.render(m,Tl)}}function IA(o){const e=[],i=[];let r=o;const c=o-co+1+UA;for(let f=0;f<c;f++){const d=Math.pow(2,r);e.push(d);const h=1/(d-2),m=-h,p=1+h,x=[m,m,p,m,p,p,m,m,p,p,m,p],v=6,_=6,M=3,R=new Float32Array(M*_*v),N=new Float32Array(M*_*v);for(let S=0;S<v;S++){const O=S%3*2/3-1,k=S>2?0:-1,C=[O,k,0,O+2/3,k,0,O+2/3,k+1,0,O,k,0,O+2/3,k+1,0,O,k+1,0];R.set(C,M*_*S);for(let D=0;D<_;D++){const U=x[D*2]*2-1,B=x[D*2+1]*2-1;S===0?Ks.set(1,B,U):S===1?Ks.set(-U,1,-B):S===2?Ks.set(-U,B,1):S===3?Ks.set(-1,B,-U):S===4?Ks.set(-U,-1,B):Ks.set(U,B,-1),Ks.toArray(N,(S*_+D)*M)}}const y=new ki;y.setAttribute("position",new ua(R,M)),y.setAttribute("outputDirection",new ua(N,M)),i.push(new _n(y,null)),r>co&&r--}return{lodMeshes:i,sizeLods:e}}function gx(o,e,i){const r=new Vi(o,e,i);return r.texture.mapping=ku,r.texture.name="PMREM.cubeUv",r.scissorTest=!0,r}function so(o,e,i,r,c){o.viewport.set(e,i,r,c),o.scissor.set(e,i,r,c)}function BA(o,e,i){return new ha({name:"PMREMGGXConvolution",defines:{GGX_SAMPLES:OA,CUBEUV_TEXEL_WIDTH:1/e,CUBEUV_TEXEL_HEIGHT:1/i,CUBEUV_MAX_MIP:`${o}.0`},uniforms:{envMap:{value:null},roughness:{value:0},mipInt:{value:0}},vertexShader:Wu(),fragmentShader:`

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
		`,blending:Ia,depthTest:!1,depthWrite:!1})}function zA(o,e,i){return new ha({name:"SphericalGaussianBlur",defines:{SAMPLES:LA,CUBEUV_TEXEL_WIDTH:1/e,CUBEUV_TEXEL_HEIGHT:1/i,CUBEUV_MAX_MIP:`${o}.0`},uniforms:{envMap:{value:null},sigma:{value:0},mipInt:{value:0}},vertexShader:Wu(),fragmentShader:`

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
		`,blending:Ia,depthTest:!1,depthWrite:!1})}function vx(){return new ha({name:"EquirectangularToCubeUV",uniforms:{envMap:{value:null}},vertexShader:Wu(),fragmentShader:`

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
		`,blending:Ia,depthTest:!1,depthWrite:!1})}function _x(){return new ha({name:"CubemapToCubeUV",uniforms:{envMap:{value:null},flipEnvMap:{value:-1}},vertexShader:Wu(),fragmentShader:`

			precision mediump float;
			precision mediump int;

			uniform float flipEnvMap;

			varying vec3 vOutputDirection;

			uniform samplerCube envMap;

			void main() {

				gl_FragColor = textureCube( envMap, vec3( flipEnvMap * vOutputDirection.x, vOutputDirection.yz ) );

			}
		`,blending:Ia,depthTest:!1,depthWrite:!1})}function Wu(){return`

		precision mediump float;
		precision mediump int;

		attribute vec3 outputDirection;

		varying vec3 vOutputDirection;

		void main() {

			vOutputDirection = outputDirection;
			gl_Position = vec4( position, 1.0 );

		}
	`}class _S extends Vi{constructor(e=1,i={}){super(e,e,i),this.isWebGLCubeRenderTarget=!0;const r={width:e,height:e,depth:1},c=[r,r,r,r,r,r];this.texture=new fS(c),this._setTextureOptions(i),this.texture.isRenderTargetTexture=!0}fromEquirectangularTexture(e,i){this.texture.type=i.type,this.texture.colorSpace=i.colorSpace,this.texture.generateMipmaps=i.generateMipmaps,this.texture.minFilter=i.minFilter,this.texture.magFilter=i.magFilter;const r={uniforms:{tEquirect:{value:null}},vertexShader:`

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
			`},c=new Ms(5,5,5),f=new ha({name:"CubemapFromEquirect",uniforms:vo(r.uniforms),vertexShader:r.vertexShader,fragmentShader:r.fragmentShader,side:qn,blending:Ia});f.uniforms.tEquirect.value=i;const d=new _n(c,f),h=i.minFilter;return i.minFilter===Qs&&(i.minFilter=zn),new Vb(1,10,this).update(e,d),i.minFilter=h,d.geometry.dispose(),d.material.dispose(),this}clear(e,i=!0,r=!0,c=!0){const f=e.getRenderTarget();for(let d=0;d<6;d++)e.setRenderTarget(this,d),e.clear(i,r,c);e.setRenderTarget(f)}}function FA(o){let e=new WeakMap,i=new WeakMap,r=null;function c(_,M=!1){return _==null?null:M?d(_):f(_)}function f(_){if(_&&_.isTexture){const M=_.mapping;if(M===Ah||M===Rh)if(e.has(_)){const R=e.get(_).texture;return h(R,_.mapping)}else{const R=_.image;if(R&&R.height>0){const N=new _S(R.height);return N.fromEquirectangularTexture(o,_),e.set(_,N),_.addEventListener("dispose",p),h(N.texture,_.mapping)}else return null}}return _}function d(_){if(_&&_.isTexture){const M=_.mapping,R=M===Ah||M===Rh,N=M===tr||M===po;if(R||N){let y=i.get(_);const S=y!==void 0?y.texture.pmremVersion:0;if(_.isRenderTargetTexture&&_.pmremVersion!==S)return r===null&&(r=new Yp(o)),y=R?r.fromEquirectangular(_,y):r.fromCubemap(_,y),y.texture.pmremVersion=_.pmremVersion,i.set(_,y),y.texture;if(y!==void 0)return y.texture;{const O=_.image;return R&&O&&O.height>0||N&&O&&m(O)?(r===null&&(r=new Yp(o)),y=R?r.fromEquirectangular(_):r.fromCubemap(_),y.texture.pmremVersion=_.pmremVersion,i.set(_,y),_.addEventListener("dispose",x),y.texture):null}}}return _}function h(_,M){return M===Ah?_.mapping=tr:M===Rh&&(_.mapping=po),_}function m(_){let M=0;const R=6;for(let N=0;N<R;N++)_[N]!==void 0&&M++;return M===R}function p(_){const M=_.target;M.removeEventListener("dispose",p);const R=e.get(M);R!==void 0&&(e.delete(M),R.dispose())}function x(_){const M=_.target;M.removeEventListener("dispose",x);const R=i.get(M);R!==void 0&&(i.delete(M),R.dispose())}function v(){e=new WeakMap,i=new WeakMap,r!==null&&(r.dispose(),r=null)}return{get:c,dispose:v}}function HA(o){const e={};function i(r){if(e[r]!==void 0)return e[r];const c=o.getExtension(r);return e[r]=c,c}return{has:function(r){return i(r)!==null},init:function(){i("EXT_color_buffer_float"),i("WEBGL_clip_cull_distance"),i("OES_texture_float_linear"),i("EXT_color_buffer_half_float"),i("WEBGL_multisampled_render_to_texture"),i("WEBGL_render_shared_exponent")},get:function(r){const c=i(r);return c===null&&fo("WebGLRenderer: "+r+" extension not supported."),c}}}function GA(o,e,i,r){const c={},f=new WeakMap;function d(v){const _=v.target;_.index!==null&&e.remove(_.index);for(const R in _.attributes)e.remove(_.attributes[R]);_.removeEventListener("dispose",d),delete c[_.id];const M=f.get(_);M&&(e.remove(M),f.delete(_)),r.releaseStatesOfGeometry(_),_.isInstancedBufferGeometry===!0&&delete _._maxInstanceCount,i.memory.geometries--}function h(v,_){return c[_.id]===!0||(_.addEventListener("dispose",d),c[_.id]=!0,i.memory.geometries++),_}function m(v){const _=v.attributes;for(const M in _)e.update(_[M],o.ARRAY_BUFFER)}function p(v){const _=[],M=v.index,R=v.attributes.position;let N=0;if(R===void 0)return;if(M!==null){const O=M.array;N=M.version;for(let k=0,C=O.length;k<C;k+=3){const D=O[k+0],U=O[k+1],B=O[k+2];_.push(D,U,U,B,B,D)}}else{const O=R.array;N=R.version;for(let k=0,C=O.length/3-1;k<C;k+=3){const D=k+0,U=k+1,B=k+2;_.push(D,U,U,B,B,D)}}const y=new(R.count>=65535?cS:lS)(_,1);y.version=N;const S=f.get(v);S&&e.remove(S),f.set(v,y)}function x(v){const _=f.get(v);if(_){const M=v.index;M!==null&&_.version<M.version&&p(v)}else p(v);return f.get(v)}return{get:h,update:m,getWireframeAttribute:x}}function VA(o,e,i){let r;function c(v){r=v}let f,d;function h(v){f=v.type,d=v.bytesPerElement}function m(v,_){o.drawElements(r,_,f,v*d),i.update(_,r,1)}function p(v,_,M){M!==0&&(o.drawElementsInstanced(r,_,f,v*d,M),i.update(_,r,M))}function x(v,_,M){if(M===0)return;e.get("WEBGL_multi_draw").multiDrawElementsWEBGL(r,_,0,f,v,0,M);let N=0;for(let y=0;y<M;y++)N+=_[y];i.update(N,r,1)}this.setMode=c,this.setIndex=h,this.render=m,this.renderInstances=p,this.renderMultiDraw=x}function kA(o){const e={geometries:0,textures:0},i={frame:0,calls:0,triangles:0,points:0,lines:0};function r(f,d,h){switch(i.calls++,d){case o.TRIANGLES:i.triangles+=h*(f/3);break;case o.LINES:i.lines+=h*(f/2);break;case o.LINE_STRIP:i.lines+=h*(f-1);break;case o.LINE_LOOP:i.lines+=h*f;break;case o.POINTS:i.points+=h*f;break;default:ze("WebGLInfo: Unknown draw mode:",d);break}}function c(){i.calls=0,i.triangles=0,i.points=0,i.lines=0}return{memory:e,render:i,programs:null,autoReset:!0,reset:c,update:r}}function XA(o,e,i){const r=new WeakMap,c=new on;function f(d,h,m){const p=d.morphTargetInfluences,x=h.morphAttributes.position||h.morphAttributes.normal||h.morphAttributes.color,v=x!==void 0?x.length:0;let _=r.get(h);if(_===void 0||_.count!==v){let X=function(){b.dispose(),r.delete(h),h.removeEventListener("dispose",X)};var M=X;_!==void 0&&_.texture.dispose();const R=h.morphAttributes.position!==void 0,N=h.morphAttributes.normal!==void 0,y=h.morphAttributes.color!==void 0,S=h.morphAttributes.position||[],O=h.morphAttributes.normal||[],k=h.morphAttributes.color||[];let C=0;R===!0&&(C=1),N===!0&&(C=2),y===!0&&(C=3);let D=h.attributes.position.count*C,U=1;D>e.maxTextureSize&&(U=Math.ceil(D/e.maxTextureSize),D=e.maxTextureSize);const B=new Float32Array(D*U*4*v),b=new sS(B,D,U,v);b.type=Hi,b.needsUpdate=!0;const L=C*4;for(let j=0;j<v;j++){const J=S[j],P=O[j],q=k[j],$=D*U*4*j;for(let Y=0;Y<J.count;Y++){const Q=Y*L;R===!0&&(c.fromBufferAttribute(J,Y),B[$+Q+0]=c.x,B[$+Q+1]=c.y,B[$+Q+2]=c.z,B[$+Q+3]=0),N===!0&&(c.fromBufferAttribute(P,Y),B[$+Q+4]=c.x,B[$+Q+5]=c.y,B[$+Q+6]=c.z,B[$+Q+7]=0),y===!0&&(c.fromBufferAttribute(q,Y),B[$+Q+8]=c.x,B[$+Q+9]=c.y,B[$+Q+10]=c.z,B[$+Q+11]=q.itemSize===4?c.w:1)}}_={count:v,texture:b,size:new Re(D,U)},r.set(h,_),h.addEventListener("dispose",X)}if(d.isInstancedMesh===!0&&d.morphTexture!==null)m.getUniforms().setValue(o,"morphTexture",d.morphTexture,i);else{let R=0;for(let y=0;y<p.length;y++)R+=p[y];const N=h.morphTargetsRelative?1:1-R;m.getUniforms().setValue(o,"morphTargetBaseInfluence",N),m.getUniforms().setValue(o,"morphTargetInfluences",p)}m.getUniforms().setValue(o,"morphTargetsTexture",_.texture,i),m.getUniforms().setValue(o,"morphTargetsTextureSize",_.size)}return{update:f}}function WA(o,e,i,r,c){let f=new WeakMap;function d(p){const x=c.render.frame,v=p.geometry,_=e.get(p,v);if(f.get(_)!==x&&(e.update(_),f.set(_,x)),p.isInstancedMesh&&(p.hasEventListener("dispose",m)===!1&&p.addEventListener("dispose",m),f.get(p)!==x&&(i.update(p.instanceMatrix,o.ARRAY_BUFFER),p.instanceColor!==null&&i.update(p.instanceColor,o.ARRAY_BUFFER),f.set(p,x))),p.isSkinnedMesh){const M=p.skeleton;f.get(M)!==x&&(M.update(),f.set(M,x))}return _}function h(){f=new WeakMap}function m(p){const x=p.target;x.removeEventListener("dispose",m),r.releaseStatesOfObject(x),i.remove(x.instanceMatrix),x.instanceColor!==null&&i.remove(x.instanceColor)}return{update:d,dispose:h}}const qA={[Wx]:"LINEAR_TONE_MAPPING",[qx]:"REINHARD_TONE_MAPPING",[Yx]:"CINEON_TONE_MAPPING",[tm]:"ACES_FILMIC_TONE_MAPPING",[Zx]:"AGX_TONE_MAPPING",[Kx]:"NEUTRAL_TONE_MAPPING",[jx]:"CUSTOM_TONE_MAPPING"};function YA(o,e,i,r,c,f){const d=new Vi(e,i,{type:o,depthBuffer:c,stencilBuffer:f,samples:r?4:0,storeMultisampledDepthBuffer:!1,storeMultisampledStencilBuffer:!1,resolveDepthBuffer:!1,resolveStencilBuffer:!1});let h=null,m=null;const p=new ki;p.setAttribute("position",new Yn([-1,3,0,-1,-1,0,3,-1,0],3)),p.setAttribute("uv",new Yn([0,2,0,0,2,0],2));const x=new Ob({uniforms:{tDiffuse:{value:null}},vertexShader:`
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
			}`,depthTest:!1,depthWrite:!1}),v=new _n(p,x),_=new gm(-1,1,1,-1,0,1);let M=null,R=null,N=!1,y,S=null,O=[],k=!1;this.setSize=function(C,D){d.setSize(C,D),h!==null&&h.setSize(C,D),m!==null&&m.setSize(C,D);for(let U=0;U<O.length;U++){const B=O[U];B.setSize&&B.setSize(C,D)}},this.setEffects=function(C){O=C,k=O.length>0&&O[0].isRenderPass===!0;const D=d.width,U=d.height;O.length>0&&h===null&&(h=new Vi(D,U,{type:da,depthBuffer:!1,stencilBuffer:!1}),m=new Vi(D,U,{type:da,depthBuffer:!1,stencilBuffer:!1}));for(let B=0;B<O.length;B++){const b=O[B];b.setSize&&b.setSize(D,U)}},this.begin=function(C,D){if(N||C.toneMapping===ca&&O.length===0)return!1;if(S=D,D!==null){const U=D.width,B=D.height;(d.width!==U||d.height!==B)&&this.setSize(U,B)}return k===!1&&C.setRenderTarget(d),y=C.toneMapping,C.toneMapping=ca,!0},this.hasRenderPass=function(){return k},this.end=function(C,D){C.toneMapping=y,N=!0;let U=d,B=h;for(let b=0;b<O.length;b++){const L=O[b];L.enabled!==!1&&(L.render(C,B,U,D),L.needsSwap!==!1&&(U=B,B=B===h?m:h))}if(M!==C.outputColorSpace||R!==C.toneMapping){M=C.outputColorSpace,R=C.toneMapping,x.defines={},Pe.getTransfer(M)===Ye&&(x.defines.SRGB_TRANSFER="");const b=qA[R];b&&(x.defines[b]=""),x.needsUpdate=!0}x.uniforms.tDiffuse.value=U.texture,C.setRenderTarget(S),C.render(v,_),S=null,N=!1},this.isCompositing=function(){return N},this.dispose=function(){d.dispose(),h!==null&&h.dispose(),m!==null&&m.dispose(),p.dispose(),x.dispose()}}const xS=new Fn,jp=new Il(1,1),SS=new sS,yS=new cb,MS=new fS,xx=[],Sx=[],yx=new Float32Array(16),Mx=new Float32Array(9),Ex=new Float32Array(4);function xo(o,e,i){const r=o[0];if(r<=0||r>0)return o;const c=e*i;let f=xx[c];if(f===void 0&&(f=new Float32Array(c),xx[c]=f),e!==0){r.toArray(f,0);for(let d=1,h=0;d!==e;++d)h+=i,o[d].toArray(f,h)}return f}function yn(o,e){if(o.length!==e.length)return!1;for(let i=0,r=o.length;i<r;i++)if(o[i]!==e[i])return!1;return!0}function Mn(o,e){for(let i=0,r=e.length;i<r;i++)o[i]=e[i]}function qu(o,e){let i=Sx[e];i===void 0&&(i=new Int32Array(e),Sx[e]=i);for(let r=0;r!==e;++r)i[r]=o.allocateTextureUnit();return i}function jA(o,e){const i=this.cache;i[0]!==e&&(o.uniform1f(this.addr,e),i[0]=e)}function ZA(o,e){const i=this.cache;if(e.x!==void 0)(i[0]!==e.x||i[1]!==e.y)&&(o.uniform2f(this.addr,e.x,e.y),i[0]=e.x,i[1]=e.y);else{if(yn(i,e))return;o.uniform2fv(this.addr,e),Mn(i,e)}}function KA(o,e){const i=this.cache;if(e.x!==void 0)(i[0]!==e.x||i[1]!==e.y||i[2]!==e.z)&&(o.uniform3f(this.addr,e.x,e.y,e.z),i[0]=e.x,i[1]=e.y,i[2]=e.z);else if(e.r!==void 0)(i[0]!==e.r||i[1]!==e.g||i[2]!==e.b)&&(o.uniform3f(this.addr,e.r,e.g,e.b),i[0]=e.r,i[1]=e.g,i[2]=e.b);else{if(yn(i,e))return;o.uniform3fv(this.addr,e),Mn(i,e)}}function QA(o,e){const i=this.cache;if(e.x!==void 0)(i[0]!==e.x||i[1]!==e.y||i[2]!==e.z||i[3]!==e.w)&&(o.uniform4f(this.addr,e.x,e.y,e.z,e.w),i[0]=e.x,i[1]=e.y,i[2]=e.z,i[3]=e.w);else{if(yn(i,e))return;o.uniform4fv(this.addr,e),Mn(i,e)}}function JA(o,e){const i=this.cache,r=e.elements;if(r===void 0){if(yn(i,e))return;o.uniformMatrix2fv(this.addr,!1,e),Mn(i,e)}else{if(yn(i,r))return;Ex.set(r),o.uniformMatrix2fv(this.addr,!1,Ex),Mn(i,r)}}function $A(o,e){const i=this.cache,r=e.elements;if(r===void 0){if(yn(i,e))return;o.uniformMatrix3fv(this.addr,!1,e),Mn(i,e)}else{if(yn(i,r))return;Mx.set(r),o.uniformMatrix3fv(this.addr,!1,Mx),Mn(i,r)}}function tR(o,e){const i=this.cache,r=e.elements;if(r===void 0){if(yn(i,e))return;o.uniformMatrix4fv(this.addr,!1,e),Mn(i,e)}else{if(yn(i,r))return;yx.set(r),o.uniformMatrix4fv(this.addr,!1,yx),Mn(i,r)}}function eR(o,e){const i=this.cache;i[0]!==e&&(o.uniform1i(this.addr,e),i[0]=e)}function nR(o,e){const i=this.cache;if(e.x!==void 0)(i[0]!==e.x||i[1]!==e.y)&&(o.uniform2i(this.addr,e.x,e.y),i[0]=e.x,i[1]=e.y);else{if(yn(i,e))return;o.uniform2iv(this.addr,e),Mn(i,e)}}function iR(o,e){const i=this.cache;if(e.x!==void 0)(i[0]!==e.x||i[1]!==e.y||i[2]!==e.z)&&(o.uniform3i(this.addr,e.x,e.y,e.z),i[0]=e.x,i[1]=e.y,i[2]=e.z);else{if(yn(i,e))return;o.uniform3iv(this.addr,e),Mn(i,e)}}function aR(o,e){const i=this.cache;if(e.x!==void 0)(i[0]!==e.x||i[1]!==e.y||i[2]!==e.z||i[3]!==e.w)&&(o.uniform4i(this.addr,e.x,e.y,e.z,e.w),i[0]=e.x,i[1]=e.y,i[2]=e.z,i[3]=e.w);else{if(yn(i,e))return;o.uniform4iv(this.addr,e),Mn(i,e)}}function sR(o,e){const i=this.cache;i[0]!==e&&(o.uniform1ui(this.addr,e),i[0]=e)}function rR(o,e){const i=this.cache;if(e.x!==void 0)(i[0]!==e.x||i[1]!==e.y)&&(o.uniform2ui(this.addr,e.x,e.y),i[0]=e.x,i[1]=e.y);else{if(yn(i,e))return;o.uniform2uiv(this.addr,e),Mn(i,e)}}function oR(o,e){const i=this.cache;if(e.x!==void 0)(i[0]!==e.x||i[1]!==e.y||i[2]!==e.z)&&(o.uniform3ui(this.addr,e.x,e.y,e.z),i[0]=e.x,i[1]=e.y,i[2]=e.z);else{if(yn(i,e))return;o.uniform3uiv(this.addr,e),Mn(i,e)}}function lR(o,e){const i=this.cache;if(e.x!==void 0)(i[0]!==e.x||i[1]!==e.y||i[2]!==e.z||i[3]!==e.w)&&(o.uniform4ui(this.addr,e.x,e.y,e.z,e.w),i[0]=e.x,i[1]=e.y,i[2]=e.z,i[3]=e.w);else{if(yn(i,e))return;o.uniform4uiv(this.addr,e),Mn(i,e)}}function cR(o,e,i){const r=this.cache,c=i.allocateTextureUnit();r[0]!==c&&(o.uniform1i(this.addr,c),r[0]=c);let f;this.type===o.SAMPLER_2D_SHADOW?(jp.compareFunction=i.isReversedDepthBuffer()?cm:lm,f=jp):f=xS,i.setTexture2D(e||f,c)}function uR(o,e,i){const r=this.cache,c=i.allocateTextureUnit();r[0]!==c&&(o.uniform1i(this.addr,c),r[0]=c),i.setTexture3D(e||yS,c)}function fR(o,e,i){const r=this.cache,c=i.allocateTextureUnit();r[0]!==c&&(o.uniform1i(this.addr,c),r[0]=c),i.setTextureCube(e||MS,c)}function dR(o,e,i){const r=this.cache,c=i.allocateTextureUnit();r[0]!==c&&(o.uniform1i(this.addr,c),r[0]=c),i.setTexture2DArray(e||SS,c)}function hR(o){switch(o){case 5126:return jA;case 35664:return ZA;case 35665:return KA;case 35666:return QA;case 35674:return JA;case 35675:return $A;case 35676:return tR;case 5124:case 35670:return eR;case 35667:case 35671:return nR;case 35668:case 35672:return iR;case 35669:case 35673:return aR;case 5125:return sR;case 36294:return rR;case 36295:return oR;case 36296:return lR;case 35678:case 36198:case 36298:case 36306:case 35682:return cR;case 35679:case 36299:case 36307:return uR;case 35680:case 36300:case 36308:case 36293:return fR;case 36289:case 36303:case 36311:case 36292:return dR}}function pR(o,e){o.uniform1fv(this.addr,e)}function mR(o,e){const i=xo(e,this.size,2);o.uniform2fv(this.addr,i)}function gR(o,e){const i=xo(e,this.size,3);o.uniform3fv(this.addr,i)}function vR(o,e){const i=xo(e,this.size,4);o.uniform4fv(this.addr,i)}function _R(o,e){const i=xo(e,this.size,4);o.uniformMatrix2fv(this.addr,!1,i)}function xR(o,e){const i=xo(e,this.size,9);o.uniformMatrix3fv(this.addr,!1,i)}function SR(o,e){const i=xo(e,this.size,16);o.uniformMatrix4fv(this.addr,!1,i)}function yR(o,e){o.uniform1iv(this.addr,e)}function MR(o,e){o.uniform2iv(this.addr,e)}function ER(o,e){o.uniform3iv(this.addr,e)}function bR(o,e){o.uniform4iv(this.addr,e)}function TR(o,e){o.uniform1uiv(this.addr,e)}function AR(o,e){o.uniform2uiv(this.addr,e)}function RR(o,e){o.uniform3uiv(this.addr,e)}function wR(o,e){o.uniform4uiv(this.addr,e)}function CR(o,e,i){const r=this.cache,c=e.length,f=qu(i,c);yn(r,f)||(o.uniform1iv(this.addr,f),Mn(r,f));let d;this.type===o.SAMPLER_2D_SHADOW?d=jp:d=xS;for(let h=0;h!==c;++h)i.setTexture2D(e[h]||d,f[h])}function NR(o,e,i){const r=this.cache,c=e.length,f=qu(i,c);yn(r,f)||(o.uniform1iv(this.addr,f),Mn(r,f));for(let d=0;d!==c;++d)i.setTexture3D(e[d]||yS,f[d])}function DR(o,e,i){const r=this.cache,c=e.length,f=qu(i,c);yn(r,f)||(o.uniform1iv(this.addr,f),Mn(r,f));for(let d=0;d!==c;++d)i.setTextureCube(e[d]||MS,f[d])}function UR(o,e,i){const r=this.cache,c=e.length,f=qu(i,c);yn(r,f)||(o.uniform1iv(this.addr,f),Mn(r,f));for(let d=0;d!==c;++d)i.setTexture2DArray(e[d]||SS,f[d])}function LR(o){switch(o){case 5126:return pR;case 35664:return mR;case 35665:return gR;case 35666:return vR;case 35674:return _R;case 35675:return xR;case 35676:return SR;case 5124:case 35670:return yR;case 35667:case 35671:return MR;case 35668:case 35672:return ER;case 35669:case 35673:return bR;case 5125:return TR;case 36294:return AR;case 36295:return RR;case 36296:return wR;case 35678:case 36198:case 36298:case 36306:case 35682:return CR;case 35679:case 36299:case 36307:return NR;case 35680:case 36300:case 36308:case 36293:return DR;case 36289:case 36303:case 36311:case 36292:return UR}}class OR{constructor(e,i,r){this.id=e,this.addr=r,this.cache=[],this.type=i.type,this.setValue=hR(i.type)}}class PR{constructor(e,i,r){this.id=e,this.addr=r,this.cache=[],this.type=i.type,this.size=i.size,this.setValue=LR(i.type)}}class IR{constructor(e){this.id=e,this.seq=[],this.map={}}setValue(e,i,r){const c=this.seq;for(let f=0,d=c.length;f!==d;++f){const h=c[f];h.setValue(e,i[h.id],r)}}}const ap=/(\w+)(\])?(\[|\.)?/g;function bx(o,e){o.seq.push(e),o.map[e.id]=e}function BR(o,e,i){const r=o.name,c=r.length;for(ap.lastIndex=0;;){const f=ap.exec(r),d=ap.lastIndex;let h=f[1];const m=f[2]==="]",p=f[3];if(m&&(h=h|0),p===void 0||p==="["&&d+2===c){bx(i,p===void 0?new OR(h,o,e):new PR(h,o,e));break}else{let v=i.map[h];v===void 0&&(v=new IR(h),bx(i,v)),i=v}}}class Pu{constructor(e,i){this.seq=[],this.map={};const r=e.getProgramParameter(i,e.ACTIVE_UNIFORMS);for(let d=0;d<r;++d){const h=e.getActiveUniform(i,d),m=e.getUniformLocation(i,h.name);BR(h,m,this)}const c=[],f=[];for(const d of this.seq)d.type===e.SAMPLER_2D_SHADOW||d.type===e.SAMPLER_CUBE_SHADOW||d.type===e.SAMPLER_2D_ARRAY_SHADOW?c.push(d):f.push(d);c.length>0&&(this.seq=c.concat(f))}setValue(e,i,r,c){const f=this.map[i];f!==void 0&&f.setValue(e,r,c)}setOptional(e,i,r){const c=i[r];c!==void 0&&this.setValue(e,r,c)}static upload(e,i,r,c){for(let f=0,d=i.length;f!==d;++f){const h=i[f],m=r[h.id];m.needsUpdate!==!1&&h.setValue(e,m.value,c)}}static seqWithValue(e,i){const r=[];for(let c=0,f=e.length;c!==f;++c){const d=e[c];d.id in i&&r.push(d)}return r}}function Tx(o,e,i){const r=o.createShader(e);return o.shaderSource(r,i),o.compileShader(r),r}const zR=37297;let FR=0;function HR(o,e){const i=o.split(`
`),r=[],c=Math.max(e-6,0),f=Math.min(e+6,i.length);for(let d=c;d<f;d++){const h=d+1;r.push(`${h===e?">":" "} ${h}: ${i[d]}`)}return r.join(`
`)}const Ax=new de;function GR(o){Pe._getMatrix(Ax,Pe.workingColorSpace,o);const e=`mat3( ${Ax.elements.map(i=>i.toFixed(4))} )`;switch(Pe.getTransfer(o)){case Hu:return[e,"LinearTransferOETF"];case Ye:return[e,"sRGBTransferOETF"];default:return oe("WebGLProgram: Unsupported color space: ",o),[e,"LinearTransferOETF"]}}function Rx(o,e,i){const r=o.getShaderParameter(e,o.COMPILE_STATUS),f=(o.getShaderInfoLog(e)||"").trim();if(r&&f==="")return"";const d=/ERROR: 0:(\d+)/.exec(f);if(d){const h=parseInt(d[1]);return i.toUpperCase()+`

`+f+`

`+HR(o.getShaderSource(e),h)}else return f}function VR(o,e){const i=GR(e);return[`vec4 ${o}( vec4 value ) {`,`	return ${i[1]}( vec4( value.rgb * ${i[0]}, value.a ) );`,"}"].join(`
`)}const kR={[Wx]:"Linear",[qx]:"Reinhard",[Yx]:"Cineon",[tm]:"ACESFilmic",[Zx]:"AgX",[Kx]:"Neutral",[jx]:"Custom"};function XR(o,e){const i=kR[e];return i===void 0?(oe("WebGLProgram: Unsupported toneMapping:",e),"vec3 "+o+"( vec3 color ) { return LinearToneMapping( color ); }"):"vec3 "+o+"( vec3 color ) { return "+i+"ToneMapping( color ); }"}const Cu=new dt;function WR(){Pe.getLuminanceCoefficients(Cu);const o=Cu.x.toFixed(4),e=Cu.y.toFixed(4),i=Cu.z.toFixed(4);return["float luminance( const in vec3 rgb ) {",`	const vec3 weights = vec3( ${o}, ${e}, ${i} );`,"	return dot( weights, rgb );","}"].join(`
`)}function qR(o){return[o.extensionClipCullDistance?"#extension GL_ANGLE_clip_cull_distance : require":"",o.extensionMultiDraw?"#extension GL_ANGLE_multi_draw : require":""].filter(Cl).join(`
`)}function YR(o){const e=[];for(const i in o){const r=o[i];r!==!1&&e.push("#define "+i+" "+r)}return e.join(`
`)}function jR(o,e){const i={},r=o.getProgramParameter(e,o.ACTIVE_ATTRIBUTES);for(let c=0;c<r;c++){const f=o.getActiveAttrib(e,c),d=f.name;let h=1;f.type===o.FLOAT_MAT2&&(h=2),f.type===o.FLOAT_MAT3&&(h=3),f.type===o.FLOAT_MAT4&&(h=4),i[d]={type:f.type,location:o.getAttribLocation(e,d),locationSize:h}}return i}function Cl(o){return o!==""}function wx(o,e){const i=e.numSpotLightShadows+e.numSpotLightMaps-e.numSpotLightShadowsWithMaps;return o.replace(/NUM_SUN_LIGHTS/g,e.numSunLights).replace(/NUM_DIR_LIGHTS/g,e.numDirLights).replace(/NUM_SPOT_LIGHTS/g,e.numSpotLights).replace(/NUM_SPOT_LIGHT_MAPS/g,e.numSpotLightMaps).replace(/NUM_SPOT_LIGHT_COORDS/g,i).replace(/NUM_RECT_AREA_LIGHTS/g,e.numRectAreaLights).replace(/NUM_POINT_LIGHTS/g,e.numPointLights).replace(/NUM_HEMI_LIGHTS/g,e.numHemiLights).replace(/NUM_SUN_LIGHT_SHADOWS/g,e.numSunLightShadows).replace(/NUM_DIR_LIGHT_SHADOWS/g,e.numDirLightShadows).replace(/NUM_SPOT_LIGHT_SHADOWS_WITH_MAPS/g,e.numSpotLightShadowsWithMaps).replace(/NUM_SPOT_LIGHT_SHADOWS/g,e.numSpotLightShadows).replace(/NUM_POINT_LIGHT_SHADOWS/g,e.numPointLightShadows)}function Cx(o,e){return o.replace(/NUM_CLIPPING_PLANES/g,e.numClippingPlanes).replace(/UNION_CLIPPING_PLANES/g,e.numClippingPlanes-e.numClipIntersection)}const ZR=/^[ \t]*#include +<([\w\d./]+)>/gm;function Zp(o){return o.replace(ZR,QR)}const KR=new Map;function QR(o,e){let i=_e[e];if(i===void 0){const r=KR.get(e);if(r!==void 0)i=_e[r],oe('WebGLRenderer: Shader chunk "%s" has been deprecated. Use "%s" instead.',e,r);else throw new Error("THREE.WebGLProgram: Can not resolve #include <"+e+">")}return Zp(i)}const JR=/#pragma unroll_loop_start\s+for\s*\(\s*int\s+i\s*=\s*(\d+)\s*;\s*i\s*<\s*(\d+)\s*;\s*i\s*\+\+\s*\)\s*{([\s\S]+?)}\s+#pragma unroll_loop_end/g;function Nx(o){return o.replace(JR,$R)}function $R(o,e,i,r){let c="";for(let f=parseInt(e);f<parseInt(i);f++)c+=r.replace(/\[\s*i\s*\]/g,"[ "+f+" ]").replace(/UNROLLED_LOOP_INDEX/g,f);return c}function Dx(o){let e=`precision ${o.precision} float;
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
	`;return o.precision==="highp"?e+=`
#define HIGH_PRECISION`:o.precision==="mediump"?e+=`
#define MEDIUM_PRECISION`:o.precision==="lowp"&&(e+=`
#define LOW_PRECISION`),e}const tw={[Nu]:"SHADOWMAP_TYPE_PCF",[wl]:"SHADOWMAP_TYPE_VSM"};function ew(o){return tw[o.shadowMapType]||"SHADOWMAP_TYPE_BASIC"}const nw={[tr]:"ENVMAP_TYPE_CUBE",[po]:"ENVMAP_TYPE_CUBE",[ku]:"ENVMAP_TYPE_CUBE_UV"};function iw(o){return o.envMap===!1?"ENVMAP_TYPE_CUBE":nw[o.envMapMode]||"ENVMAP_TYPE_CUBE"}const aw={[po]:"ENVMAP_MODE_REFRACTION"};function sw(o){return o.envMap===!1?"ENVMAP_MODE_REFLECTION":aw[o.envMapMode]||"ENVMAP_MODE_REFLECTION"}const rw={[$p]:"ENVMAP_BLENDING_MULTIPLY",[HE]:"ENVMAP_BLENDING_MIX",[GE]:"ENVMAP_BLENDING_ADD"};function ow(o){return o.envMap===!1?"ENVMAP_BLENDING_NONE":rw[o.combine]||"ENVMAP_BLENDING_NONE"}function lw(o){const e=o.envMapCubeUVHeight;if(e===null)return null;const i=Math.log2(e)-2,r=1/e;return{texelWidth:1/(3*Math.max(Math.pow(2,i),112)),texelHeight:r,maxMip:i}}function cw(o,e,i,r){const c=o.getContext(),f=i.defines;let d=i.vertexShader,h=i.fragmentShader;const m=ew(i),p=iw(i),x=sw(i),v=ow(i),_=lw(i),M=qR(i),R=YR(f),N=c.createProgram();let y,S,O=i.glslVersion?"#version "+i.glslVersion+`
`:"";i.isRawShaderMaterial?(y=["#define SHADER_TYPE "+i.shaderType,"#define SHADER_NAME "+i.shaderName,R].filter(Cl).join(`
`),y.length>0&&(y+=`
`),S=["#define SHADER_TYPE "+i.shaderType,"#define SHADER_NAME "+i.shaderName,R].filter(Cl).join(`
`),S.length>0&&(S+=`
`)):(y=[Dx(i),"#define SHADER_TYPE "+i.shaderType,"#define SHADER_NAME "+i.shaderName,R,i.extensionClipCullDistance?"#define USE_CLIP_DISTANCE":"",i.batching?"#define USE_BATCHING":"",i.batchingColor?"#define USE_BATCHING_COLOR":"",i.instancing?"#define USE_INSTANCING":"",i.instancingColor?"#define USE_INSTANCING_COLOR":"",i.instancingMorph?"#define USE_INSTANCING_MORPH":"",i.useFog&&i.fog?"#define USE_FOG":"",i.useFog&&i.fogExp2?"#define FOG_EXP2":"",i.map?"#define USE_MAP":"",i.envMap?"#define USE_ENVMAP":"",i.envMap?"#define "+x:"",i.lightMap?"#define USE_LIGHTMAP":"",i.aoMap?"#define USE_AOMAP":"",i.bumpMap?"#define USE_BUMPMAP":"",i.normalMap?"#define USE_NORMALMAP":"",i.normalMapObjectSpace?"#define USE_NORMALMAP_OBJECTSPACE":"",i.normalMapTangentSpace?"#define USE_NORMALMAP_TANGENTSPACE":"",i.displacementMap?"#define USE_DISPLACEMENTMAP":"",i.emissiveMap?"#define USE_EMISSIVEMAP":"",i.anisotropy?"#define USE_ANISOTROPY":"",i.anisotropyMap?"#define USE_ANISOTROPYMAP":"",i.clearcoatMap?"#define USE_CLEARCOATMAP":"",i.clearcoatRoughnessMap?"#define USE_CLEARCOAT_ROUGHNESSMAP":"",i.clearcoatNormalMap?"#define USE_CLEARCOAT_NORMALMAP":"",i.iridescenceMap?"#define USE_IRIDESCENCEMAP":"",i.iridescenceThicknessMap?"#define USE_IRIDESCENCE_THICKNESSMAP":"",i.specularMap?"#define USE_SPECULARMAP":"",i.specularColorMap?"#define USE_SPECULAR_COLORMAP":"",i.specularIntensityMap?"#define USE_SPECULAR_INTENSITYMAP":"",i.roughnessMap?"#define USE_ROUGHNESSMAP":"",i.metalnessMap?"#define USE_METALNESSMAP":"",i.alphaMap?"#define USE_ALPHAMAP":"",i.alphaHash?"#define USE_ALPHAHASH":"",i.transmission?"#define USE_TRANSMISSION":"",i.transmissionMap?"#define USE_TRANSMISSIONMAP":"",i.thicknessMap?"#define USE_THICKNESSMAP":"",i.sheenColorMap?"#define USE_SHEEN_COLORMAP":"",i.sheenRoughnessMap?"#define USE_SHEEN_ROUGHNESSMAP":"",i.mapUv?"#define MAP_UV "+i.mapUv:"",i.alphaMapUv?"#define ALPHAMAP_UV "+i.alphaMapUv:"",i.lightMapUv?"#define LIGHTMAP_UV "+i.lightMapUv:"",i.aoMapUv?"#define AOMAP_UV "+i.aoMapUv:"",i.emissiveMapUv?"#define EMISSIVEMAP_UV "+i.emissiveMapUv:"",i.bumpMapUv?"#define BUMPMAP_UV "+i.bumpMapUv:"",i.normalMapUv?"#define NORMALMAP_UV "+i.normalMapUv:"",i.displacementMapUv?"#define DISPLACEMENTMAP_UV "+i.displacementMapUv:"",i.metalnessMapUv?"#define METALNESSMAP_UV "+i.metalnessMapUv:"",i.roughnessMapUv?"#define ROUGHNESSMAP_UV "+i.roughnessMapUv:"",i.anisotropyMapUv?"#define ANISOTROPYMAP_UV "+i.anisotropyMapUv:"",i.clearcoatMapUv?"#define CLEARCOATMAP_UV "+i.clearcoatMapUv:"",i.clearcoatNormalMapUv?"#define CLEARCOAT_NORMALMAP_UV "+i.clearcoatNormalMapUv:"",i.clearcoatRoughnessMapUv?"#define CLEARCOAT_ROUGHNESSMAP_UV "+i.clearcoatRoughnessMapUv:"",i.iridescenceMapUv?"#define IRIDESCENCEMAP_UV "+i.iridescenceMapUv:"",i.iridescenceThicknessMapUv?"#define IRIDESCENCE_THICKNESSMAP_UV "+i.iridescenceThicknessMapUv:"",i.sheenColorMapUv?"#define SHEEN_COLORMAP_UV "+i.sheenColorMapUv:"",i.sheenRoughnessMapUv?"#define SHEEN_ROUGHNESSMAP_UV "+i.sheenRoughnessMapUv:"",i.specularMapUv?"#define SPECULARMAP_UV "+i.specularMapUv:"",i.specularColorMapUv?"#define SPECULAR_COLORMAP_UV "+i.specularColorMapUv:"",i.specularIntensityMapUv?"#define SPECULAR_INTENSITYMAP_UV "+i.specularIntensityMapUv:"",i.transmissionMapUv?"#define TRANSMISSIONMAP_UV "+i.transmissionMapUv:"",i.thicknessMapUv?"#define THICKNESSMAP_UV "+i.thicknessMapUv:"",i.vertexTangents&&i.flatShading===!1?"#define USE_TANGENT":"",i.vertexNormals?"#define HAS_NORMAL":"",i.vertexColors?"#define USE_COLOR":"",i.vertexAlphas?"#define USE_COLOR_ALPHA":"",i.vertexUv1s?"#define USE_UV1":"",i.vertexUv2s?"#define USE_UV2":"",i.vertexUv3s?"#define USE_UV3":"",i.pointsUvs?"#define USE_POINTS_UV":"",i.flatShading?"#define FLAT_SHADED":"",i.skinning?"#define USE_SKINNING":"",i.morphTargets?"#define USE_MORPHTARGETS":"",i.morphNormals&&i.flatShading===!1?"#define USE_MORPHNORMALS":"",i.morphColors?"#define USE_MORPHCOLORS":"",i.morphTargetsCount>0?"#define MORPHTARGETS_TEXTURE_STRIDE "+i.morphTextureStride:"",i.morphTargetsCount>0?"#define MORPHTARGETS_COUNT "+i.morphTargetsCount:"",i.doubleSided?"#define DOUBLE_SIDED":"",i.flipSided?"#define FLIP_SIDED":"",i.shadowMapEnabled?"#define USE_SHADOWMAP":"",i.shadowMapEnabled?"#define "+m:"",i.sizeAttenuation?"#define USE_SIZEATTENUATION":"",i.numLightProbes>0?"#define USE_LIGHT_PROBES":"",i.logarithmicDepthBuffer?"#define USE_LOGARITHMIC_DEPTH_BUFFER":"",i.reversedDepthBuffer?"#define USE_REVERSED_DEPTH_BUFFER":"","uniform mat4 modelMatrix;","uniform mat4 modelViewMatrix;","uniform mat4 projectionMatrix;","uniform mat4 viewMatrix;","uniform mat3 normalMatrix;","uniform vec3 cameraPosition;","uniform bool isOrthographic;","#ifdef USE_INSTANCING","	attribute mat4 instanceMatrix;","#endif","#ifdef USE_INSTANCING_COLOR","	attribute vec3 instanceColor;","#endif","#ifdef USE_INSTANCING_MORPH","	uniform sampler2D morphTexture;","#endif","attribute vec3 position;","attribute vec3 normal;","attribute vec2 uv;","#ifdef USE_UV1","	attribute vec2 uv1;","#endif","#ifdef USE_UV2","	attribute vec2 uv2;","#endif","#ifdef USE_UV3","	attribute vec2 uv3;","#endif","#ifdef USE_TANGENT","	attribute vec4 tangent;","#endif","#if defined( USE_COLOR_ALPHA )","	attribute vec4 color;","#elif defined( USE_COLOR )","	attribute vec3 color;","#endif","#ifdef USE_SKINNING","	attribute vec4 skinIndex;","	attribute vec4 skinWeight;","#endif",`
`].filter(Cl).join(`
`),S=[Dx(i),"#define SHADER_TYPE "+i.shaderType,"#define SHADER_NAME "+i.shaderName,R,i.useFog&&i.fog?"#define USE_FOG":"",i.useFog&&i.fogExp2?"#define FOG_EXP2":"",i.alphaToCoverage?"#define ALPHA_TO_COVERAGE":"",i.map?"#define USE_MAP":"",i.matcap?"#define USE_MATCAP":"",i.envMap?"#define USE_ENVMAP":"",i.envMap?"#define "+p:"",i.envMap?"#define "+x:"",i.envMap?"#define "+v:"",_?"#define CUBEUV_TEXEL_WIDTH "+_.texelWidth:"",_?"#define CUBEUV_TEXEL_HEIGHT "+_.texelHeight:"",_?"#define CUBEUV_MAX_MIP "+_.maxMip+".0":"",i.lightMap?"#define USE_LIGHTMAP":"",i.aoMap?"#define USE_AOMAP":"",i.bumpMap?"#define USE_BUMPMAP":"",i.normalMap?"#define USE_NORMALMAP":"",i.normalMapObjectSpace?"#define USE_NORMALMAP_OBJECTSPACE":"",i.normalMapTangentSpace?"#define USE_NORMALMAP_TANGENTSPACE":"",i.packedNormalMap?"#define USE_PACKED_NORMALMAP":"",i.emissiveMap?"#define USE_EMISSIVEMAP":"",i.anisotropy?"#define USE_ANISOTROPY":"",i.anisotropyMap?"#define USE_ANISOTROPYMAP":"",i.clearcoat?"#define USE_CLEARCOAT":"",i.clearcoatMap?"#define USE_CLEARCOATMAP":"",i.clearcoatRoughnessMap?"#define USE_CLEARCOAT_ROUGHNESSMAP":"",i.clearcoatNormalMap?"#define USE_CLEARCOAT_NORMALMAP":"",i.dispersion?"#define USE_DISPERSION":"",i.retroreflection?"#define USE_RETROREFLECTION":"",i.iridescence?"#define USE_IRIDESCENCE":"",i.iridescenceMap?"#define USE_IRIDESCENCEMAP":"",i.iridescenceThicknessMap?"#define USE_IRIDESCENCE_THICKNESSMAP":"",i.specularMap?"#define USE_SPECULARMAP":"",i.specularColorMap?"#define USE_SPECULAR_COLORMAP":"",i.specularIntensityMap?"#define USE_SPECULAR_INTENSITYMAP":"",i.roughnessMap?"#define USE_ROUGHNESSMAP":"",i.metalnessMap?"#define USE_METALNESSMAP":"",i.alphaMap?"#define USE_ALPHAMAP":"",i.alphaTest?"#define USE_ALPHATEST":"",i.alphaHash?"#define USE_ALPHAHASH":"",i.sheen?"#define USE_SHEEN":"",i.sheenColorMap?"#define USE_SHEEN_COLORMAP":"",i.sheenRoughnessMap?"#define USE_SHEEN_ROUGHNESSMAP":"",i.transmission?"#define USE_TRANSMISSION":"",i.transmissionMap?"#define USE_TRANSMISSIONMAP":"",i.thicknessMap?"#define USE_THICKNESSMAP":"",i.vertexTangents&&i.flatShading===!1?"#define USE_TANGENT":"",i.vertexColors||i.instancingColor?"#define USE_COLOR":"",i.vertexAlphas||i.batchingColor?"#define USE_COLOR_ALPHA":"",i.vertexUv1s?"#define USE_UV1":"",i.vertexUv2s?"#define USE_UV2":"",i.vertexUv3s?"#define USE_UV3":"",i.pointsUvs?"#define USE_POINTS_UV":"",i.gradientMap?"#define USE_GRADIENTMAP":"",i.flatShading?"#define FLAT_SHADED":"",i.doubleSided?"#define DOUBLE_SIDED":"",i.flipSided?"#define FLIP_SIDED":"",i.shadowMapEnabled?"#define USE_SHADOWMAP":"",i.shadowMapEnabled?"#define "+m:"",i.premultipliedAlpha?"#define PREMULTIPLIED_ALPHA":"",i.numLightProbes>0?"#define USE_LIGHT_PROBES":"",i.numLightProbeGrids>0?"#define USE_LIGHT_PROBES_GRID":"",i.decodeVideoTexture?"#define DECODE_VIDEO_TEXTURE":"",i.decodeVideoTextureEmissive?"#define DECODE_VIDEO_TEXTURE_EMISSIVE":"",i.logarithmicDepthBuffer?"#define USE_LOGARITHMIC_DEPTH_BUFFER":"",i.reversedDepthBuffer?"#define USE_REVERSED_DEPTH_BUFFER":"","uniform mat4 viewMatrix;","uniform vec3 cameraPosition;","uniform bool isOrthographic;",i.toneMapping!==ca?"#define TONE_MAPPING":"",i.toneMapping!==ca?_e.tonemapping_pars_fragment:"",i.toneMapping!==ca?XR("toneMapping",i.toneMapping):"",i.dithering?"#define DITHERING":"",i.opaque?"#define OPAQUE":"",_e.colorspace_pars_fragment,VR("linearToOutputTexel",i.outputColorSpace),WR(),i.useDepthPacking?"#define DEPTH_PACKING "+i.depthPacking:"",`
`].filter(Cl).join(`
`)),d=Zp(d),d=wx(d,i),d=Cx(d,i),h=Zp(h),h=wx(h,i),h=Cx(h,i),d=Nx(d),h=Nx(h),i.isRawShaderMaterial!==!0&&(O=`#version 300 es
`,y=[M,"#define attribute in","#define varying out","#define texture2D texture"].join(`
`)+`
`+y,S=["#define varying in",i.glslVersion===H_?"":"layout(location = 0) out highp vec4 pc_fragColor;",i.glslVersion===H_?"":"#define gl_FragColor pc_fragColor","#define gl_FragDepthEXT gl_FragDepth","#define texture2D texture","#define textureCube texture","#define texture2DProj textureProj","#define texture2DLodEXT textureLod","#define texture2DProjLodEXT textureProjLod","#define textureCubeLodEXT textureLod","#define texture2DGradEXT textureGrad","#define texture2DProjGradEXT textureProjGrad","#define textureCubeGradEXT textureGrad"].join(`
`)+`
`+S);const k=O+y+d,C=O+S+h,D=Tx(c,c.VERTEX_SHADER,k),U=Tx(c,c.FRAGMENT_SHADER,C);c.attachShader(N,D),c.attachShader(N,U),i.index0AttributeName!==void 0?c.bindAttribLocation(N,0,i.index0AttributeName):i.hasPositionAttribute===!0&&c.bindAttribLocation(N,0,"position"),c.linkProgram(N);function B(j){if(o.debug.checkShaderErrors){const J=c.getProgramInfoLog(N)||"",P=c.getShaderInfoLog(D)||"",q=c.getShaderInfoLog(U)||"",$=J.trim(),Y=P.trim(),Q=q.trim();let pt=!0,ot=!0;if(c.getProgramParameter(N,c.LINK_STATUS)===!1)if(pt=!1,typeof o.debug.onShaderError=="function")o.debug.onShaderError(c,N,D,U);else{const vt=Rx(c,D,"vertex"),Tt=Rx(c,U,"fragment");ze("WebGLProgram: Shader Error "+c.getError()+" - VALIDATE_STATUS "+c.getProgramParameter(N,c.VALIDATE_STATUS)+`

Material Name: `+j.name+`
Material Type: `+j.type+`

Program Info Log: `+$+`
`+vt+`
`+Tt)}else $!==""?oe("WebGLProgram: Program Info Log:",$):(Y===""||Q==="")&&(ot=!1);ot&&(j.diagnostics={runnable:pt,programLog:$,vertexShader:{log:Y,prefix:y},fragmentShader:{log:Q,prefix:S}})}c.deleteShader(D),c.deleteShader(U),b=new Pu(c,N),L=jR(c,N)}let b;this.getUniforms=function(){return b===void 0&&B(this),b};let L;this.getAttributes=function(){return L===void 0&&B(this),L};let X=i.rendererExtensionParallelShaderCompile===!1;return this.isReady=function(){return X===!1&&(X=c.getProgramParameter(N,zR)),X},this.destroy=function(){r.releaseStatesOfProgram(this),c.deleteProgram(N),this.program=void 0},this.type=i.shaderType,this.name=i.shaderName,this.id=FR++,this.cacheKey=e,this.usedTimes=1,this.program=N,this.vertexShader=D,this.fragmentShader=U,this}let uw=0;class fw{constructor(){this.shaderCache=new Map,this.materialCache=new Map}update(e,i,r){const c=this._getShaderCacheForMaterial(e);return c.has(i)===!1&&(c.add(i),i.usedTimes++),c.has(r)===!1&&(c.add(r),r.usedTimes++),this}remove(e){const i=this.materialCache.get(e);for(const r of i)r.usedTimes--,r.usedTimes===0&&this.shaderCache.delete(r.code);return this.materialCache.delete(e),this}getVertexShaderStage(e){return this._getShaderStage(e.vertexShader)}getFragmentShaderStage(e){return this._getShaderStage(e.fragmentShader)}dispose(){this.shaderCache.clear(),this.materialCache.clear()}_getShaderCacheForMaterial(e){const i=this.materialCache;let r=i.get(e);return r===void 0&&(r=new Set,i.set(e,r)),r}_getShaderStage(e){const i=this.shaderCache;let r=i.get(e);return r===void 0&&(r=new dw(e),i.set(e,r)),r}}class dw{constructor(e){this.id=uw++,this.code=e,this.usedTimes=0}}function hw(o){return o===er||o===Iu||o===Bu}function pw(o,e,i,r,c,f){const d=new rS,h=new fw,m=new Set,p=[],x=new Map,v=r.logarithmicDepthBuffer;let _=r.precision;const M={MeshDepthMaterial:"depth",MeshDistanceMaterial:"distance",MeshNormalMaterial:"normal",MeshBasicMaterial:"basic",MeshLambertMaterial:"lambert",MeshPhongMaterial:"phong",MeshToonMaterial:"toon",MeshStandardMaterial:"physical",MeshPhysicalMaterial:"physical",MeshMatcapMaterial:"matcap",LineBasicMaterial:"basic",LineDashedMaterial:"dashed",PointsMaterial:"points",ShadowMaterial:"shadow",SpriteMaterial:"sprite"};function R(b){return m.add(b),b===0?"uv":`uv${b}`}function N(b,L,X,j,J,P){const q=j.fog,$=J.geometry,Y=b.isMeshStandardMaterial||b.isMeshLambertMaterial||b.isMeshPhongMaterial?j.environment:null,Q=b.isMeshStandardMaterial||b.isMeshLambertMaterial&&!b.envMap||b.isMeshPhongMaterial&&!b.envMap,pt=e.get(b.envMap||Y,Q),ot=pt&&pt.mapping===ku?pt.image.height:null,vt=M[b.type];b.precision!==null&&(_=r.getMaxPrecision(b.precision),_!==b.precision&&oe("WebGLProgram.getParameters:",b.precision,"not supported, using",_,"instead."));const Tt=$.morphAttributes.position||$.morphAttributes.normal||$.morphAttributes.color,Zt=Tt!==void 0?Tt.length:0;let Yt=0;$.morphAttributes.position!==void 0&&(Yt=1),$.morphAttributes.normal!==void 0&&(Yt=2),$.morphAttributes.color!==void 0&&(Yt=3);let H,F,gt,G;if(vt){const Le=oa[vt];H=Le.vertexShader,F=Le.fragmentShader}else{H=b.vertexShader,F=b.fragmentShader;const Le=h.getVertexShaderStage(b),ue=h.getFragmentShaderStage(b);h.update(b,Le,ue),gt=Le.id,G=ue.id}const st=o.getRenderTarget(),At=o.state.buffers.depth.getReversed(),Dt=J.isInstancedMesh===!0,xt=J.isBatchedMesh===!0,Ct=!!b.map,Fe=!!b.matcap,le=!!pt,pe=!!b.aoMap,ve=!!b.lightMap,te=!!b.bumpMap&&b.wireframe===!1,ne=!!b.normalMap,Be=!!b.displacementMap,sn=!!b.emissiveMap,Ue=!!b.metalnessMap,je=!!b.roughnessMap,Z=b.anisotropy>0,Je=b.clearcoat>0,Me=b.dispersion>0,I=b.retroreflectivity>0,E=b.iridescence>0,at=b.sheen>0,lt=b.transmission>0,et=Z&&!!b.anisotropyMap,St=Je&&!!b.clearcoatMap,wt=Je&&!!b.clearcoatNormalMap,_t=Je&&!!b.clearcoatRoughnessMap,yt=E&&!!b.iridescenceMap,Nt=E&&!!b.iridescenceThicknessMap,Jt=at&&!!b.sheenColorMap,Bt=at&&!!b.sheenRoughnessMap,It=!!b.specularMap,kt=!!b.specularColorMap,ie=!!b.specularIntensityMap,ce=lt&&!!b.transmissionMap,K=lt&&!!b.thicknessMap,Ut=!!b.gradientMap,bt=!!b.alphaMap,Lt=b.alphaTest>0,Vt=!!b.alphaHash,Rt=!!b.extensions;let $t=ca;b.toneMapped&&(st===null||st.isXRRenderTarget===!0)&&($t=o.toneMapping);const Gt={shaderID:vt,shaderType:b.type,shaderName:b.name,vertexShader:H,fragmentShader:F,defines:b.defines,customVertexShaderID:gt,customFragmentShaderID:G,isRawShaderMaterial:b.isRawShaderMaterial===!0,glslVersion:b.glslVersion,precision:_,batching:xt,batchingColor:xt&&J._colorsTexture!==null,instancing:Dt,instancingColor:Dt&&J.instanceColor!==null,instancingMorph:Dt&&J.morphTexture!==null,outputColorSpace:st===null?o.outputColorSpace:st.isXRRenderTarget===!0?st.texture.colorSpace:Pe.workingColorSpace,alphaToCoverage:!!b.alphaToCoverage,map:Ct,matcap:Fe,envMap:le,envMapMode:le&&pt.mapping,envMapCubeUVHeight:ot,aoMap:pe,lightMap:ve,bumpMap:te,normalMap:ne,displacementMap:Be,emissiveMap:sn,normalMapObjectSpace:ne&&b.normalMapType===XE,normalMapTangentSpace:ne&&b.normalMapType===zu,packedNormalMap:ne&&b.normalMapType===zu&&hw(b.normalMap.format),metalnessMap:Ue,roughnessMap:je,anisotropy:Z,anisotropyMap:et,clearcoat:Je,clearcoatMap:St,clearcoatNormalMap:wt,clearcoatRoughnessMap:_t,dispersion:Me,retroreflection:I,iridescence:E,iridescenceMap:yt,iridescenceThicknessMap:Nt,sheen:at,sheenColorMap:Jt,sheenRoughnessMap:Bt,specularMap:It,specularColorMap:kt,specularIntensityMap:ie,transmission:lt,transmissionMap:ce,thicknessMap:K,gradientMap:Ut,opaque:b.transparent===!1&&b.blending===Dl&&b.alphaToCoverage===!1,alphaMap:bt,alphaTest:Lt,alphaHash:Vt,combine:b.combine,mapUv:Ct&&R(b.map.channel),aoMapUv:pe&&R(b.aoMap.channel),lightMapUv:ve&&R(b.lightMap.channel),bumpMapUv:te&&R(b.bumpMap.channel),normalMapUv:ne&&R(b.normalMap.channel),displacementMapUv:Be&&R(b.displacementMap.channel),emissiveMapUv:sn&&R(b.emissiveMap.channel),metalnessMapUv:Ue&&R(b.metalnessMap.channel),roughnessMapUv:je&&R(b.roughnessMap.channel),anisotropyMapUv:et&&R(b.anisotropyMap.channel),clearcoatMapUv:St&&R(b.clearcoatMap.channel),clearcoatNormalMapUv:wt&&R(b.clearcoatNormalMap.channel),clearcoatRoughnessMapUv:_t&&R(b.clearcoatRoughnessMap.channel),iridescenceMapUv:yt&&R(b.iridescenceMap.channel),iridescenceThicknessMapUv:Nt&&R(b.iridescenceThicknessMap.channel),sheenColorMapUv:Jt&&R(b.sheenColorMap.channel),sheenRoughnessMapUv:Bt&&R(b.sheenRoughnessMap.channel),specularMapUv:It&&R(b.specularMap.channel),specularColorMapUv:kt&&R(b.specularColorMap.channel),specularIntensityMapUv:ie&&R(b.specularIntensityMap.channel),transmissionMapUv:ce&&R(b.transmissionMap.channel),thicknessMapUv:K&&R(b.thicknessMap.channel),alphaMapUv:bt&&R(b.alphaMap.channel),vertexTangents:!!$.attributes.tangent&&(ne||Z),vertexNormals:!!$.attributes.normal,vertexColors:b.vertexColors,vertexAlphas:b.vertexColors===!0&&!!$.attributes.color&&$.attributes.color.itemSize===4,pointsUvs:J.isPoints===!0&&!!$.attributes.uv&&(Ct||bt),fog:!!q,useFog:b.fog===!0,fogExp2:!!q&&q.isFogExp2,flatShading:b.wireframe===!1&&(b.flatShading===!0||$.attributes.normal===void 0&&ne===!1&&(b.isMeshLambertMaterial||b.isMeshPhongMaterial||b.isMeshStandardMaterial||b.isMeshPhysicalMaterial)),sizeAttenuation:b.sizeAttenuation===!0,logarithmicDepthBuffer:v,reversedDepthBuffer:At,skinning:J.isSkinnedMesh===!0,hasPositionAttribute:$.attributes.position!==void 0,morphTargets:$.morphAttributes.position!==void 0,morphNormals:$.morphAttributes.normal!==void 0,morphColors:$.morphAttributes.color!==void 0,morphTargetsCount:Zt,morphTextureStride:Yt,numSunLights:L.sun.length,numDirLights:L.directional.length,numPointLights:L.point.length,numSpotLights:L.spot.length,numSpotLightMaps:L.spotLightMap.length,numRectAreaLights:L.rectArea.length,numHemiLights:L.hemi.length,numSunLightShadows:L.sunShadowMap.length,numDirLightShadows:L.directionalShadowMap.length,numPointLightShadows:L.pointShadowMap.length,numSpotLightShadows:L.spotShadowMap.length,numSpotLightShadowsWithMaps:L.numSpotLightShadowsWithMaps,numLightProbes:L.numLightProbes,numLightProbeGrids:P.length,numClippingPlanes:f.numPlanes,numClipIntersection:f.numIntersection,dithering:b.dithering,shadowMapEnabled:o.shadowMap.enabled&&X.length>0,shadowMapType:o.shadowMap.type,toneMapping:$t,decodeVideoTexture:Ct&&b.map.isVideoTexture===!0&&Pe.getTransfer(b.map.colorSpace)===Ye,decodeVideoTextureEmissive:sn&&b.emissiveMap.isVideoTexture===!0&&Pe.getTransfer(b.emissiveMap.colorSpace)===Ye,premultipliedAlpha:b.premultipliedAlpha,doubleSided:b.side===Bi,flipSided:b.side===qn,useDepthPacking:b.depthPacking>=0,depthPacking:b.depthPacking||0,index0AttributeName:b.index0AttributeName,extensionClipCullDistance:Rt&&b.extensions.clipCullDistance===!0&&i.has("WEBGL_clip_cull_distance"),extensionMultiDraw:(Rt&&b.extensions.multiDraw===!0||xt)&&i.has("WEBGL_multi_draw"),rendererExtensionParallelShaderCompile:i.has("KHR_parallel_shader_compile"),customProgramCacheKey:b.customProgramCacheKey()};return Gt.vertexUv1s=m.has(1),Gt.vertexUv2s=m.has(2),Gt.vertexUv3s=m.has(3),m.clear(),Gt}function y(b){const L=[];if(b.shaderID?L.push(b.shaderID):(L.push(b.customVertexShaderID),L.push(b.customFragmentShaderID)),b.defines!==void 0)for(const X in b.defines)L.push(X),L.push(b.defines[X]);return b.isRawShaderMaterial===!1&&(S(L,b),O(L,b),L.push(o.outputColorSpace)),L.push(b.customProgramCacheKey),L.join()}function S(b,L){b.push(L.precision),b.push(L.outputColorSpace),b.push(L.envMapMode),b.push(L.envMapCubeUVHeight),b.push(L.mapUv),b.push(L.alphaMapUv),b.push(L.lightMapUv),b.push(L.aoMapUv),b.push(L.bumpMapUv),b.push(L.normalMapUv),b.push(L.displacementMapUv),b.push(L.emissiveMapUv),b.push(L.metalnessMapUv),b.push(L.roughnessMapUv),b.push(L.anisotropyMapUv),b.push(L.clearcoatMapUv),b.push(L.clearcoatNormalMapUv),b.push(L.clearcoatRoughnessMapUv),b.push(L.iridescenceMapUv),b.push(L.iridescenceThicknessMapUv),b.push(L.sheenColorMapUv),b.push(L.sheenRoughnessMapUv),b.push(L.specularMapUv),b.push(L.specularColorMapUv),b.push(L.specularIntensityMapUv),b.push(L.transmissionMapUv),b.push(L.thicknessMapUv),b.push(L.combine),b.push(L.fogExp2),b.push(L.sizeAttenuation),b.push(L.morphTargetsCount),b.push(L.morphAttributeCount),b.push(L.numSunLights),b.push(L.numDirLights),b.push(L.numPointLights),b.push(L.numSpotLights),b.push(L.numSpotLightMaps),b.push(L.numHemiLights),b.push(L.numRectAreaLights),b.push(L.numSunLightShadows),b.push(L.numDirLightShadows),b.push(L.numPointLightShadows),b.push(L.numSpotLightShadows),b.push(L.numSpotLightShadowsWithMaps),b.push(L.numLightProbes),b.push(L.shadowMapType),b.push(L.toneMapping),b.push(L.numClippingPlanes),b.push(L.numClipIntersection),b.push(L.depthPacking)}function O(b,L){d.disableAll(),L.instancing&&d.enable(0),L.instancingColor&&d.enable(1),L.instancingMorph&&d.enable(2),L.matcap&&d.enable(3),L.envMap&&d.enable(4),L.normalMapObjectSpace&&d.enable(5),L.normalMapTangentSpace&&d.enable(6),L.clearcoat&&d.enable(7),L.iridescence&&d.enable(8),L.alphaTest&&d.enable(9),L.vertexColors&&d.enable(10),L.vertexAlphas&&d.enable(11),L.vertexUv1s&&d.enable(12),L.vertexUv2s&&d.enable(13),L.vertexUv3s&&d.enable(14),L.vertexTangents&&d.enable(15),L.anisotropy&&d.enable(16),L.alphaHash&&d.enable(17),L.batching&&d.enable(18),L.dispersion&&d.enable(19),L.retroreflection&&d.enable(24),L.batchingColor&&d.enable(20),L.gradientMap&&d.enable(21),L.packedNormalMap&&d.enable(22),L.vertexNormals&&d.enable(23),b.push(d.mask),d.disableAll(),L.fog&&d.enable(0),L.useFog&&d.enable(1),L.flatShading&&d.enable(2),L.logarithmicDepthBuffer&&d.enable(3),L.reversedDepthBuffer&&d.enable(4),L.skinning&&d.enable(5),L.morphTargets&&d.enable(6),L.morphNormals&&d.enable(7),L.morphColors&&d.enable(8),L.premultipliedAlpha&&d.enable(9),L.shadowMapEnabled&&d.enable(10),L.doubleSided&&d.enable(11),L.flipSided&&d.enable(12),L.useDepthPacking&&d.enable(13),L.dithering&&d.enable(14),L.transmission&&d.enable(15),L.sheen&&d.enable(16),L.opaque&&d.enable(17),L.pointsUvs&&d.enable(18),L.decodeVideoTexture&&d.enable(19),L.decodeVideoTextureEmissive&&d.enable(20),L.alphaToCoverage&&d.enable(21),L.numLightProbeGrids>0&&d.enable(22),L.hasPositionAttribute&&d.enable(23),b.push(d.mask)}function k(b){const L=M[b.type];let X;if(L){const j=oa[L];X=Db.clone(j.uniforms)}else X=b.uniforms;return X}function C(b,L){let X=x.get(L);return X!==void 0?++X.usedTimes:(X=new cw(o,L,b,c),p.push(X),x.set(L,X)),X}function D(b){if(--b.usedTimes===0){const L=p.indexOf(b);p[L]=p[p.length-1],p.pop(),x.delete(b.cacheKey),b.destroy()}}function U(b){h.remove(b)}function B(){h.dispose()}return{getParameters:N,getProgramCacheKey:y,getUniforms:k,acquireProgram:C,releaseProgram:D,releaseShaderCache:U,programs:p,dispose:B}}function mw(){let o=new WeakMap;function e(d){return o.has(d)}function i(d){let h=o.get(d);return h===void 0&&(h={},o.set(d,h)),h}function r(d){o.delete(d)}function c(d,h,m){o.get(d)[h]=m}function f(){o=new WeakMap}return{has:e,get:i,remove:r,update:c,dispose:f}}function gw(o,e){return o.groupOrder!==e.groupOrder?o.groupOrder-e.groupOrder:o.renderOrder!==e.renderOrder?o.renderOrder-e.renderOrder:o.material.id!==e.material.id?o.material.id-e.material.id:o.materialVariant!==e.materialVariant?o.materialVariant-e.materialVariant:o.z!==e.z?o.z-e.z:o.id-e.id}function Ux(o,e){return o.groupOrder!==e.groupOrder?o.groupOrder-e.groupOrder:o.renderOrder!==e.renderOrder?o.renderOrder-e.renderOrder:o.z!==e.z?e.z-o.z:o.id-e.id}function Lx(){const o=[];let e=0;const i=[],r=[],c=[];function f(){e=0,i.length=0,r.length=0,c.length=0}function d(_){let M=0;return _.isInstancedMesh&&(M+=2),_.isSkinnedMesh&&(M+=1),M}function h(_,M,R,N,y,S){let O=o[e];return O===void 0?(O={id:_.id,object:_,geometry:M,material:R,materialVariant:d(_),groupOrder:N,renderOrder:_.renderOrder,z:y,group:S},o[e]=O):(O.id=_.id,O.object=_,O.geometry=M,O.material=R,O.materialVariant=d(_),O.groupOrder=N,O.renderOrder=_.renderOrder,O.z=y,O.group=S),e++,O}function m(_,M,R,N,y,S,O){O.reversedDepth===!0&&(y=-y);const k=h(_,M,R,N,y,S);R.transmission>0?r.push(k):R.transparent===!0?c.push(k):i.push(k)}function p(_,M,R,N,y,S){const O=h(_,M,R,N,y,S);R.transmission>0?r.unshift(O):R.transparent===!0?c.unshift(O):i.unshift(O)}function x(_,M){i.length>1&&i.sort(_||gw),r.length>1&&r.sort(M||Ux),c.length>1&&c.sort(M||Ux)}function v(){for(let _=e,M=o.length;_<M;_++){const R=o[_];if(R.id===null)break;R.id=null,R.object=null,R.geometry=null,R.material=null,R.group=null}}return{opaque:i,transmissive:r,transparent:c,init:f,push:m,unshift:p,finish:v,sort:x}}function vw(){let o=new WeakMap;function e(r,c){const f=o.get(r);let d;return f===void 0?(d=new Lx,o.set(r,[d])):c>=f.length?(d=new Lx,f.push(d)):d=f[c],d}function i(){o=new WeakMap}return{get:e,dispose:i}}function _w(){const o={};return{get:function(e){if(o[e.id]!==void 0)return o[e.id];let i;switch(e.type){case"SunLight":case"DirectionalLight":i={direction:new dt,color:new he};break;case"SpotLight":i={position:new dt,direction:new dt,color:new he,distance:0,coneCos:0,penumbraCos:0,decay:0};break;case"PointLight":i={position:new dt,color:new he,distance:0,decay:0};break;case"HemisphereLight":i={direction:new dt,skyColor:new he,groundColor:new he};break;case"RectAreaLight":i={color:new he,position:new dt,halfWidth:new dt,halfHeight:new dt};break}return o[e.id]=i,i}}}function xw(){const o={};return{get:function(e){if(o[e.id]!==void 0)return o[e.id];let i;switch(e.type){case"SunLight":case"DirectionalLight":i={shadowIntensity:1,shadowBias:0,shadowNormalBias:0,shadowRadius:1,shadowMapSize:new Re};break;case"SpotLight":i={shadowIntensity:1,shadowBias:0,shadowNormalBias:0,shadowRadius:1,shadowMapSize:new Re};break;case"PointLight":i={shadowIntensity:1,shadowBias:0,shadowNormalBias:0,shadowRadius:1,shadowMapSize:new Re,shadowCameraNear:1,shadowCameraFar:1e3};break}return o[e.id]=i,i}}}let Sw=0;function yw(o,e){return(e.castShadow?2:0)-(o.castShadow?2:0)+(e.map?1:0)-(o.map?1:0)}function Mw(o){const e=new _w,i=xw(),r={version:0,hash:{sunLength:-1,directionalLength:-1,pointLength:-1,spotLength:-1,rectAreaLength:-1,hemiLength:-1,numSunShadows:-1,numDirectionalShadows:-1,numPointShadows:-1,numSpotShadows:-1,numSpotMaps:-1,numLightProbes:-1},ambient:[0,0,0],probe:[],sun:[],sunShadow:[],sunShadowMap:[],sunShadowMatrix:[],sunShadowCascade:[],directional:[],directionalShadow:[],directionalShadowMap:[],directionalShadowMatrix:[],spot:[],spotLightMap:[],spotShadow:[],spotShadowMap:[],spotLightMatrix:[],rectArea:[],rectAreaLTC1:null,rectAreaLTC2:null,point:[],pointShadow:[],pointShadowMap:[],pointShadowMatrix:[],hemi:[],numSpotLightShadowsWithMaps:0,numLightProbes:0};for(let p=0;p<9;p++)r.probe.push(new dt);const c=new dt,f=new nn,d=new nn;function h(p){let x=0,v=0,_=0;for(let J=0;J<9;J++)r.probe[J].set(0,0,0);let M=0,R=0,N=0,y=0,S=0,O=0,k=0,C=0,D=0,U=0,B=0,b=0,L=0,X=0;p.sort(yw);for(let J=0,P=p.length;J<P;J++){const q=p[J],$=q.color,Y=q.intensity,Q=q.distance;let pt=null;if(q.shadow&&q.shadow.map&&(q.shadow.map.texture.format===er?pt=q.shadow.map.texture:pt=q.shadow.map.depthTexture||q.shadow.map.texture),q.isAmbientLight)x+=$.r*Y,v+=$.g*Y,_+=$.b*Y;else if(q.isLightProbe){for(let ot=0;ot<9;ot++)r.probe[ot].addScaledVector(q.sh.coefficients[ot],Y);X++}else if(q.isSunLight){const ot=e.get(q);if(ot.color.copy(q.color).multiplyScalar(q.intensity),q.castShadow){const vt=q.shadow,Tt=i.get(q);Tt.shadowIntensity=vt.intensity,Tt.shadowBias=vt.bias,Tt.shadowNormalBias=vt.normalBias,Tt.shadowRadius=vt.radius,Tt.shadowMapSize.copy(vt.mapSize).multiply(vt.getFrameExtents()),r.sunShadow[R]=Tt,r.sunShadowMap[R]=pt;const Zt=vt.getViewportCount();for(let Yt=0;Yt<Zt;Yt++)r.sunShadowMatrix[N+Yt]=vt.getMatrix(Yt),r.sunShadowCascade[N+Yt]=vt._cascadeData[Yt];N+=Zt,R++}r.sun[M]=ot,M++}else if(q.isDirectionalLight){const ot=e.get(q);if(ot.color.copy(q.color).multiplyScalar(q.intensity),q.castShadow){const vt=q.shadow,Tt=i.get(q);Tt.shadowIntensity=vt.intensity,Tt.shadowBias=vt.bias,Tt.shadowNormalBias=vt.normalBias,Tt.shadowRadius=vt.radius,Tt.shadowMapSize=vt.mapSize,r.directionalShadow[y]=Tt,r.directionalShadowMap[y]=pt,r.directionalShadowMatrix[y]=q.shadow.matrix,D++}r.directional[y]=ot,y++}else if(q.isSpotLight){const ot=e.get(q);ot.position.setFromMatrixPosition(q.matrixWorld),ot.color.copy($).multiplyScalar(Y),ot.distance=Q,ot.coneCos=Math.cos(q.angle),ot.penumbraCos=Math.cos(q.angle*(1-q.penumbra)),ot.decay=q.decay,r.spot[O]=ot;const vt=q.shadow;if(q.map&&(r.spotLightMap[b]=q.map,b++,vt.updateMatrices(q),q.castShadow&&L++),r.spotLightMatrix[O]=vt.matrix,q.castShadow){const Tt=i.get(q);Tt.shadowIntensity=vt.intensity,Tt.shadowBias=vt.bias,Tt.shadowNormalBias=vt.normalBias,Tt.shadowRadius=vt.radius,Tt.shadowMapSize=vt.mapSize,r.spotShadow[O]=Tt,r.spotShadowMap[O]=pt,B++}O++}else if(q.isRectAreaLight){const ot=e.get(q);ot.color.copy($).multiplyScalar(Y),ot.halfWidth.set(q.width*.5,0,0),ot.halfHeight.set(0,q.height*.5,0),r.rectArea[k]=ot,k++}else if(q.isPointLight){const ot=e.get(q);if(ot.color.copy(q.color).multiplyScalar(q.intensity),ot.distance=q.distance,ot.decay=q.decay,q.castShadow){const vt=q.shadow,Tt=i.get(q);Tt.shadowIntensity=vt.intensity,Tt.shadowBias=vt.bias,Tt.shadowNormalBias=vt.normalBias,Tt.shadowRadius=vt.radius,Tt.shadowMapSize=vt.mapSize,Tt.shadowCameraNear=vt.camera.near,Tt.shadowCameraFar=vt.camera.far,r.pointShadow[S]=Tt,r.pointShadowMap[S]=pt,r.pointShadowMatrix[S]=q.shadow.matrix,U++}r.point[S]=ot,S++}else if(q.isHemisphereLight){const ot=e.get(q);ot.skyColor.copy(q.color).multiplyScalar(Y),ot.groundColor.copy(q.groundColor).multiplyScalar(Y),r.hemi[C]=ot,C++}}k>0&&(o.has("OES_texture_float_linear")===!0?(r.rectAreaLTC1=Ht.LTC_FLOAT_1,r.rectAreaLTC2=Ht.LTC_FLOAT_2):(r.rectAreaLTC1=Ht.LTC_HALF_1,r.rectAreaLTC2=Ht.LTC_HALF_2)),r.ambient[0]=x,r.ambient[1]=v,r.ambient[2]=_;const j=r.hash;(j.sunLength!==M||j.directionalLength!==y||j.pointLength!==S||j.spotLength!==O||j.rectAreaLength!==k||j.hemiLength!==C||j.numSunShadows!==R||j.numDirectionalShadows!==D||j.numPointShadows!==U||j.numSpotShadows!==B||j.numSpotMaps!==b||j.numLightProbes!==X)&&(r.sun.length=M,r.directional.length=y,r.spot.length=O,r.rectArea.length=k,r.point.length=S,r.hemi.length=C,r.sunShadow.length=R,r.sunShadowMap.length=R,r.sunShadowMatrix.length=N,r.sunShadowCascade.length=N,r.directionalShadow.length=D,r.directionalShadowMap.length=D,r.directionalShadowMatrix.length=D,r.pointShadow.length=U,r.pointShadowMap.length=U,r.pointShadowMatrix.length=U,r.spotShadow.length=B,r.spotShadowMap.length=B,r.spotLightMatrix.length=B+b-L,r.spotLightMap.length=b,r.numSpotLightShadowsWithMaps=L,r.numLightProbes=X,j.sunLength=M,j.directionalLength=y,j.pointLength=S,j.spotLength=O,j.rectAreaLength=k,j.hemiLength=C,j.numSunShadows=R,j.numDirectionalShadows=D,j.numPointShadows=U,j.numSpotShadows=B,j.numSpotMaps=b,j.numLightProbes=X,r.version=Sw++)}function m(p,x){let v=0,_=0,M=0,R=0,N=0,y=0;const S=x.matrixWorldInverse;for(let O=0,k=p.length;O<k;O++){const C=p[O];if(C.isSunLight){const D=r.sun[v];D.direction.setFromMatrixPosition(C.matrixWorld),D.direction.transformDirection(S),v++}else if(C.isDirectionalLight){const D=r.directional[_];D.direction.setFromMatrixPosition(C.matrixWorld),c.setFromMatrixPosition(C.target.matrixWorld),D.direction.sub(c),D.direction.transformDirection(S),_++}else if(C.isSpotLight){const D=r.spot[R];D.position.setFromMatrixPosition(C.matrixWorld),D.position.applyMatrix4(S),D.direction.setFromMatrixPosition(C.matrixWorld),c.setFromMatrixPosition(C.target.matrixWorld),D.direction.sub(c),D.direction.transformDirection(S),R++}else if(C.isRectAreaLight){const D=r.rectArea[N];D.position.setFromMatrixPosition(C.matrixWorld),D.position.applyMatrix4(S),d.identity(),f.copy(C.matrixWorld),f.premultiply(S),d.extractRotation(f),D.halfWidth.set(C.width*.5,0,0),D.halfHeight.set(0,C.height*.5,0),D.halfWidth.applyMatrix4(d),D.halfHeight.applyMatrix4(d),N++}else if(C.isPointLight){const D=r.point[M];D.position.setFromMatrixPosition(C.matrixWorld),D.position.applyMatrix4(S),M++}else if(C.isHemisphereLight){const D=r.hemi[y];D.direction.setFromMatrixPosition(C.matrixWorld),D.direction.transformDirection(S),y++}}}return{setup:h,setupView:m,state:r}}function Ox(o){const e=new Mw(o),i=[],r=[],c=[];function f(_){v.camera=_,i.length=0,r.length=0,c.length=0}function d(_){i.push(_)}function h(_){r.push(_)}function m(_){c.push(_)}function p(){e.setup(i)}function x(_){e.setupView(i,_)}const v={lightsArray:i,shadowsArray:r,lightProbeGridArray:c,camera:null,lights:e,transmissionRenderTarget:{},textureUnits:0};return{init:f,state:v,setupLights:p,setupLightsView:x,pushLight:d,pushShadow:h,pushLightProbeGrid:m}}function Ew(o){let e=new WeakMap;function i(c,f=0){const d=e.get(c);let h;return d===void 0?(h=new Ox(o),e.set(c,[h])):f>=d.length?(h=new Ox(o),d.push(h)):h=d[f],h}function r(){e=new WeakMap}return{get:i,dispose:r}}const bw=`void main() {
	gl_Position = vec4( position, 1.0 );
}`,Tw=`uniform sampler2D shadow_pass;
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
}`,Aw=[new dt(1,0,0),new dt(-1,0,0),new dt(0,1,0),new dt(0,-1,0),new dt(0,0,1),new dt(0,0,-1)],Rw=[new dt(0,-1,0),new dt(0,-1,0),new dt(0,0,1),new dt(0,0,-1),new dt(0,-1,0),new dt(0,-1,0)],Px=new nn,Al=new dt,sp=new dt;function ww(o,e,i){let r=new hm;const c=new Re,f=new Re,d=new on,h=new Ib,m=new Bb,p={},x=i.maxTextureSize,v={[$s]:qn,[qn]:$s,[Bi]:Bi},_=new ha({defines:{VSM_SAMPLES:8},uniforms:{shadow_pass:{value:null},resolution:{value:new Re},radius:{value:4}},vertexShader:bw,fragmentShader:Tw}),M=_.clone();M.defines.HORIZONTAL_PASS=1;const R=new ki;R.setAttribute("position",new ua(new Float32Array([-1,-1,.5,3,-1,.5,-1,3,.5]),3));const N=new _n(R,_),y=this;this.enabled=!1,this.autoUpdate=!0,this.needsUpdate=!1,this.type=Nu;let S=this.type;this.render=function(U,B,b){if(y.enabled===!1||y.autoUpdate===!1&&y.needsUpdate===!1||U.length===0)return;this.type===Vx&&(oe("WebGLShadowMap: PCFSoftShadowMap has been removed. Using PCFShadowMap instead."),this.type=Nu);const L=o.getRenderTarget(),X=o.getActiveCubeFace(),j=o.getActiveMipmapLevel(),J=o.state;J.setBlending(Ia),J.buffers.depth.getReversed()===!0?J.buffers.color.setClear(0,0,0,0):J.buffers.color.setClear(1,1,1,1),J.buffers.depth.setTest(!0),J.setScissorTest(!1);const P=S!==this.type;P&&B.traverse(function(q){q.material&&(Array.isArray(q.material)?q.material.forEach($=>$.needsUpdate=!0):q.material.needsUpdate=!0)});for(let q=0,$=U.length;q<$;q++){const Y=U[q],Q=Y.shadow;if(Q===void 0){oe("WebGLShadowMap:",Y,"has no shadow.");continue}if(Q.autoUpdate===!1&&Q.needsUpdate===!1)continue;c.copy(Q.mapSize);const pt=Q.getFrameExtents();c.multiply(pt),f.copy(Q.mapSize),(c.x>x||c.y>x)&&(c.x>x&&(f.x=Math.floor(x/pt.x),c.x=f.x*pt.x,Q.mapSize.x=f.x),c.y>x&&(f.y=Math.floor(x/pt.y),c.y=f.y*pt.y,Q.mapSize.y=f.y));const ot=o.state.buffers.depth.getReversed();if(Q.camera._reversedDepth=ot,Q.map===null||P===!0){if(Q.map!==null&&(Q.map.depthTexture!==null&&(Q.map.depthTexture.dispose(),Q.map.depthTexture=null),Q.map.dispose()),this.type===wl){if(Y.isPointLight){oe("WebGLShadowMap: VSM shadow maps are not supported for PointLights. Use PCF or BasicShadowMap instead.");continue}Q.map=new Vi(c.x,c.y,{format:er,type:da,minFilter:zn,magFilter:zn,generateMipmaps:!1}),Q.map.texture.name=Y.name+".shadowMap",Q.map.depthTexture=new Il(c.x,c.y,Hi),Q.map.depthTexture.name=Y.name+".shadowMapDepth",Q.map.depthTexture.format=za,Q.map.depthTexture.compareFunction=null,Q.map.depthTexture.minFilter=On,Q.map.depthTexture.magFilter=On}else Y.isPointLight?(Q.map=new _S(c.x),Q.map.depthTexture=new wb(c.x,fa)):(Q.map=new Vi(c.x,c.y),Q.map.depthTexture=new Il(c.x,c.y,fa)),Q.map.depthTexture.name=Y.name+".shadowMap",Q.map.depthTexture.format=za,this.type===Nu?(Q.map.depthTexture.compareFunction=ot?cm:lm,Q.map.depthTexture.minFilter=zn,Q.map.depthTexture.magFilter=zn):(Q.map.depthTexture.compareFunction=null,Q.map.depthTexture.minFilter=On,Q.map.depthTexture.magFilter=On);Q.camera.updateProjectionMatrix()}Q.map.isWebGLCubeRenderTarget!==!0&&(Q.map.width!==c.x||Q.map.height!==c.y)&&Q.map.setSize(c.x,c.y);const vt=Q.map.isWebGLCubeRenderTarget?6:Q.getViewportCount();Y.isPointLight!==!0&&Q.updateMatrices(Y,b);for(let Tt=0;Tt<vt;Tt++){const Zt=Q.getCamera(Tt);if(Y.isPointLight){const Yt=Q.camera,H=Q.matrix,F=Y.distance||Yt.far;F!==Yt.far&&(Yt.far=F,Yt.updateProjectionMatrix()),Al.setFromMatrixPosition(Y.matrixWorld),Yt.position.copy(Al),sp.copy(Yt.position),sp.add(Aw[Tt]),Yt.up.copy(Rw[Tt]),Yt.lookAt(sp),Yt.updateMatrixWorld(),H.makeTranslation(-Al.x,-Al.y,-Al.z),Px.multiplyMatrices(Yt.projectionMatrix,Yt.matrixWorldInverse),Q._frustum.setFromProjectionMatrix(Px,Yt.coordinateSystem,Yt.reversedDepth)}if(Q.map.isWebGLCubeRenderTarget)o.setRenderTarget(Q.map,Tt),o.clear();else{Tt===0&&(o.setRenderTarget(Q.map),o.clear());const Yt=Q.getViewport(Tt);d.set(f.x*Yt.x,f.y*Yt.y,f.x*Yt.z,f.y*Yt.w),J.viewport(d)}r=Q.getFrustum(Tt),C(B,b,Zt,Y,this.type)}Q.isPointLightShadow!==!0&&this.type===wl&&O(Q,b),Q.needsUpdate=!1}S=this.type,y.needsUpdate=!1,o.setRenderTarget(L,X,j)};function O(U,B){const b=e.update(N);_.defines.VSM_SAMPLES!==U.blurSamples&&(_.defines.VSM_SAMPLES=U.blurSamples,M.defines.VSM_SAMPLES=U.blurSamples,_.needsUpdate=!0,M.needsUpdate=!0),U.mapPass===null?U.mapPass=new Vi(c.x,c.y,{format:er,type:da}):(U.mapPass.width!==U.map.width||U.mapPass.height!==U.map.height)&&U.mapPass.setSize(U.map.width,U.map.height),_.uniforms.shadow_pass.value=U.map.depthTexture,_.uniforms.resolution.value.set(U.map.width,U.map.height),_.uniforms.radius.value=U.radius,o.setRenderTarget(U.mapPass),o.clear(),o.renderBufferDirect(B,null,b,_,N,null),M.uniforms.shadow_pass.value=U.mapPass.texture,M.uniforms.resolution.value.set(U.map.width,U.map.height),M.uniforms.radius.value=U.radius,o.setRenderTarget(U.map),o.clear(),o.renderBufferDirect(B,null,b,M,N,null)}function k(U,B,b,L){let X=null;const j=b.isPointLight===!0?U.customDistanceMaterial:U.customDepthMaterial;if(j!==void 0)X=j;else if(X=b.isPointLight===!0?m:h,o.localClippingEnabled&&B.clipShadows===!0&&Array.isArray(B.clippingPlanes)&&B.clippingPlanes.length!==0||B.displacementMap&&B.displacementScale!==0||B.alphaMap&&B.alphaTest>0||B.map&&B.alphaTest>0||B.alphaToCoverage===!0){const J=X.uuid,P=B.uuid;let q=p[J];q===void 0&&(q={},p[J]=q);let $=q[P];$===void 0&&($=X.clone(),q[P]=$,B.addEventListener("dispose",D)),X=$}if(X.visible=B.visible,X.wireframe=B.wireframe,L===wl?X.side=B.shadowSide!==null?B.shadowSide:B.side:X.side=B.shadowSide!==null?B.shadowSide:v[B.side],X.alphaMap=B.alphaMap,X.alphaTest=B.alphaToCoverage===!0?.5:B.alphaTest,X.map=B.map,X.clipShadows=B.clipShadows,X.clippingPlanes=B.clippingPlanes,X.clipIntersection=B.clipIntersection,X.displacementMap=B.displacementMap,X.displacementScale=B.displacementScale,X.displacementBias=B.displacementBias,X.wireframeLinewidth=B.wireframeLinewidth,X.linewidth=B.linewidth,b.isPointLight===!0&&X.isMeshDistanceMaterial===!0){const J=o.properties.get(X);J.light=b}return X}function C(U,B,b,L,X){if(U.visible===!1)return;if(U.layers.test(B.layers)&&(U.isMesh||U.isLine||U.isPoints)&&(U.castShadow||U.receiveShadow&&X===wl)&&(!U.frustumCulled||U.intersectsFrustum(r))){U.modelViewMatrix.multiplyMatrices(b.matrixWorldInverse,U.matrixWorld);const P=e.update(U),q=U.material;if(Array.isArray(q)){const $=P.groups;for(let Y=0,Q=$.length;Y<Q;Y++){const pt=$[Y],ot=q[pt.materialIndex];if(ot&&ot.visible){const vt=k(U,ot,L,X);U.onBeforeShadow(o,U,B,b,P,vt,pt),o.renderBufferDirect(b,null,P,vt,U,pt),U.onAfterShadow(o,U,B,b,P,vt,pt)}}}else if(q.visible){const $=k(U,q,L,X);U.onBeforeShadow(o,U,B,b,P,$,null),o.renderBufferDirect(b,null,P,$,U,null),U.onAfterShadow(o,U,B,b,P,$,null)}}const J=U.children;for(let P=0,q=J.length;P<q;P++)C(J[P],B,b,L,X)}function D(U){U.target.removeEventListener("dispose",D);for(const b in p){const L=p[b],X=U.target.uuid;X in L&&(L[X].dispose(),delete L[X])}}}function Cw(o,e){function i(){let K=!1;const Ut=new on;let bt=null;const Lt=new on(0,0,0,0);return{setMask:function(Vt){bt!==Vt&&!K&&(o.colorMask(Vt,Vt,Vt,Vt),bt=Vt)},setLocked:function(Vt){K=Vt},setClear:function(Vt,Rt,$t,Gt,Le){Le===!0&&(Vt*=Gt,Rt*=Gt,$t*=Gt),Ut.set(Vt,Rt,$t,Gt),Lt.equals(Ut)===!1&&(o.clearColor(Vt,Rt,$t,Gt),Lt.copy(Ut))},reset:function(){K=!1,bt=null,Lt.set(-1,0,0,0)}}}function r(){let K=!1,Ut=!1,bt=null,Lt=null,Vt=null;return{setReversed:function(Rt){if(Ut!==Rt){const $t=e.get("EXT_clip_control");Rt?$t.clipControlEXT($t.LOWER_LEFT_EXT,$t.ZERO_TO_ONE_EXT):$t.clipControlEXT($t.LOWER_LEFT_EXT,$t.NEGATIVE_ONE_TO_ONE_EXT),Ut=Rt;const Gt=Vt;Vt=null,this.setClear(Gt)}},getReversed:function(){return Ut},setTest:function(Rt){Rt?st(o.DEPTH_TEST):At(o.DEPTH_TEST)},setMask:function(Rt){bt!==Rt&&!K&&(o.depthMask(Rt),bt=Rt)},setFunc:function(Rt){if(Ut&&(Rt=nb[Rt]),Lt!==Rt){switch(Rt){case op:o.depthFunc(o.NEVER);break;case lp:o.depthFunc(o.ALWAYS);break;case cp:o.depthFunc(o.LESS);break;case Ul:o.depthFunc(o.LEQUAL);break;case up:o.depthFunc(o.EQUAL);break;case fp:o.depthFunc(o.GEQUAL);break;case dp:o.depthFunc(o.GREATER);break;case hp:o.depthFunc(o.NOTEQUAL);break;default:o.depthFunc(o.LEQUAL)}Lt=Rt}},setLocked:function(Rt){K=Rt},setClear:function(Rt){Vt!==Rt&&(Vt=Rt,Ut&&(Rt=1-Rt),o.clearDepth(Rt))},reset:function(){K=!1,bt=null,Lt=null,Vt=null,Ut=!1}}}function c(){let K=!1,Ut=null,bt=null,Lt=null,Vt=null,Rt=null,$t=null,Gt=null,Le=null;return{setTest:function(ue){K||(ue?st(o.STENCIL_TEST):At(o.STENCIL_TEST))},setMask:function(ue){Ut!==ue&&!K&&(o.stencilMask(ue),Ut=ue)},setFunc:function(ue,ni,mi){(bt!==ue||Lt!==ni||Vt!==mi)&&(o.stencilFunc(ue,ni,mi),bt=ue,Lt=ni,Vt=mi)},setOp:function(ue,ni,mi){(Rt!==ue||$t!==ni||Gt!==mi)&&(o.stencilOp(ue,ni,mi),Rt=ue,$t=ni,Gt=mi)},setLocked:function(ue){K=ue},setClear:function(ue){Le!==ue&&(o.clearStencil(ue),Le=ue)},reset:function(){K=!1,Ut=null,bt=null,Lt=null,Vt=null,Rt=null,$t=null,Gt=null,Le=null}}}const f=new i,d=new r,h=new c,m=new WeakMap,p=new WeakMap;let x={},v={},_={},M=new WeakMap,R=[],N=null,y=!1,S=null,O=null,k=null,C=null,D=null,U=null,B=null,b=new he(0,0,0),L=0,X=!1,j=null,J=null,P=null,q=null,$=null;const Y=o.getParameter(o.MAX_COMBINED_TEXTURE_IMAGE_UNITS);let Q=!1,pt=0;const ot=o.getParameter(o.VERSION);ot.indexOf("WebGL")!==-1?(pt=parseFloat(/^WebGL (\d)/.exec(ot)[1]),Q=pt>=1):ot.indexOf("OpenGL ES")!==-1&&(pt=parseFloat(/^OpenGL ES (\d)/.exec(ot)[1]),Q=pt>=2);let vt=null,Tt={};const Zt=o.getParameter(o.SCISSOR_BOX),Yt=o.getParameter(o.VIEWPORT),H=new on().fromArray(Zt),F=new on().fromArray(Yt);function gt(K,Ut,bt,Lt){const Vt=new Uint8Array(4),Rt=o.createTexture();o.bindTexture(K,Rt),o.texParameteri(K,o.TEXTURE_MIN_FILTER,o.NEAREST),o.texParameteri(K,o.TEXTURE_MAG_FILTER,o.NEAREST);for(let $t=0;$t<bt;$t++)K===o.TEXTURE_3D||K===o.TEXTURE_2D_ARRAY?o.texImage3D(Ut,0,o.RGBA,1,1,Lt,0,o.RGBA,o.UNSIGNED_BYTE,Vt):o.texImage2D(Ut+$t,0,o.RGBA,1,1,0,o.RGBA,o.UNSIGNED_BYTE,Vt);return Rt}const G={};G[o.TEXTURE_2D]=gt(o.TEXTURE_2D,o.TEXTURE_2D,1),G[o.TEXTURE_CUBE_MAP]=gt(o.TEXTURE_CUBE_MAP,o.TEXTURE_CUBE_MAP_POSITIVE_X,6),G[o.TEXTURE_2D_ARRAY]=gt(o.TEXTURE_2D_ARRAY,o.TEXTURE_2D_ARRAY,1,1),G[o.TEXTURE_3D]=gt(o.TEXTURE_3D,o.TEXTURE_3D,1,1),f.setClear(0,0,0,1),d.setClear(1),h.setClear(0),st(o.DEPTH_TEST),d.setFunc(Ul),te(!1),ne(I_),st(o.CULL_FACE),pe(Ia);function st(K){x[K]!==!0&&(o.enable(K),x[K]=!0)}function At(K){x[K]!==!1&&(o.disable(K),x[K]=!1)}function Dt(K,Ut){return _[K]!==Ut?(o.bindFramebuffer(K,Ut),_[K]=Ut,K===o.DRAW_FRAMEBUFFER&&(_[o.FRAMEBUFFER]=Ut),K===o.FRAMEBUFFER&&(_[o.DRAW_FRAMEBUFFER]=Ut),!0):!1}function xt(K,Ut){let bt=R,Lt=!1;if(K){bt=M.get(Ut),bt===void 0&&(bt=[],M.set(Ut,bt));const Vt=K.textures;if(bt.length!==Vt.length||bt[0]!==o.COLOR_ATTACHMENT0){for(let Rt=0,$t=Vt.length;Rt<$t;Rt++)bt[Rt]=o.COLOR_ATTACHMENT0+Rt;bt.length=Vt.length,Lt=!0}}else bt[0]!==o.BACK&&(bt[0]=o.BACK,Lt=!0);Lt&&o.drawBuffers(bt)}function Ct(K){return N!==K?(o.useProgram(K),N=K,!0):!1}const Fe={[oo]:o.FUNC_ADD,[EE]:o.FUNC_SUBTRACT,[bE]:o.FUNC_REVERSE_SUBTRACT};Fe[TE]=o.MIN,Fe[AE]=o.MAX;const le={[RE]:o.ZERO,[wE]:o.ONE,[CE]:o.SRC_COLOR,[kx]:o.SRC_ALPHA,[PE]:o.SRC_ALPHA_SATURATE,[LE]:o.DST_COLOR,[DE]:o.DST_ALPHA,[NE]:o.ONE_MINUS_SRC_COLOR,[Xx]:o.ONE_MINUS_SRC_ALPHA,[OE]:o.ONE_MINUS_DST_COLOR,[UE]:o.ONE_MINUS_DST_ALPHA,[IE]:o.CONSTANT_COLOR,[BE]:o.ONE_MINUS_CONSTANT_COLOR,[zE]:o.CONSTANT_ALPHA,[FE]:o.ONE_MINUS_CONSTANT_ALPHA};function pe(K,Ut,bt,Lt,Vt,Rt,$t,Gt,Le,ue){if(K===Ia){y===!0&&(At(o.BLEND),y=!1);return}if(y===!1&&(st(o.BLEND),y=!0),K!==ME){if(K!==S||ue!==X){if((O!==oo||D!==oo)&&(o.blendEquation(o.FUNC_ADD),O=oo,D=oo),ue)switch(K){case Dl:o.blendFuncSeparate(o.ONE,o.ONE_MINUS_SRC_ALPHA,o.ONE,o.ONE_MINUS_SRC_ALPHA);break;case B_:o.blendFunc(o.ONE,o.ONE);break;case z_:o.blendFuncSeparate(o.ZERO,o.ONE_MINUS_SRC_COLOR,o.ZERO,o.ONE);break;case F_:o.blendFuncSeparate(o.DST_COLOR,o.ONE_MINUS_SRC_ALPHA,o.ZERO,o.ONE);break;default:ze("WebGLState: Invalid blending: ",K);break}else switch(K){case Dl:o.blendFuncSeparate(o.SRC_ALPHA,o.ONE_MINUS_SRC_ALPHA,o.ONE,o.ONE_MINUS_SRC_ALPHA);break;case B_:o.blendFuncSeparate(o.SRC_ALPHA,o.ONE,o.ONE,o.ONE);break;case z_:ze("WebGLState: SubtractiveBlending requires material.premultipliedAlpha = true");break;case F_:ze("WebGLState: MultiplyBlending requires material.premultipliedAlpha = true");break;default:ze("WebGLState: Invalid blending: ",K);break}k=null,C=null,U=null,B=null,b.set(0,0,0),L=0,S=K,X=ue}return}Vt=Vt||Ut,Rt=Rt||bt,$t=$t||Lt,(Ut!==O||Vt!==D)&&(o.blendEquationSeparate(Fe[Ut],Fe[Vt]),O=Ut,D=Vt),(bt!==k||Lt!==C||Rt!==U||$t!==B)&&(o.blendFuncSeparate(le[bt],le[Lt],le[Rt],le[$t]),k=bt,C=Lt,U=Rt,B=$t),(Gt.equals(b)===!1||Le!==L)&&(o.blendColor(Gt.r,Gt.g,Gt.b,Le),b.copy(Gt),L=Le),S=K,X=!1}function ve(K,Ut){K.side===Bi?At(o.CULL_FACE):st(o.CULL_FACE);let bt=K.side===qn;Ut&&(bt=!bt),te(bt),K.blending===Dl&&K.transparent===!1?pe(Ia):pe(K.blending,K.blendEquation,K.blendSrc,K.blendDst,K.blendEquationAlpha,K.blendSrcAlpha,K.blendDstAlpha,K.blendColor,K.blendAlpha,K.premultipliedAlpha),d.setFunc(K.depthFunc),d.setTest(K.depthTest),d.setMask(K.depthWrite),f.setMask(K.colorWrite);const Lt=K.stencilWrite;h.setTest(Lt),Lt&&(h.setMask(K.stencilWriteMask),h.setFunc(K.stencilFunc,K.stencilRef,K.stencilFuncMask),h.setOp(K.stencilFail,K.stencilZFail,K.stencilZPass)),sn(K.polygonOffset,K.polygonOffsetFactor,K.polygonOffsetUnits),K.alphaToCoverage===!0?st(o.SAMPLE_ALPHA_TO_COVERAGE):At(o.SAMPLE_ALPHA_TO_COVERAGE)}function te(K){j!==K&&(K?o.frontFace(o.CW):o.frontFace(o.CCW),j=K)}function ne(K){K!==SE?(st(o.CULL_FACE),K!==J&&(K===I_?o.cullFace(o.BACK):K===yE?o.cullFace(o.FRONT):o.cullFace(o.FRONT_AND_BACK))):At(o.CULL_FACE),J=K}function Be(K){K!==P&&(Q&&o.lineWidth(K),P=K)}function sn(K,Ut,bt){K?(st(o.POLYGON_OFFSET_FILL),(q!==Ut||$!==bt)&&(q=Ut,$=bt,d.getReversed()&&(Ut=-Ut),o.polygonOffset(Ut,bt))):At(o.POLYGON_OFFSET_FILL)}function Ue(K){K?st(o.SCISSOR_TEST):At(o.SCISSOR_TEST)}function je(K){K===void 0&&(K=o.TEXTURE0+Y-1),vt!==K&&(o.activeTexture(K),vt=K)}function Z(K,Ut,bt){bt===void 0&&(vt===null?bt=o.TEXTURE0+Y-1:bt=vt);let Lt=Tt[bt];Lt===void 0&&(Lt={type:void 0,texture:void 0},Tt[bt]=Lt),(Lt.type!==K||Lt.texture!==Ut)&&(vt!==bt&&(o.activeTexture(bt),vt=bt),o.bindTexture(K,Ut||G[K]),Lt.type=K,Lt.texture=Ut)}function Je(){const K=Tt[vt];K!==void 0&&K.type!==void 0&&(o.bindTexture(K.type,null),K.type=void 0,K.texture=void 0)}function Me(){try{o.compressedTexImage2D(...arguments)}catch(K){ze("WebGLState:",K)}}function I(){try{o.compressedTexImage3D(...arguments)}catch(K){ze("WebGLState:",K)}}function E(){try{o.texSubImage2D(...arguments)}catch(K){ze("WebGLState:",K)}}function at(){try{o.texSubImage3D(...arguments)}catch(K){ze("WebGLState:",K)}}function lt(){try{o.compressedTexSubImage2D(...arguments)}catch(K){ze("WebGLState:",K)}}function et(){try{o.compressedTexSubImage3D(...arguments)}catch(K){ze("WebGLState:",K)}}function St(){try{o.texStorage2D(...arguments)}catch(K){ze("WebGLState:",K)}}function wt(){try{o.texStorage3D(...arguments)}catch(K){ze("WebGLState:",K)}}function _t(){try{o.texImage2D(...arguments)}catch(K){ze("WebGLState:",K)}}function yt(){try{o.texImage3D(...arguments)}catch(K){ze("WebGLState:",K)}}function Nt(K){return v[K]!==void 0?v[K]:o.getParameter(K)}function Jt(K,Ut){v[K]!==Ut&&(o.pixelStorei(K,Ut),v[K]=Ut)}function Bt(K){H.equals(K)===!1&&(o.scissor(K.x,K.y,K.z,K.w),H.copy(K))}function It(K){F.equals(K)===!1&&(o.viewport(K.x,K.y,K.z,K.w),F.copy(K))}function kt(K,Ut){let bt=p.get(Ut);bt===void 0&&(bt=new WeakMap,p.set(Ut,bt));let Lt=bt.get(K);Lt===void 0&&(Lt=o.getUniformBlockIndex(Ut,K.name),bt.set(K,Lt))}function ie(K,Ut){const Lt=p.get(Ut).get(K);m.get(Ut)!==Lt&&(o.uniformBlockBinding(Ut,Lt,K.__bindingPointIndex),m.set(Ut,Lt))}function ce(){o.disable(o.BLEND),o.disable(o.CULL_FACE),o.disable(o.DEPTH_TEST),o.disable(o.POLYGON_OFFSET_FILL),o.disable(o.SCISSOR_TEST),o.disable(o.STENCIL_TEST),o.disable(o.SAMPLE_ALPHA_TO_COVERAGE),o.blendEquation(o.FUNC_ADD),o.blendFunc(o.ONE,o.ZERO),o.blendFuncSeparate(o.ONE,o.ZERO,o.ONE,o.ZERO),o.blendColor(0,0,0,0),o.colorMask(!0,!0,!0,!0),o.clearColor(0,0,0,0),o.depthMask(!0),o.depthFunc(o.LESS),d.setReversed(!1),o.clearDepth(1),o.stencilMask(4294967295),o.stencilFunc(o.ALWAYS,0,4294967295),o.stencilOp(o.KEEP,o.KEEP,o.KEEP),o.clearStencil(0),o.cullFace(o.BACK),o.frontFace(o.CCW),o.polygonOffset(0,0),o.activeTexture(o.TEXTURE0),o.bindFramebuffer(o.FRAMEBUFFER,null),o.bindFramebuffer(o.DRAW_FRAMEBUFFER,null),o.bindFramebuffer(o.READ_FRAMEBUFFER,null),o.useProgram(null),o.lineWidth(1),o.scissor(0,0,o.canvas.width,o.canvas.height),o.viewport(0,0,o.canvas.width,o.canvas.height),o.pixelStorei(o.PACK_ALIGNMENT,4),o.pixelStorei(o.UNPACK_ALIGNMENT,4),o.pixelStorei(o.UNPACK_FLIP_Y_WEBGL,!1),o.pixelStorei(o.UNPACK_PREMULTIPLY_ALPHA_WEBGL,!1),o.pixelStorei(o.UNPACK_COLORSPACE_CONVERSION_WEBGL,o.BROWSER_DEFAULT_WEBGL),o.pixelStorei(o.PACK_ROW_LENGTH,0),o.pixelStorei(o.PACK_SKIP_PIXELS,0),o.pixelStorei(o.PACK_SKIP_ROWS,0),o.pixelStorei(o.UNPACK_ROW_LENGTH,0),o.pixelStorei(o.UNPACK_IMAGE_HEIGHT,0),o.pixelStorei(o.UNPACK_SKIP_PIXELS,0),o.pixelStorei(o.UNPACK_SKIP_ROWS,0),o.pixelStorei(o.UNPACK_SKIP_IMAGES,0),x={},v={},vt=null,Tt={},_={},M=new WeakMap,R=[],N=null,y=!1,S=null,O=null,k=null,C=null,D=null,U=null,B=null,b=new he(0,0,0),L=0,X=!1,j=null,J=null,P=null,q=null,$=null,H.set(0,0,o.canvas.width,o.canvas.height),F.set(0,0,o.canvas.width,o.canvas.height),f.reset(),d.reset(),h.reset()}return{buffers:{color:f,depth:d,stencil:h},enable:st,disable:At,bindFramebuffer:Dt,drawBuffers:xt,useProgram:Ct,setBlending:pe,setMaterial:ve,setFlipSided:te,setCullFace:ne,setLineWidth:Be,setPolygonOffset:sn,setScissorTest:Ue,activeTexture:je,bindTexture:Z,unbindTexture:Je,compressedTexImage2D:Me,compressedTexImage3D:I,texImage2D:_t,texImage3D:yt,pixelStorei:Jt,getParameter:Nt,updateUBOMapping:kt,uniformBlockBinding:ie,texStorage2D:St,texStorage3D:wt,texSubImage2D:E,texSubImage3D:at,compressedTexSubImage2D:lt,compressedTexSubImage3D:et,scissor:Bt,viewport:It,reset:ce}}function Nw(o,e,i,r,c,f,d){const h=e.has("WEBGL_multisampled_render_to_texture")?e.get("WEBGL_multisampled_render_to_texture"):null,m=typeof navigator>"u"?!1:/OculusBrowser/g.test(navigator.userAgent),p=new Re,x=new WeakMap,v=new Set;let _;const M=new WeakMap;let R=!1;try{R=typeof OffscreenCanvas<"u"&&new OffscreenCanvas(1,1).getContext("2d")!==null}catch{}function N(I,E){return R?new OffscreenCanvas(I,E):Gu("canvas")}function y(I,E,at){let lt=1;const et=Me(I);if((et.width>at||et.height>at)&&(lt=at/Math.max(et.width,et.height)),lt<1)if(typeof HTMLImageElement<"u"&&I instanceof HTMLImageElement||typeof HTMLCanvasElement<"u"&&I instanceof HTMLCanvasElement||typeof ImageBitmap<"u"&&I instanceof ImageBitmap||typeof VideoFrame<"u"&&I instanceof VideoFrame){const St=Math.floor(lt*et.width),wt=Math.floor(lt*et.height);_===void 0&&(_=N(St,wt));const _t=E?N(St,wt):_;return _t.width=St,_t.height=wt,_t.getContext("2d").drawImage(I,0,0,St,wt),oe("WebGLRenderer: Texture has been resized from ("+et.width+"x"+et.height+") to ("+St+"x"+wt+")."),_t}else return"data"in I&&oe("WebGLRenderer: Image in DataTexture is too big ("+et.width+"x"+et.height+")."),I;return I}function S(I){return I.generateMipmaps}function O(I){o.generateMipmap(I)}function k(I){return I.isWebGLCubeRenderTarget?o.TEXTURE_CUBE_MAP:I.isWebGL3DRenderTarget?o.TEXTURE_3D:I.isWebGLArrayRenderTarget||I.isCompressedArrayTexture?o.TEXTURE_2D_ARRAY:o.TEXTURE_2D}function C(I,E,at,lt,et,St=!1){if(I!==null){if(o[I]!==void 0)return o[I];oe("WebGLRenderer: Attempt to use non-existing WebGL internal format '"+I+"'")}let wt;lt&&(wt=e.get("EXT_texture_norm16"),wt||oe("WebGLRenderer: Unable to use normalized textures without EXT_texture_norm16 extension"));let _t=E;if(E===o.RED&&(at===o.FLOAT&&(_t=o.R32F),at===o.HALF_FLOAT&&(_t=o.R16F),at===o.UNSIGNED_BYTE&&(_t=o.R8),at===o.UNSIGNED_SHORT&&wt&&(_t=wt.R16_EXT),at===o.SHORT&&wt&&(_t=wt.R16_SNORM_EXT)),E===o.RED_INTEGER&&(at===o.UNSIGNED_BYTE&&(_t=o.R8UI),at===o.UNSIGNED_SHORT&&(_t=o.R16UI),at===o.UNSIGNED_INT&&(_t=o.R32UI),at===o.BYTE&&(_t=o.R8I),at===o.SHORT&&(_t=o.R16I),at===o.INT&&(_t=o.R32I)),E===o.RG&&(at===o.FLOAT&&(_t=o.RG32F),at===o.HALF_FLOAT&&(_t=o.RG16F),at===o.UNSIGNED_BYTE&&(_t=o.RG8),at===o.UNSIGNED_SHORT&&wt&&(_t=wt.RG16_EXT),at===o.SHORT&&wt&&(_t=wt.RG16_SNORM_EXT)),E===o.RG_INTEGER&&(at===o.UNSIGNED_BYTE&&(_t=o.RG8UI),at===o.UNSIGNED_SHORT&&(_t=o.RG16UI),at===o.UNSIGNED_INT&&(_t=o.RG32UI),at===o.BYTE&&(_t=o.RG8I),at===o.SHORT&&(_t=o.RG16I),at===o.INT&&(_t=o.RG32I)),E===o.RGB_INTEGER&&(at===o.UNSIGNED_BYTE&&(_t=o.RGB8UI),at===o.UNSIGNED_SHORT&&(_t=o.RGB16UI),at===o.UNSIGNED_INT&&(_t=o.RGB32UI),at===o.BYTE&&(_t=o.RGB8I),at===o.SHORT&&(_t=o.RGB16I),at===o.INT&&(_t=o.RGB32I)),E===o.RGBA_INTEGER&&(at===o.UNSIGNED_BYTE&&(_t=o.RGBA8UI),at===o.UNSIGNED_SHORT&&(_t=o.RGBA16UI),at===o.UNSIGNED_INT&&(_t=o.RGBA32UI),at===o.BYTE&&(_t=o.RGBA8I),at===o.SHORT&&(_t=o.RGBA16I),at===o.INT&&(_t=o.RGBA32I)),E===o.RGB&&(at===o.UNSIGNED_SHORT&&wt&&(_t=wt.RGB16_EXT),at===o.SHORT&&wt&&(_t=wt.RGB16_SNORM_EXT),at===o.UNSIGNED_INT_5_9_9_9_REV&&(_t=o.RGB9_E5),at===o.UNSIGNED_INT_10F_11F_11F_REV&&(_t=o.R11F_G11F_B10F)),E===o.RGBA){const yt=St?Hu:Pe.getTransfer(et);at===o.FLOAT&&(_t=o.RGBA32F),at===o.HALF_FLOAT&&(_t=o.RGBA16F),at===o.UNSIGNED_BYTE&&(_t=yt===Ye?o.SRGB8_ALPHA8:o.RGBA8),at===o.UNSIGNED_SHORT&&wt&&(_t=wt.RGBA16_EXT),at===o.SHORT&&wt&&(_t=wt.RGBA16_SNORM_EXT),at===o.UNSIGNED_SHORT_4_4_4_4&&(_t=o.RGBA4),at===o.UNSIGNED_SHORT_5_5_5_1&&(_t=o.RGB5_A1)}return(_t===o.R16F||_t===o.R32F||_t===o.RG16F||_t===o.RG32F||_t===o.RGBA16F||_t===o.RGBA32F)&&e.get("EXT_color_buffer_float"),_t}function D(I,E){let at;return I?E===null||E===fa||E===Ol?at=o.DEPTH24_STENCIL8:E===Hi?at=o.DEPTH32F_STENCIL8:E===Ll&&(at=o.DEPTH24_STENCIL8,oe("DepthTexture: 16 bit depth attachment is not supported with stencil. Using 24-bit attachment.")):E===null||E===fa||E===Ol?at=o.DEPTH_COMPONENT24:E===Hi?at=o.DEPTH_COMPONENT32F:E===Ll&&(at=o.DEPTH_COMPONENT16),at}function U(I,E){return S(I)===!0||I.isFramebufferTexture&&I.minFilter!==On&&I.minFilter!==zn?Math.log2(Math.max(E.width,E.height))+1:I.mipmaps!==void 0&&I.mipmaps.length>0?I.mipmaps.length:I.isCompressedTexture&&Array.isArray(I.image)?E.mipmaps.length:1}function B(I){const E=I.target;E.removeEventListener("dispose",B),L(E),E.isVideoTexture&&x.delete(E),E.isHTMLTexture&&v.delete(E)}function b(I){const E=I.target;E.removeEventListener("dispose",b),j(E)}function L(I){const E=r.get(I);if(E.__webglInit===void 0)return;const at=I.source,lt=M.get(at);if(lt){const et=lt[E.__cacheKey];et.usedTimes--,et.usedTimes===0&&X(I),Object.keys(lt).length===0&&M.delete(at)}r.remove(I)}function X(I){const E=r.get(I);o.deleteTexture(E.__webglTexture);const at=I.source,lt=M.get(at);delete lt[E.__cacheKey],d.memory.textures--}function j(I){const E=r.get(I);if(I.depthTexture&&(I.depthTexture.dispose(),r.remove(I.depthTexture)),I.isWebGLCubeRenderTarget)for(let lt=0;lt<6;lt++){if(Array.isArray(E.__webglFramebuffer[lt]))for(let et=0;et<E.__webglFramebuffer[lt].length;et++)o.deleteFramebuffer(E.__webglFramebuffer[lt][et]);else o.deleteFramebuffer(E.__webglFramebuffer[lt]);E.__webglDepthbuffer&&o.deleteRenderbuffer(E.__webglDepthbuffer[lt])}else{if(Array.isArray(E.__webglFramebuffer))for(let lt=0;lt<E.__webglFramebuffer.length;lt++)o.deleteFramebuffer(E.__webglFramebuffer[lt]);else o.deleteFramebuffer(E.__webglFramebuffer);if(E.__webglDepthbuffer&&o.deleteRenderbuffer(E.__webglDepthbuffer),E.__webglMultisampledFramebuffer&&o.deleteFramebuffer(E.__webglMultisampledFramebuffer),E.__webglColorRenderbuffer)for(let lt=0;lt<E.__webglColorRenderbuffer.length;lt++)E.__webglColorRenderbuffer[lt]&&o.deleteRenderbuffer(E.__webglColorRenderbuffer[lt]);E.__webglDepthRenderbuffer&&o.deleteRenderbuffer(E.__webglDepthRenderbuffer)}const at=I.textures;for(let lt=0,et=at.length;lt<et;lt++){const St=r.get(at[lt]);St.__webglTexture&&(o.deleteTexture(St.__webglTexture),d.memory.textures--),r.remove(at[lt])}r.remove(I)}let J=0;function P(){J=0}function q(){return J}function $(I){J=I}function Y(){const I=J;return I>=c.maxTextures&&oe("WebGLTextures: Trying to use "+(I+1)+" texture units while this GPU supports only "+c.maxTextures),J+=1,I}function Q(I){const E=[];return E.push(I.wrapS),E.push(I.wrapT),E.push(I.wrapR||0),E.push(I.magFilter),E.push(I.minFilter),E.push(I.anisotropy),E.push(I.internalFormat),E.push(I.format),E.push(I.type),E.push(I.generateMipmaps),E.push(I.premultiplyAlpha),E.push(I.flipY),E.push(I.unpackAlignment),E.push(I.colorSpace),E.join()}function pt(I,E){const at=r.get(I);if(I.isVideoTexture&&Z(I),I.isRenderTargetTexture===!1&&I.isExternalTexture!==!0&&I.version>0&&at.__version!==I.version){const lt=I.image;if(lt===null)oe("WebGLRenderer: Texture marked for update but no image data found.");else if(lt.complete===!1)oe("WebGLRenderer: Texture marked for update but image is incomplete");else{At(at,I,E);return}}else I.isExternalTexture&&(at.__webglTexture=I.sourceTexture?I.sourceTexture:null);i.bindTexture(o.TEXTURE_2D,at.__webglTexture,o.TEXTURE0+E)}function ot(I,E){const at=r.get(I);if(I.isRenderTargetTexture===!1&&I.version>0&&at.__version!==I.version){At(at,I,E);return}else I.isExternalTexture&&(at.__webglTexture=I.sourceTexture?I.sourceTexture:null);i.bindTexture(o.TEXTURE_2D_ARRAY,at.__webglTexture,o.TEXTURE0+E)}function vt(I,E){const at=r.get(I);if(I.isRenderTargetTexture===!1&&I.version>0&&at.__version!==I.version){At(at,I,E);return}i.bindTexture(o.TEXTURE_3D,at.__webglTexture,o.TEXTURE0+E)}function Tt(I,E){const at=r.get(I);if(I.isCubeDepthTexture!==!0&&I.version>0&&at.__version!==I.version){Dt(at,I,E);return}i.bindTexture(o.TEXTURE_CUBE_MAP,at.__webglTexture,o.TEXTURE0+E)}const Zt={[pp]:o.REPEAT,[Pa]:o.CLAMP_TO_EDGE,[mp]:o.MIRRORED_REPEAT},Yt={[On]:o.NEAREST,[VE]:o.NEAREST_MIPMAP_NEAREST,[ou]:o.NEAREST_MIPMAP_LINEAR,[zn]:o.LINEAR,[wh]:o.LINEAR_MIPMAP_NEAREST,[Qs]:o.LINEAR_MIPMAP_LINEAR},H={[qE]:o.NEVER,[QE]:o.ALWAYS,[YE]:o.LESS,[lm]:o.LEQUAL,[jE]:o.EQUAL,[cm]:o.GEQUAL,[ZE]:o.GREATER,[KE]:o.NOTEQUAL};function F(I,E){if(E.type===Hi&&e.has("OES_texture_float_linear")===!1&&(E.magFilter===zn||E.magFilter===wh||E.magFilter===ou||E.magFilter===Qs||E.minFilter===zn||E.minFilter===wh||E.minFilter===ou||E.minFilter===Qs)&&oe("WebGLRenderer: Unable to use linear filtering with floating point textures. OES_texture_float_linear not supported on this device."),o.texParameteri(I,o.TEXTURE_WRAP_S,Zt[E.wrapS]),o.texParameteri(I,o.TEXTURE_WRAP_T,Zt[E.wrapT]),(I===o.TEXTURE_3D||I===o.TEXTURE_2D_ARRAY)&&o.texParameteri(I,o.TEXTURE_WRAP_R,Zt[E.wrapR]),o.texParameteri(I,o.TEXTURE_MAG_FILTER,Yt[E.magFilter]),o.texParameteri(I,o.TEXTURE_MIN_FILTER,Yt[E.minFilter]),E.compareFunction&&(o.texParameteri(I,o.TEXTURE_COMPARE_MODE,o.COMPARE_REF_TO_TEXTURE),o.texParameteri(I,o.TEXTURE_COMPARE_FUNC,H[E.compareFunction])),e.has("EXT_texture_filter_anisotropic")===!0){if(E.magFilter===On||E.minFilter!==ou&&E.minFilter!==Qs||E.type===Hi&&e.has("OES_texture_float_linear")===!1)return;if(E.anisotropy>1||r.get(E).__currentAnisotropy){const at=e.get("EXT_texture_filter_anisotropic");o.texParameterf(I,at.TEXTURE_MAX_ANISOTROPY_EXT,Math.min(E.anisotropy,c.getMaxAnisotropy())),r.get(E).__currentAnisotropy=E.anisotropy}}}function gt(I,E){let at=!1;I.__webglInit===void 0&&(I.__webglInit=!0,E.addEventListener("dispose",B));const lt=E.source;let et=M.get(lt);et===void 0&&(et={},M.set(lt,et));const St=Q(E);if(St!==I.__cacheKey){et[St]===void 0&&(et[St]={texture:o.createTexture(),usedTimes:0},d.memory.textures++,at=!0),et[St].usedTimes++;const wt=et[I.__cacheKey];wt!==void 0&&(et[I.__cacheKey].usedTimes--,wt.usedTimes===0&&X(E)),I.__cacheKey=St,I.__webglTexture=et[St].texture}return at}function G(I,E,at){return Math.floor(Math.floor(I/at)/E)}function st(I,E,at,lt){const St=I.updateRanges;if(St.length===0)i.texSubImage2D(o.TEXTURE_2D,0,0,0,E.width,E.height,at,lt,E.data);else{St.sort((Jt,Bt)=>Jt.start-Bt.start);let wt=0;for(let Jt=1;Jt<St.length;Jt++){const Bt=St[wt],It=St[Jt],kt=Bt.start+Bt.count,ie=G(It.start,E.width,4),ce=G(Bt.start,E.width,4);It.start<=kt+1&&ie===ce&&G(It.start+It.count-1,E.width,4)===ie?Bt.count=Math.max(Bt.count,It.start+It.count-Bt.start):(++wt,St[wt]=It)}St.length=wt+1;const _t=i.getParameter(o.UNPACK_ROW_LENGTH),yt=i.getParameter(o.UNPACK_SKIP_PIXELS),Nt=i.getParameter(o.UNPACK_SKIP_ROWS);i.pixelStorei(o.UNPACK_ROW_LENGTH,E.width);for(let Jt=0,Bt=St.length;Jt<Bt;Jt++){const It=St[Jt],kt=Math.floor(It.start/4),ie=Math.ceil(It.count/4),ce=kt%E.width,K=Math.floor(kt/E.width),Ut=ie,bt=1;i.pixelStorei(o.UNPACK_SKIP_PIXELS,ce),i.pixelStorei(o.UNPACK_SKIP_ROWS,K),i.texSubImage2D(o.TEXTURE_2D,0,ce,K,Ut,bt,at,lt,E.data)}I.clearUpdateRanges(),i.pixelStorei(o.UNPACK_ROW_LENGTH,_t),i.pixelStorei(o.UNPACK_SKIP_PIXELS,yt),i.pixelStorei(o.UNPACK_SKIP_ROWS,Nt)}}function At(I,E,at){let lt=o.TEXTURE_2D;(E.isDataArrayTexture||E.isCompressedArrayTexture)&&(lt=o.TEXTURE_2D_ARRAY),E.isData3DTexture&&(lt=o.TEXTURE_3D);const et=gt(I,E),St=E.source;i.bindTexture(lt,I.__webglTexture,o.TEXTURE0+at);const wt=r.get(St);if(St.version!==wt.__version||et===!0){if(i.activeTexture(o.TEXTURE0+at),(typeof ImageBitmap<"u"&&E.image instanceof ImageBitmap)===!1){const bt=Pe.getPrimaries(Pe.workingColorSpace),Lt=E.colorSpace===ys?null:Pe.getPrimaries(E.colorSpace),Vt=E.colorSpace===ys||bt===Lt?o.NONE:o.BROWSER_DEFAULT_WEBGL;i.pixelStorei(o.UNPACK_FLIP_Y_WEBGL,E.flipY),i.pixelStorei(o.UNPACK_PREMULTIPLY_ALPHA_WEBGL,E.premultiplyAlpha),i.pixelStorei(o.UNPACK_COLORSPACE_CONVERSION_WEBGL,Vt)}i.pixelStorei(o.UNPACK_ALIGNMENT,E.unpackAlignment);let yt=y(E.image,!1,c.maxTextureSize);yt=Je(E,yt);const Nt=f.convert(E.format,E.colorSpace),Jt=f.convert(E.type);let Bt=C(E.internalFormat,Nt,Jt,E.normalized,E.colorSpace,E.isVideoTexture);F(lt,E);let It;const kt=E.mipmaps,ie=E.isVideoTexture!==!0,ce=wt.__version===void 0||et===!0,K=St.dataReady,Ut=U(E,yt);if(E.isDepthTexture)Bt=D(E.format===Js,E.type),ce&&(ie?i.texStorage2D(o.TEXTURE_2D,1,Bt,yt.width,yt.height):i.texImage2D(o.TEXTURE_2D,0,Bt,yt.width,yt.height,0,Nt,Jt,null));else if(E.isDataTexture)if(kt.length>0){ie&&ce&&i.texStorage2D(o.TEXTURE_2D,Ut,Bt,kt[0].width,kt[0].height);for(let bt=0,Lt=kt.length;bt<Lt;bt++)It=kt[bt],ie?K&&i.texSubImage2D(o.TEXTURE_2D,bt,0,0,It.width,It.height,Nt,Jt,It.data):i.texImage2D(o.TEXTURE_2D,bt,Bt,It.width,It.height,0,Nt,Jt,It.data);E.generateMipmaps=!1}else ie?(ce&&i.texStorage2D(o.TEXTURE_2D,Ut,Bt,yt.width,yt.height),K&&st(E,yt,Nt,Jt)):i.texImage2D(o.TEXTURE_2D,0,Bt,yt.width,yt.height,0,Nt,Jt,yt.data);else if(E.isCompressedTexture)if(E.isCompressedArrayTexture){ie&&ce&&i.texStorage3D(o.TEXTURE_2D_ARRAY,Ut,Bt,kt[0].width,kt[0].height,yt.depth);for(let bt=0,Lt=kt.length;bt<Lt;bt++)if(It=kt[bt],E.format!==Gi)if(Nt!==null)if(ie){if(K)if(E.layerUpdates.size>0){const Vt=px(It.width,It.height,E.format,E.type);for(const Rt of E.layerUpdates){const $t=It.data.subarray(Rt*Vt/It.data.BYTES_PER_ELEMENT,(Rt+1)*Vt/It.data.BYTES_PER_ELEMENT);i.compressedTexSubImage3D(o.TEXTURE_2D_ARRAY,bt,0,0,Rt,It.width,It.height,1,Nt,$t)}}else i.compressedTexSubImage3D(o.TEXTURE_2D_ARRAY,bt,0,0,0,It.width,It.height,yt.depth,Nt,It.data)}else i.compressedTexImage3D(o.TEXTURE_2D_ARRAY,bt,Bt,It.width,It.height,yt.depth,0,It.data,0,0);else oe("WebGLRenderer: Attempt to load unsupported compressed texture format in .uploadTexture()");else ie?K&&i.texSubImage3D(o.TEXTURE_2D_ARRAY,bt,0,0,0,It.width,It.height,yt.depth,Nt,Jt,It.data):i.texImage3D(o.TEXTURE_2D_ARRAY,bt,Bt,It.width,It.height,yt.depth,0,Nt,Jt,It.data);E.layerUpdates.size>0&&E.clearLayerUpdates()}else{ie&&ce&&i.texStorage2D(o.TEXTURE_2D,Ut,Bt,kt[0].width,kt[0].height);for(let bt=0,Lt=kt.length;bt<Lt;bt++)It=kt[bt],E.format!==Gi?Nt!==null?ie?K&&i.compressedTexSubImage2D(o.TEXTURE_2D,bt,0,0,It.width,It.height,Nt,It.data):i.compressedTexImage2D(o.TEXTURE_2D,bt,Bt,It.width,It.height,0,It.data):oe("WebGLRenderer: Attempt to load unsupported compressed texture format in .uploadTexture()"):ie?K&&i.texSubImage2D(o.TEXTURE_2D,bt,0,0,It.width,It.height,Nt,Jt,It.data):i.texImage2D(o.TEXTURE_2D,bt,Bt,It.width,It.height,0,Nt,Jt,It.data)}else if(E.isDataArrayTexture)if(ie){if(ce&&i.texStorage3D(o.TEXTURE_2D_ARRAY,Ut,Bt,yt.width,yt.height,yt.depth),K)if(E.layerUpdates.size>0){const bt=px(yt.width,yt.height,E.format,E.type);for(const Lt of E.layerUpdates){const Vt=yt.data.subarray(Lt*bt/yt.data.BYTES_PER_ELEMENT,(Lt+1)*bt/yt.data.BYTES_PER_ELEMENT);i.texSubImage3D(o.TEXTURE_2D_ARRAY,0,0,0,Lt,yt.width,yt.height,1,Nt,Jt,Vt)}E.clearLayerUpdates()}else i.texSubImage3D(o.TEXTURE_2D_ARRAY,0,0,0,0,yt.width,yt.height,yt.depth,Nt,Jt,yt.data)}else i.texImage3D(o.TEXTURE_2D_ARRAY,0,Bt,yt.width,yt.height,yt.depth,0,Nt,Jt,yt.data);else if(E.isData3DTexture)ie?(ce&&i.texStorage3D(o.TEXTURE_3D,Ut,Bt,yt.width,yt.height,yt.depth),K&&i.texSubImage3D(o.TEXTURE_3D,0,0,0,0,yt.width,yt.height,yt.depth,Nt,Jt,yt.data)):i.texImage3D(o.TEXTURE_3D,0,Bt,yt.width,yt.height,yt.depth,0,Nt,Jt,yt.data);else if(E.isFramebufferTexture){if(ce)if(ie)i.texStorage2D(o.TEXTURE_2D,Ut,Bt,yt.width,yt.height);else{let bt=yt.width,Lt=yt.height;for(let Vt=0;Vt<Ut;Vt++)i.texImage2D(o.TEXTURE_2D,Vt,Bt,bt,Lt,0,Nt,Jt,null),bt>>=1,Lt>>=1}}else if(E.isHTMLTexture){if("texElementImage2D"in o){const bt=o.canvas;if(bt.hasAttribute("layoutsubtree")||bt.setAttribute("layoutsubtree","true"),yt.parentNode!==bt){bt.appendChild(yt),v.add(E),bt.onpaint=Lt=>{const Vt=Lt.changedElements;for(const Rt of v)Vt.includes(Rt.image)&&(Rt.needsUpdate=!0)},bt.requestPaint();return}if(o.texElementImage2D.length===3)o.texElementImage2D(o.TEXTURE_2D,o.RGBA8,yt);else{const Vt=o.RGBA,Rt=o.RGBA,$t=o.UNSIGNED_BYTE;o.texElementImage2D(o.TEXTURE_2D,0,Vt,Rt,$t,yt)}o.texParameteri(o.TEXTURE_2D,o.TEXTURE_MIN_FILTER,o.LINEAR),o.texParameteri(o.TEXTURE_2D,o.TEXTURE_WRAP_S,o.CLAMP_TO_EDGE),o.texParameteri(o.TEXTURE_2D,o.TEXTURE_WRAP_T,o.CLAMP_TO_EDGE)}}else if(kt.length>0){if(ie&&ce){const bt=Me(kt[0]);i.texStorage2D(o.TEXTURE_2D,Ut,Bt,bt.width,bt.height)}for(let bt=0,Lt=kt.length;bt<Lt;bt++)It=kt[bt],ie?K&&i.texSubImage2D(o.TEXTURE_2D,bt,0,0,Nt,Jt,It):i.texImage2D(o.TEXTURE_2D,bt,Bt,Nt,Jt,It);E.generateMipmaps=!1}else if(ie){if(ce){const bt=Me(yt);i.texStorage2D(o.TEXTURE_2D,Ut,Bt,bt.width,bt.height)}K&&i.texSubImage2D(o.TEXTURE_2D,0,0,0,Nt,Jt,yt)}else i.texImage2D(o.TEXTURE_2D,0,Bt,Nt,Jt,yt);S(E)&&O(lt),wt.__version=St.version,E.onUpdate&&E.onUpdate(E)}I.__version=E.version}function Dt(I,E,at){if(E.image.length!==6)return;const lt=gt(I,E),et=E.source;i.bindTexture(o.TEXTURE_CUBE_MAP,I.__webglTexture,o.TEXTURE0+at);const St=r.get(et);if(et.version!==St.__version||lt===!0){i.activeTexture(o.TEXTURE0+at);const wt=Pe.getPrimaries(Pe.workingColorSpace),_t=E.colorSpace===ys?null:Pe.getPrimaries(E.colorSpace),yt=E.colorSpace===ys||wt===_t?o.NONE:o.BROWSER_DEFAULT_WEBGL;i.pixelStorei(o.UNPACK_FLIP_Y_WEBGL,E.flipY),i.pixelStorei(o.UNPACK_PREMULTIPLY_ALPHA_WEBGL,E.premultiplyAlpha),i.pixelStorei(o.UNPACK_ALIGNMENT,E.unpackAlignment),i.pixelStorei(o.UNPACK_COLORSPACE_CONVERSION_WEBGL,yt);const Nt=E.isCompressedTexture||E.image[0].isCompressedTexture,Jt=E.image[0]&&E.image[0].isDataTexture,Bt=[];for(let Rt=0;Rt<6;Rt++)!Nt&&!Jt?Bt[Rt]=y(E.image[Rt],!0,c.maxCubemapSize):Bt[Rt]=Jt?E.image[Rt].image:E.image[Rt],Bt[Rt]=Je(E,Bt[Rt]);const It=Bt[0],kt=f.convert(E.format,E.colorSpace),ie=f.convert(E.type),ce=C(E.internalFormat,kt,ie,E.normalized,E.colorSpace),K=E.isVideoTexture!==!0,Ut=St.__version===void 0||lt===!0,bt=et.dataReady;let Lt=U(E,It);F(o.TEXTURE_CUBE_MAP,E);let Vt;if(Nt){K&&Ut&&i.texStorage2D(o.TEXTURE_CUBE_MAP,Lt,ce,It.width,It.height);for(let Rt=0;Rt<6;Rt++){Vt=Bt[Rt].mipmaps;for(let $t=0;$t<Vt.length;$t++){const Gt=Vt[$t];E.format!==Gi?kt!==null?K?bt&&i.compressedTexSubImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Rt,$t,0,0,Gt.width,Gt.height,kt,Gt.data):i.compressedTexImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Rt,$t,ce,Gt.width,Gt.height,0,Gt.data):oe("WebGLRenderer: Attempt to load unsupported compressed texture format in .setTextureCube()"):K?bt&&i.texSubImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Rt,$t,0,0,Gt.width,Gt.height,kt,ie,Gt.data):i.texImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Rt,$t,ce,Gt.width,Gt.height,0,kt,ie,Gt.data)}}}else{if(Vt=E.mipmaps,K&&Ut){Vt.length>0&&Lt++;const Rt=Me(Bt[0]);i.texStorage2D(o.TEXTURE_CUBE_MAP,Lt,ce,Rt.width,Rt.height)}for(let Rt=0;Rt<6;Rt++)if(Jt){K?bt&&i.texSubImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Rt,0,0,0,Bt[Rt].width,Bt[Rt].height,kt,ie,Bt[Rt].data):i.texImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Rt,0,ce,Bt[Rt].width,Bt[Rt].height,0,kt,ie,Bt[Rt].data);for(let $t=0;$t<Vt.length;$t++){const Le=Vt[$t].image[Rt].image;K?bt&&i.texSubImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Rt,$t+1,0,0,Le.width,Le.height,kt,ie,Le.data):i.texImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Rt,$t+1,ce,Le.width,Le.height,0,kt,ie,Le.data)}}else{K?bt&&i.texSubImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Rt,0,0,0,kt,ie,Bt[Rt]):i.texImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Rt,0,ce,kt,ie,Bt[Rt]);for(let $t=0;$t<Vt.length;$t++){const Gt=Vt[$t];K?bt&&i.texSubImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Rt,$t+1,0,0,kt,ie,Gt.image[Rt]):i.texImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+Rt,$t+1,ce,kt,ie,Gt.image[Rt])}}}S(E)&&O(o.TEXTURE_CUBE_MAP),St.__version=et.version,E.onUpdate&&E.onUpdate(E)}I.__version=E.version}function xt(I,E,at,lt,et,St){const wt=f.convert(at.format,at.colorSpace),_t=f.convert(at.type),yt=C(at.internalFormat,wt,_t,at.normalized,at.colorSpace),Nt=r.get(E),Jt=r.get(at);if(Jt.__renderTarget=E,!Nt.__hasExternalTextures){const Bt=Math.max(1,E.width>>St),It=Math.max(1,E.height>>St);et===o.TEXTURE_3D||et===o.TEXTURE_2D_ARRAY?i.texImage3D(et,St,yt,Bt,It,E.depth,0,wt,_t,null):i.texImage2D(et,St,yt,Bt,It,0,wt,_t,null)}i.bindFramebuffer(o.FRAMEBUFFER,I),je(E)?h.framebufferTexture2DMultisampleEXT(o.FRAMEBUFFER,lt,et,Jt.__webglTexture,0,Ue(E)):(et===o.TEXTURE_2D||et>=o.TEXTURE_CUBE_MAP_POSITIVE_X&&et<=o.TEXTURE_CUBE_MAP_NEGATIVE_Z)&&o.framebufferTexture2D(o.FRAMEBUFFER,lt,et,Jt.__webglTexture,St),i.bindFramebuffer(o.FRAMEBUFFER,null)}function Ct(I,E,at){if(o.bindRenderbuffer(o.RENDERBUFFER,I),E.depthBuffer){const lt=E.depthTexture,et=lt&&lt.isDepthTexture?lt.type:null,St=D(E.stencilBuffer,et),wt=E.stencilBuffer?o.DEPTH_STENCIL_ATTACHMENT:o.DEPTH_ATTACHMENT;je(E)?h.renderbufferStorageMultisampleEXT(o.RENDERBUFFER,Ue(E),St,E.width,E.height):at?o.renderbufferStorageMultisample(o.RENDERBUFFER,Ue(E),St,E.width,E.height):o.renderbufferStorage(o.RENDERBUFFER,St,E.width,E.height),o.framebufferRenderbuffer(o.FRAMEBUFFER,wt,o.RENDERBUFFER,I)}else{const lt=E.textures;for(let et=0;et<lt.length;et++){const St=lt[et],wt=f.convert(St.format,St.colorSpace),_t=f.convert(St.type),yt=C(St.internalFormat,wt,_t,St.normalized,St.colorSpace);je(E)?h.renderbufferStorageMultisampleEXT(o.RENDERBUFFER,Ue(E),yt,E.width,E.height):at?o.renderbufferStorageMultisample(o.RENDERBUFFER,Ue(E),yt,E.width,E.height):o.renderbufferStorage(o.RENDERBUFFER,yt,E.width,E.height)}}o.bindRenderbuffer(o.RENDERBUFFER,null)}function Fe(I,E,at){const lt=E.isWebGLCubeRenderTarget===!0;if(i.bindFramebuffer(o.FRAMEBUFFER,I),!(E.depthTexture&&E.depthTexture.isDepthTexture))throw new Error("THREE.WebGLTextures: renderTarget.depthTexture must be an instance of THREE.DepthTexture.");const et=r.get(E.depthTexture);if(et.__renderTarget=E,(!et.__webglTexture||E.depthTexture.image.width!==E.width||E.depthTexture.image.height!==E.height)&&(E.depthTexture.image.width=E.width,E.depthTexture.image.height=E.height,E.depthTexture.needsUpdate=!0),lt){if(et.__webglInit===void 0&&(et.__webglInit=!0,E.depthTexture.addEventListener("dispose",B)),et.__webglTexture===void 0){et.__webglTexture=o.createTexture(),i.bindTexture(o.TEXTURE_CUBE_MAP,et.__webglTexture),F(o.TEXTURE_CUBE_MAP,E.depthTexture);const Nt=f.convert(E.depthTexture.format),Jt=f.convert(E.depthTexture.type);let Bt;E.depthTexture.format===za?Bt=o.DEPTH_COMPONENT24:E.depthTexture.format===Js&&(Bt=o.DEPTH24_STENCIL8);for(let It=0;It<6;It++)o.texImage2D(o.TEXTURE_CUBE_MAP_POSITIVE_X+It,0,Bt,E.width,E.height,0,Nt,Jt,null)}}else pt(E.depthTexture,0);const St=et.__webglTexture,wt=Ue(E),_t=lt?o.TEXTURE_CUBE_MAP_POSITIVE_X+at:o.TEXTURE_2D,yt=E.depthTexture.format===Js?o.DEPTH_STENCIL_ATTACHMENT:o.DEPTH_ATTACHMENT;if(E.depthTexture.format===za)je(E)?h.framebufferTexture2DMultisampleEXT(o.FRAMEBUFFER,yt,_t,St,0,wt):o.framebufferTexture2D(o.FRAMEBUFFER,yt,_t,St,0);else if(E.depthTexture.format===Js)je(E)?h.framebufferTexture2DMultisampleEXT(o.FRAMEBUFFER,yt,_t,St,0,wt):o.framebufferTexture2D(o.FRAMEBUFFER,yt,_t,St,0);else throw new Error("THREE.WebGLTextures: Unknown depthTexture format.")}function le(I){const E=r.get(I),at=I.isWebGLCubeRenderTarget===!0;if(E.__boundDepthTexture!==I.depthTexture){const lt=I.depthTexture;if(E.__depthDisposeCallback&&E.__depthDisposeCallback(),lt){const et=()=>{delete E.__boundDepthTexture,delete E.__depthDisposeCallback,lt.removeEventListener("dispose",et)};lt.addEventListener("dispose",et),E.__depthDisposeCallback=et}E.__boundDepthTexture=lt}if(I.depthTexture&&!E.__autoAllocateDepthBuffer)if(at)for(let lt=0;lt<6;lt++)Fe(E.__webglFramebuffer[lt],I,lt);else{const lt=I.texture.mipmaps;lt&&lt.length>0?Fe(E.__webglFramebuffer[0],I,0):Fe(E.__webglFramebuffer,I,0)}else if(at){E.__webglDepthbuffer=[];for(let lt=0;lt<6;lt++)if(i.bindFramebuffer(o.FRAMEBUFFER,E.__webglFramebuffer[lt]),E.__webglDepthbuffer[lt]===void 0)E.__webglDepthbuffer[lt]=o.createRenderbuffer(),Ct(E.__webglDepthbuffer[lt],I,!1);else{const et=I.stencilBuffer?o.DEPTH_STENCIL_ATTACHMENT:o.DEPTH_ATTACHMENT,St=E.__webglDepthbuffer[lt];o.bindRenderbuffer(o.RENDERBUFFER,St),o.framebufferRenderbuffer(o.FRAMEBUFFER,et,o.RENDERBUFFER,St)}}else{const lt=I.texture.mipmaps;if(lt&&lt.length>0?i.bindFramebuffer(o.FRAMEBUFFER,E.__webglFramebuffer[0]):i.bindFramebuffer(o.FRAMEBUFFER,E.__webglFramebuffer),E.__webglDepthbuffer===void 0)E.__webglDepthbuffer=o.createRenderbuffer(),Ct(E.__webglDepthbuffer,I,!1);else{const et=I.stencilBuffer?o.DEPTH_STENCIL_ATTACHMENT:o.DEPTH_ATTACHMENT,St=E.__webglDepthbuffer;o.bindRenderbuffer(o.RENDERBUFFER,St),o.framebufferRenderbuffer(o.FRAMEBUFFER,et,o.RENDERBUFFER,St)}}i.bindFramebuffer(o.FRAMEBUFFER,null)}function pe(I,E,at){const lt=r.get(I);E!==void 0&&xt(lt.__webglFramebuffer,I,I.texture,o.COLOR_ATTACHMENT0,o.TEXTURE_2D,0),at!==void 0&&le(I)}function ve(I){const E=I.texture,at=r.get(I),lt=r.get(E);I.addEventListener("dispose",b);const et=I.textures,St=I.isWebGLCubeRenderTarget===!0,wt=et.length>1;if(wt||(lt.__webglTexture===void 0&&(lt.__webglTexture=o.createTexture()),lt.__version=E.version,d.memory.textures++),St){at.__webglFramebuffer=[];for(let _t=0;_t<6;_t++)if(E.mipmaps&&E.mipmaps.length>0){at.__webglFramebuffer[_t]=[];for(let yt=0;yt<E.mipmaps.length;yt++)at.__webglFramebuffer[_t][yt]=o.createFramebuffer()}else at.__webglFramebuffer[_t]=o.createFramebuffer()}else{if(E.mipmaps&&E.mipmaps.length>0){at.__webglFramebuffer=[];for(let _t=0;_t<E.mipmaps.length;_t++)at.__webglFramebuffer[_t]=o.createFramebuffer()}else at.__webglFramebuffer=o.createFramebuffer();if(wt)for(let _t=0,yt=et.length;_t<yt;_t++){const Nt=r.get(et[_t]);Nt.__webglTexture===void 0&&(Nt.__webglTexture=o.createTexture(),d.memory.textures++)}if(I.samples>0&&je(I)===!1){at.__webglMultisampledFramebuffer=o.createFramebuffer(),at.__webglColorRenderbuffer=[],i.bindFramebuffer(o.FRAMEBUFFER,at.__webglMultisampledFramebuffer);for(let _t=0;_t<et.length;_t++){const yt=et[_t];at.__webglColorRenderbuffer[_t]=o.createRenderbuffer(),o.bindRenderbuffer(o.RENDERBUFFER,at.__webglColorRenderbuffer[_t]);const Nt=f.convert(yt.format,yt.colorSpace),Jt=f.convert(yt.type),Bt=C(yt.internalFormat,Nt,Jt,yt.normalized,yt.colorSpace,I.isXRRenderTarget===!0),It=Ue(I);o.renderbufferStorageMultisample(o.RENDERBUFFER,It,Bt,I.width,I.height),o.framebufferRenderbuffer(o.FRAMEBUFFER,o.COLOR_ATTACHMENT0+_t,o.RENDERBUFFER,at.__webglColorRenderbuffer[_t])}o.bindRenderbuffer(o.RENDERBUFFER,null),I.depthBuffer&&(at.__webglDepthRenderbuffer=o.createRenderbuffer(),Ct(at.__webglDepthRenderbuffer,I,!0)),i.bindFramebuffer(o.FRAMEBUFFER,null)}}if(St){i.bindTexture(o.TEXTURE_CUBE_MAP,lt.__webglTexture),F(o.TEXTURE_CUBE_MAP,E);for(let _t=0;_t<6;_t++)if(E.mipmaps&&E.mipmaps.length>0)for(let yt=0;yt<E.mipmaps.length;yt++)xt(at.__webglFramebuffer[_t][yt],I,E,o.COLOR_ATTACHMENT0,o.TEXTURE_CUBE_MAP_POSITIVE_X+_t,yt);else xt(at.__webglFramebuffer[_t],I,E,o.COLOR_ATTACHMENT0,o.TEXTURE_CUBE_MAP_POSITIVE_X+_t,0);S(E)&&O(o.TEXTURE_CUBE_MAP),i.unbindTexture()}else if(wt){for(let _t=0,yt=et.length;_t<yt;_t++){const Nt=et[_t],Jt=r.get(Nt);let Bt=o.TEXTURE_2D;(I.isWebGL3DRenderTarget||I.isWebGLArrayRenderTarget)&&(Bt=I.isWebGL3DRenderTarget?o.TEXTURE_3D:o.TEXTURE_2D_ARRAY),i.bindTexture(Bt,Jt.__webglTexture),F(Bt,Nt),xt(at.__webglFramebuffer,I,Nt,o.COLOR_ATTACHMENT0+_t,Bt,0),S(Nt)&&O(Bt)}i.unbindTexture()}else{let _t=o.TEXTURE_2D;if((I.isWebGL3DRenderTarget||I.isWebGLArrayRenderTarget)&&(_t=I.isWebGL3DRenderTarget?o.TEXTURE_3D:o.TEXTURE_2D_ARRAY),i.bindTexture(_t,lt.__webglTexture),F(_t,E),E.mipmaps&&E.mipmaps.length>0)for(let yt=0;yt<E.mipmaps.length;yt++)xt(at.__webglFramebuffer[yt],I,E,o.COLOR_ATTACHMENT0,_t,yt);else xt(at.__webglFramebuffer,I,E,o.COLOR_ATTACHMENT0,_t,0);S(E)&&O(_t),i.unbindTexture()}I.depthBuffer&&le(I)}function te(I){const E=I.textures;for(let at=0,lt=E.length;at<lt;at++){const et=E[at];if(S(et)){const St=k(I),wt=r.get(et).__webglTexture;i.bindTexture(St,wt),O(St),i.unbindTexture()}}}const ne=[],Be=[];function sn(I){if(I.samples>0){if(je(I)===!1){const E=I.textures,at=I.width,lt=I.height;let et=o.COLOR_BUFFER_BIT;const St=I.stencilBuffer?o.DEPTH_STENCIL_ATTACHMENT:o.DEPTH_ATTACHMENT,wt=r.get(I),_t=E.length>1;if(_t)for(let Nt=0;Nt<E.length;Nt++)i.bindFramebuffer(o.FRAMEBUFFER,wt.__webglMultisampledFramebuffer),o.framebufferRenderbuffer(o.FRAMEBUFFER,o.COLOR_ATTACHMENT0+Nt,o.RENDERBUFFER,null),i.bindFramebuffer(o.FRAMEBUFFER,wt.__webglFramebuffer),o.framebufferTexture2D(o.DRAW_FRAMEBUFFER,o.COLOR_ATTACHMENT0+Nt,o.TEXTURE_2D,null,0);i.bindFramebuffer(o.READ_FRAMEBUFFER,wt.__webglMultisampledFramebuffer);const yt=I.texture.mipmaps;yt&&yt.length>0?i.bindFramebuffer(o.DRAW_FRAMEBUFFER,wt.__webglFramebuffer[0]):i.bindFramebuffer(o.DRAW_FRAMEBUFFER,wt.__webglFramebuffer);for(let Nt=0;Nt<E.length;Nt++){if(I.resolveDepthBuffer&&(I.depthBuffer&&(et|=o.DEPTH_BUFFER_BIT),I.stencilBuffer&&I.resolveStencilBuffer&&(et|=o.STENCIL_BUFFER_BIT)),_t){o.framebufferRenderbuffer(o.READ_FRAMEBUFFER,o.COLOR_ATTACHMENT0,o.RENDERBUFFER,wt.__webglColorRenderbuffer[Nt]);const Jt=r.get(E[Nt]).__webglTexture;o.framebufferTexture2D(o.DRAW_FRAMEBUFFER,o.COLOR_ATTACHMENT0,o.TEXTURE_2D,Jt,0)}o.blitFramebuffer(0,0,at,lt,0,0,at,lt,et,o.NEAREST),m===!0&&(ne.length=0,Be.length=0,ne.push(o.COLOR_ATTACHMENT0+Nt),I.depthBuffer&&I.storeMultisampledDepthBuffer===!1&&(ne.push(St),Be.push(St),o.invalidateFramebuffer(o.DRAW_FRAMEBUFFER,Be)),o.invalidateFramebuffer(o.READ_FRAMEBUFFER,ne))}if(i.bindFramebuffer(o.READ_FRAMEBUFFER,null),i.bindFramebuffer(o.DRAW_FRAMEBUFFER,null),_t)for(let Nt=0;Nt<E.length;Nt++){i.bindFramebuffer(o.FRAMEBUFFER,wt.__webglMultisampledFramebuffer),o.framebufferRenderbuffer(o.FRAMEBUFFER,o.COLOR_ATTACHMENT0+Nt,o.RENDERBUFFER,wt.__webglColorRenderbuffer[Nt]);const Jt=r.get(E[Nt]).__webglTexture;i.bindFramebuffer(o.FRAMEBUFFER,wt.__webglFramebuffer),o.framebufferTexture2D(o.DRAW_FRAMEBUFFER,o.COLOR_ATTACHMENT0+Nt,o.TEXTURE_2D,Jt,0)}i.bindFramebuffer(o.DRAW_FRAMEBUFFER,wt.__webglMultisampledFramebuffer)}else if(I.depthBuffer&&I.storeMultisampledDepthBuffer===!1&&m){const E=I.stencilBuffer?o.DEPTH_STENCIL_ATTACHMENT:o.DEPTH_ATTACHMENT;o.invalidateFramebuffer(o.DRAW_FRAMEBUFFER,[E])}}}function Ue(I){return Math.min(c.maxSamples,I.samples)}function je(I){const E=r.get(I);return I.samples>0&&e.has("WEBGL_multisampled_render_to_texture")===!0&&E.__useRenderToTexture!==!1}function Z(I){const E=d.render.frame;x.get(I)!==E&&(x.set(I,E),I.update())}function Je(I,E){const at=I.colorSpace,lt=I.format,et=I.type;return I.isCompressedTexture===!0||I.isVideoTexture===!0||at!==Fu&&at!==ys&&(Pe.getTransfer(at)===Ye?(lt!==Gi||et!==pi)&&oe("WebGLTextures: sRGB encoded textures have to use RGBAFormat and UnsignedByteType."):ze("WebGLTextures: Unsupported texture color space:",at)),E}function Me(I){return typeof HTMLImageElement<"u"&&I instanceof HTMLImageElement?(p.width=I.naturalWidth||I.width,p.height=I.naturalHeight||I.height):typeof VideoFrame<"u"&&I instanceof VideoFrame?(p.width=I.displayWidth,p.height=I.displayHeight):(p.width=I.width,p.height=I.height),p}this.allocateTextureUnit=Y,this.resetTextureUnits=P,this.getTextureUnits=q,this.setTextureUnits=$,this.setTexture2D=pt,this.setTexture2DArray=ot,this.setTexture3D=vt,this.setTextureCube=Tt,this.rebindTextures=pe,this.setupRenderTarget=ve,this.updateRenderTargetMipmap=te,this.updateMultisampleRenderTarget=sn,this.setupDepthRenderbuffer=le,this.setupFrameBufferTexture=xt,this.useMultisampledRTT=je,this.isReversedDepthBuffer=function(){return i.buffers.depth.getReversed()}}function Dw(o,e){function i(r,c=ys){let f;const d=Pe.getTransfer(c);if(r===pi)return o.UNSIGNED_BYTE;if(r===nm)return o.UNSIGNED_SHORT_4_4_4_4;if(r===im)return o.UNSIGNED_SHORT_5_5_5_1;if(r===tS)return o.UNSIGNED_INT_5_9_9_9_REV;if(r===eS)return o.UNSIGNED_INT_10F_11F_11F_REV;if(r===Jx)return o.BYTE;if(r===$x)return o.SHORT;if(r===Ll)return o.UNSIGNED_SHORT;if(r===em)return o.INT;if(r===fa)return o.UNSIGNED_INT;if(r===Hi)return o.FLOAT;if(r===da)return o.HALF_FLOAT;if(r===nS)return o.ALPHA;if(r===iS)return o.RGB;if(r===Gi)return o.RGBA;if(r===za)return o.DEPTH_COMPONENT;if(r===Js)return o.DEPTH_STENCIL;if(r===am)return o.RED;if(r===sm)return o.RED_INTEGER;if(r===er)return o.RG;if(r===rm)return o.RG_INTEGER;if(r===om)return o.RGBA_INTEGER;if(r===Du||r===Uu||r===Lu||r===Ou)if(d===Ye)if(f=e.get("WEBGL_compressed_texture_s3tc_srgb"),f!==null){if(r===Du)return f.COMPRESSED_SRGB_S3TC_DXT1_EXT;if(r===Uu)return f.COMPRESSED_SRGB_ALPHA_S3TC_DXT1_EXT;if(r===Lu)return f.COMPRESSED_SRGB_ALPHA_S3TC_DXT3_EXT;if(r===Ou)return f.COMPRESSED_SRGB_ALPHA_S3TC_DXT5_EXT}else return null;else if(f=e.get("WEBGL_compressed_texture_s3tc"),f!==null){if(r===Du)return f.COMPRESSED_RGB_S3TC_DXT1_EXT;if(r===Uu)return f.COMPRESSED_RGBA_S3TC_DXT1_EXT;if(r===Lu)return f.COMPRESSED_RGBA_S3TC_DXT3_EXT;if(r===Ou)return f.COMPRESSED_RGBA_S3TC_DXT5_EXT}else return null;if(r===gp||r===vp||r===_p||r===xp)if(f=e.get("WEBGL_compressed_texture_pvrtc"),f!==null){if(r===gp)return f.COMPRESSED_RGB_PVRTC_4BPPV1_IMG;if(r===vp)return f.COMPRESSED_RGB_PVRTC_2BPPV1_IMG;if(r===_p)return f.COMPRESSED_RGBA_PVRTC_4BPPV1_IMG;if(r===xp)return f.COMPRESSED_RGBA_PVRTC_2BPPV1_IMG}else return null;if(r===Sp||r===yp||r===Mp||r===Ep||r===bp||r===Iu||r===Tp)if(f=e.get("WEBGL_compressed_texture_etc"),f!==null){if(r===Sp||r===yp)return d===Ye?f.COMPRESSED_SRGB8_ETC2:f.COMPRESSED_RGB8_ETC2;if(r===Mp)return d===Ye?f.COMPRESSED_SRGB8_ALPHA8_ETC2_EAC:f.COMPRESSED_RGBA8_ETC2_EAC;if(r===Ep)return f.COMPRESSED_R11_EAC;if(r===bp)return f.COMPRESSED_SIGNED_R11_EAC;if(r===Iu)return f.COMPRESSED_RG11_EAC;if(r===Tp)return f.COMPRESSED_SIGNED_RG11_EAC}else return null;if(r===Ap||r===Rp||r===wp||r===Cp||r===Np||r===Dp||r===Up||r===Lp||r===Op||r===Pp||r===Ip||r===Bp||r===zp||r===Fp)if(f=e.get("WEBGL_compressed_texture_astc"),f!==null){if(r===Ap)return d===Ye?f.COMPRESSED_SRGB8_ALPHA8_ASTC_4x4_KHR:f.COMPRESSED_RGBA_ASTC_4x4_KHR;if(r===Rp)return d===Ye?f.COMPRESSED_SRGB8_ALPHA8_ASTC_5x4_KHR:f.COMPRESSED_RGBA_ASTC_5x4_KHR;if(r===wp)return d===Ye?f.COMPRESSED_SRGB8_ALPHA8_ASTC_5x5_KHR:f.COMPRESSED_RGBA_ASTC_5x5_KHR;if(r===Cp)return d===Ye?f.COMPRESSED_SRGB8_ALPHA8_ASTC_6x5_KHR:f.COMPRESSED_RGBA_ASTC_6x5_KHR;if(r===Np)return d===Ye?f.COMPRESSED_SRGB8_ALPHA8_ASTC_6x6_KHR:f.COMPRESSED_RGBA_ASTC_6x6_KHR;if(r===Dp)return d===Ye?f.COMPRESSED_SRGB8_ALPHA8_ASTC_8x5_KHR:f.COMPRESSED_RGBA_ASTC_8x5_KHR;if(r===Up)return d===Ye?f.COMPRESSED_SRGB8_ALPHA8_ASTC_8x6_KHR:f.COMPRESSED_RGBA_ASTC_8x6_KHR;if(r===Lp)return d===Ye?f.COMPRESSED_SRGB8_ALPHA8_ASTC_8x8_KHR:f.COMPRESSED_RGBA_ASTC_8x8_KHR;if(r===Op)return d===Ye?f.COMPRESSED_SRGB8_ALPHA8_ASTC_10x5_KHR:f.COMPRESSED_RGBA_ASTC_10x5_KHR;if(r===Pp)return d===Ye?f.COMPRESSED_SRGB8_ALPHA8_ASTC_10x6_KHR:f.COMPRESSED_RGBA_ASTC_10x6_KHR;if(r===Ip)return d===Ye?f.COMPRESSED_SRGB8_ALPHA8_ASTC_10x8_KHR:f.COMPRESSED_RGBA_ASTC_10x8_KHR;if(r===Bp)return d===Ye?f.COMPRESSED_SRGB8_ALPHA8_ASTC_10x10_KHR:f.COMPRESSED_RGBA_ASTC_10x10_KHR;if(r===zp)return d===Ye?f.COMPRESSED_SRGB8_ALPHA8_ASTC_12x10_KHR:f.COMPRESSED_RGBA_ASTC_12x10_KHR;if(r===Fp)return d===Ye?f.COMPRESSED_SRGB8_ALPHA8_ASTC_12x12_KHR:f.COMPRESSED_RGBA_ASTC_12x12_KHR}else return null;if(r===Hp||r===Gp||r===Vp)if(f=e.get("EXT_texture_compression_bptc"),f!==null){if(r===Hp)return d===Ye?f.COMPRESSED_SRGB_ALPHA_BPTC_UNORM_EXT:f.COMPRESSED_RGBA_BPTC_UNORM_EXT;if(r===Gp)return f.COMPRESSED_RGB_BPTC_SIGNED_FLOAT_EXT;if(r===Vp)return f.COMPRESSED_RGB_BPTC_UNSIGNED_FLOAT_EXT}else return null;if(r===kp||r===Xp||r===Bu||r===Wp)if(f=e.get("EXT_texture_compression_rgtc"),f!==null){if(r===kp)return f.COMPRESSED_RED_RGTC1_EXT;if(r===Xp)return f.COMPRESSED_SIGNED_RED_RGTC1_EXT;if(r===Bu)return f.COMPRESSED_RED_GREEN_RGTC2_EXT;if(r===Wp)return f.COMPRESSED_SIGNED_RED_GREEN_RGTC2_EXT}else return null;return r===Ol?o.UNSIGNED_INT_24_8:o[r]!==void 0?o[r]:null}return{convert:i}}const Uw=`
void main() {

	gl_Position = vec4( position, 1.0 );

}`,Lw=`
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

}`;class Ow{constructor(){this.texture=null,this.mesh=null,this.depthNear=0,this.depthFar=0}init(e,i){if(this.texture===null){const r=new dS(e.texture);(e.depthNear!==i.depthNear||e.depthFar!==i.depthFar)&&(this.depthNear=e.depthNear,this.depthFar=e.depthFar),this.texture=r}}getMesh(e){if(this.texture!==null&&this.mesh===null){const i=e.cameras[0].viewport,r=new ha({vertexShader:Uw,fragmentShader:Lw,uniforms:{depthColor:{value:this.texture},depthWidth:{value:i.z},depthHeight:{value:i.w}}});this.mesh=new _n(new go(20,20),r)}return this.mesh}reset(){this.texture=null,this.mesh=null}getDepthTexture(){return this.texture}}class Pw extends nr{constructor(e,i){super();const r=this;let c=null,f=1,d=null,h="local-floor",m=1,p=null,x=null,v=null,_=null,M=null,R=null;const N=typeof XRWebGLBinding<"u",y=new Ow,S={},O=i.getContextAttributes();let k=null,C=null;const D=[],U=[],B=new Re;let b=null,L=null;const X=new ei;X.viewport=new on;const j=new ei;j.viewport=new on;const J=[X,j],P=new kb;let q=null,$=null;this.cameraAutoUpdate=!0,this.enabled=!1,this.isPresenting=!1,this.getController=function(G){let st=D[G];return st===void 0&&(st=new Bh,D[G]=st),st.getTargetRaySpace()},this.getControllerGrip=function(G){let st=D[G];return st===void 0&&(st=new Bh,D[G]=st),st.getGripSpace()},this.getHand=function(G){let st=D[G];return st===void 0&&(st=new Bh,D[G]=st),st.getHandSpace()};function Y(G){const st=U.indexOf(G.inputSource);if(st===-1)return;const At=D[st];At!==void 0&&(At.update(G.inputSource,G.frame,p||d),At.dispatchEvent({type:G.type,data:G.inputSource}))}function Q(){c.removeEventListener("select",Y),c.removeEventListener("selectstart",Y),c.removeEventListener("selectend",Y),c.removeEventListener("squeeze",Y),c.removeEventListener("squeezestart",Y),c.removeEventListener("squeezeend",Y),c.removeEventListener("end",Q),c.removeEventListener("inputsourceschange",pt);for(let G=0;G<D.length;G++){const st=U[G];st!==null&&(U[G]=null,D[G].disconnect(st))}q=null,$=null,y.reset();for(const G in S)delete S[G];if(e.setRenderTarget(k),M=null,_=null,v=null,c=null,C=null,gt.stop(),r.isPresenting=!1,e.setPixelRatio(b),e.setSize(B.width,B.height,!1),L!==null){const G=L.camera;G.fov=L.fov,G.zoom=L.zoom,G.updateProjectionMatrix(),L=null}r.dispatchEvent({type:"sessionend"})}this.setFramebufferScaleFactor=function(G){f=G,r.isPresenting===!0&&oe("WebXRManager: Cannot change framebuffer scale while presenting.")},this.setReferenceSpaceType=function(G){h=G,r.isPresenting===!0&&oe("WebXRManager: Cannot change reference space type while presenting.")},this.getReferenceSpace=function(){return p||d},this.setReferenceSpace=function(G){p=G},this.getBaseLayer=function(){return _!==null?_:M},this.getBinding=function(){return v===null&&N&&(v=new XRWebGLBinding(c,i)),v},this.getFrame=function(){return R},this.getSession=function(){return c},this.setSession=async function(G){if(c=G,c!==null){if(k=e.getRenderTarget(),c.addEventListener("select",Y),c.addEventListener("selectstart",Y),c.addEventListener("selectend",Y),c.addEventListener("squeeze",Y),c.addEventListener("squeezestart",Y),c.addEventListener("squeezeend",Y),c.addEventListener("end",Q),c.addEventListener("inputsourceschange",pt),O.xrCompatible!==!0&&await i.makeXRCompatible(),b=e.getPixelRatio(),e.getSize(B),N&&"createProjectionLayer"in XRWebGLBinding.prototype){let At=null,Dt=null,xt=null;O.depth&&(xt=O.stencil?i.DEPTH24_STENCIL8:i.DEPTH_COMPONENT24,At=O.stencil?Js:za,Dt=O.stencil?Ol:fa);const Ct={colorFormat:i.RGBA8,depthFormat:xt,scaleFactor:f};v=this.getBinding(),_=v.createProjectionLayer(Ct),c.updateRenderState({layers:[_]}),e.setPixelRatio(1),e.setSize(_.textureWidth,_.textureHeight,!1),C=new Vi(_.textureWidth,_.textureHeight,{format:Gi,type:pi,depthTexture:new Il(_.textureWidth,_.textureHeight,Dt,void 0,void 0,void 0,void 0,void 0,void 0,At),stencilBuffer:O.stencil,colorSpace:e.outputColorSpace,samples:O.antialias?4:0,resolveDepthBuffer:_.ignoreDepthValues===!1,resolveStencilBuffer:_.ignoreDepthValues===!1,storeMultisampledDepthBuffer:_.ignoreDepthValues===!1,storeMultisampledStencilBuffer:_.ignoreDepthValues===!1})}else{const At={antialias:O.antialias,alpha:!0,depth:O.depth,stencil:O.stencil,framebufferScaleFactor:f};M=new XRWebGLLayer(c,i,At),c.updateRenderState({baseLayer:M}),e.setPixelRatio(1),e.setSize(M.framebufferWidth,M.framebufferHeight,!1),C=new Vi(M.framebufferWidth,M.framebufferHeight,{format:Gi,type:pi,colorSpace:e.outputColorSpace,stencilBuffer:O.stencil,resolveDepthBuffer:M.ignoreDepthValues===!1,resolveStencilBuffer:M.ignoreDepthValues===!1,storeMultisampledDepthBuffer:M.ignoreDepthValues===!1,storeMultisampledStencilBuffer:M.ignoreDepthValues===!1})}C.isXRRenderTarget=!0,this.setFoveation(m),p=null,d=await c.requestReferenceSpace(h),gt.setContext(c),gt.start(),r.isPresenting=!0,r.dispatchEvent({type:"sessionstart"})}},this.getEnvironmentBlendMode=function(){if(c!==null)return c.environmentBlendMode},this.getDepthTexture=function(){return y.getDepthTexture()};function pt(G){for(let st=0;st<G.removed.length;st++){const At=G.removed[st],Dt=U.indexOf(At);Dt>=0&&(U[Dt]=null,D[Dt].disconnect(At))}for(let st=0;st<G.added.length;st++){const At=G.added[st];let Dt=U.indexOf(At);if(Dt===-1){for(let Ct=0;Ct<D.length;Ct++)if(Ct>=U.length){U.push(At),Dt=Ct;break}else if(U[Ct]===null){U[Ct]=At,Dt=Ct;break}if(Dt===-1)break}const xt=D[Dt];xt&&xt.connect(At)}}const ot=new dt,vt=new dt;function Tt(G,st,At){ot.setFromMatrixPosition(st.matrixWorld),vt.setFromMatrixPosition(At.matrixWorld);const Dt=ot.distanceTo(vt),xt=st.projectionMatrix.elements,Ct=At.projectionMatrix.elements,Fe=xt[14]/(xt[10]-1),le=xt[14]/(xt[10]+1),pe=(xt[9]+1)/xt[5],ve=(xt[9]-1)/xt[5],te=(xt[8]-1)/xt[0],ne=(Ct[8]+1)/Ct[0],Be=Fe*te,sn=Fe*ne,Ue=Dt/(-te+ne),je=Ue*-te;if(st.matrixWorld.decompose(G.position,G.quaternion,G.scale),G.translateX(je),G.translateZ(Ue),G.matrixWorld.compose(G.position,G.quaternion,G.scale),G.matrixWorldInverse.copy(G.matrixWorld).invert(),xt[10]===-1)G.projectionMatrix.copy(st.projectionMatrix),G.projectionMatrixInverse.copy(st.projectionMatrixInverse);else{const Z=Fe+Ue,Je=le+Ue,Me=Be-je,I=sn+(Dt-je),E=pe*le/Je*Z,at=ve*le/Je*Z;G.projectionMatrix.makePerspective(Me,I,E,at,Z,Je),G.projectionMatrixInverse.copy(G.projectionMatrix).invert()}}function Zt(G,st){st===null?G.matrixWorld.copy(G.matrix):G.matrixWorld.multiplyMatrices(st.matrixWorld,G.matrix),G.matrixWorldInverse.copy(G.matrixWorld).invert()}this.updateCamera=function(G){if(c===null)return;let st=G.near,At=G.far;y.texture!==null&&(y.depthNear>0&&(st=y.depthNear),y.depthFar>0&&(At=y.depthFar)),P.near=j.near=X.near=st,P.far=j.far=X.far=At,(q!==P.near||$!==P.far)&&(c.updateRenderState({depthNear:P.near,depthFar:P.far}),q=P.near,$=P.far),P.layers.mask=G.layers.mask|6,X.layers.mask=P.layers.mask&-5,j.layers.mask=P.layers.mask&-3;const Dt=G.parent,xt=P.cameras;Zt(P,Dt);for(let Ct=0;Ct<xt.length;Ct++)Zt(xt[Ct],Dt);xt.length===2?Tt(P,X,j):P.projectionMatrix.copy(X.projectionMatrix),L===null&&G.isPerspectiveCamera&&(L={camera:G,fov:G.fov,zoom:G.zoom}),Yt(G,P,Dt)};function Yt(G,st,At){At===null?G.matrix.copy(st.matrixWorld):(G.matrix.copy(At.matrixWorld),G.matrix.invert(),G.matrix.multiply(st.matrixWorld)),G.matrix.decompose(G.position,G.quaternion,G.scale),G.updateMatrixWorld(!0),G.projectionMatrix.copy(st.projectionMatrix),G.projectionMatrixInverse.copy(st.projectionMatrixInverse),G.isPerspectiveCamera&&(G.fov=qp*2*Math.atan(1/G.projectionMatrix.elements[5]),G.zoom=1)}this.getCamera=function(){return P},this.getFoveation=function(){if(!(_===null&&M===null))return m},this.setFoveation=function(G){m=G,_!==null&&(_.fixedFoveation=G),M!==null&&M.fixedFoveation!==void 0&&(M.fixedFoveation=G)},this.hasDepthSensing=function(){return y.texture!==null},this.getDepthSensingMesh=function(){return y.getMesh(P)},this.getCameraTexture=function(G){return S[G]};let H=null;function F(G,st){if(x=st.getViewerPose(p||d),R=st,x!==null){const At=x.views;M!==null&&(e.setRenderTargetFramebuffer(C,M.framebuffer),e.setRenderTarget(C));let Dt=!1;At.length!==P.cameras.length&&(P.cameras.length=0,Dt=!0);for(let le=0;le<At.length;le++){const pe=At[le];let ve=null;if(M!==null)ve=M.getViewport(pe);else{const ne=v.getViewSubImage(_,pe);ve=ne.viewport,le===0&&(e.setRenderTargetTextures(C,ne.colorTexture,ne.depthStencilTexture),e.setRenderTarget(C))}let te=J[le];te===void 0&&(te=new ei,te.layers.enable(le),te.viewport=new on,J[le]=te),te.matrix.fromArray(pe.transform.matrix),te.matrix.decompose(te.position,te.quaternion,te.scale),te.projectionMatrix.fromArray(pe.projectionMatrix),te.projectionMatrixInverse.copy(te.projectionMatrix).invert(),te.viewport.set(ve.x,ve.y,ve.width,ve.height),le===0&&(P.matrix.copy(te.matrix),P.matrix.decompose(P.position,P.quaternion,P.scale)),Dt===!0&&P.cameras.push(te)}const xt=c.enabledFeatures;if(xt&&xt.includes("depth-sensing")&&c.depthUsage=="gpu-optimized"&&N){v=r.getBinding();const le=v.getDepthInformation(At[0]);le&&le.isValid&&le.texture&&y.init(le,c.renderState)}if(xt&&xt.includes("camera-access")&&N){e.state.unbindTexture(),v=r.getBinding();for(let le=0;le<At.length;le++){const pe=At[le].camera;if(pe){let ve=S[pe];ve||(ve=new dS,S[pe]=ve);const te=v.getCameraImage(pe);ve.sourceTexture=te}}}}for(let At=0;At<D.length;At++){const Dt=U[At],xt=D[At];Dt!==null&&xt!==void 0&&xt.update(Dt,st,p||d)}H&&H(G,st),st.detectedPlanes&&r.dispatchEvent({type:"planesdetected",data:st}),R=null}const gt=new gS;gt.setAnimationLoop(F),this.setAnimationLoop=function(G){H=G},this.dispose=function(){}}}const Iw=new nn,ES=new de;ES.set(-1,0,0,0,1,0,0,0,1);function Bw(o,e){function i(y,S){y.matrixAutoUpdate===!0&&y.updateMatrix(),S.value.copy(y.matrix)}function r(y,S){S.color.getRGB(y.fogColor.value,hS(o)),S.isFog?(y.fogNear.value=S.near,y.fogFar.value=S.far):S.isFogExp2&&(y.fogDensity.value=S.density)}function c(y,S,O,k,C){S.isNodeMaterial?S.uniformsNeedUpdate=!1:S.isMeshBasicMaterial?f(y,S):S.isMeshLambertMaterial?(f(y,S),S.envMap&&(y.envMapIntensity.value=S.envMapIntensity)):S.isMeshToonMaterial?(f(y,S),v(y,S)):S.isMeshPhongMaterial?(f(y,S),x(y,S),S.envMap&&(y.envMapIntensity.value=S.envMapIntensity)):S.isMeshStandardMaterial?(f(y,S),_(y,S),S.isMeshPhysicalMaterial&&M(y,S,C)):S.isMeshMatcapMaterial?(f(y,S),R(y,S)):S.isMeshDepthMaterial?f(y,S):S.isMeshDistanceMaterial?(f(y,S),N(y,S)):S.isMeshNormalMaterial?f(y,S):S.isLineBasicMaterial?(d(y,S),S.isLineDashedMaterial&&h(y,S)):S.isPointsMaterial?m(y,S,O,k):S.isSpriteMaterial?p(y,S):S.isShadowMaterial?(y.color.value.copy(S.color),y.opacity.value=S.opacity):S.isShaderMaterial&&(S.uniformsNeedUpdate=!1)}function f(y,S){y.opacity.value=S.opacity,S.color&&y.diffuse.value.copy(S.color),S.emissive&&y.emissive.value.copy(S.emissive).multiplyScalar(S.emissiveIntensity),S.map&&(y.map.value=S.map,i(S.map,y.mapTransform)),S.alphaMap&&(y.alphaMap.value=S.alphaMap,i(S.alphaMap,y.alphaMapTransform)),S.bumpMap&&(y.bumpMap.value=S.bumpMap,i(S.bumpMap,y.bumpMapTransform),y.bumpScale.value=S.bumpScale,S.side===qn&&(y.bumpScale.value*=-1)),S.normalMap&&(y.normalMap.value=S.normalMap,i(S.normalMap,y.normalMapTransform),y.normalScale.value.copy(S.normalScale),S.side===qn&&y.normalScale.value.negate()),S.displacementMap&&(y.displacementMap.value=S.displacementMap,i(S.displacementMap,y.displacementMapTransform),y.displacementScale.value=S.displacementScale,y.displacementBias.value=S.displacementBias),S.emissiveMap&&(y.emissiveMap.value=S.emissiveMap,i(S.emissiveMap,y.emissiveMapTransform)),S.specularMap&&(y.specularMap.value=S.specularMap,i(S.specularMap,y.specularMapTransform)),S.alphaTest>0&&(y.alphaTest.value=S.alphaTest);const O=e.get(S),k=O.envMap,C=O.envMapRotation;k&&(y.envMap.value=k,y.envMapRotation.value.setFromMatrix4(Iw.makeRotationFromEuler(C)).transpose(),k.isCubeTexture&&k.isRenderTargetTexture===!1&&y.envMapRotation.value.premultiply(ES),y.reflectivity.value=S.reflectivity,y.ior.value=S.ior,y.refractionRatio.value=S.refractionRatio),S.lightMap&&(y.lightMap.value=S.lightMap,y.lightMapIntensity.value=S.lightMapIntensity,i(S.lightMap,y.lightMapTransform)),S.aoMap&&(y.aoMap.value=S.aoMap,y.aoMapIntensity.value=S.aoMapIntensity,i(S.aoMap,y.aoMapTransform))}function d(y,S){y.diffuse.value.copy(S.color),y.opacity.value=S.opacity,S.map&&(y.map.value=S.map,i(S.map,y.mapTransform))}function h(y,S){y.dashSize.value=S.dashSize,y.totalSize.value=S.dashSize+S.gapSize,y.scale.value=S.scale}function m(y,S,O,k){y.diffuse.value.copy(S.color),y.opacity.value=S.opacity,y.size.value=S.size*O,y.scale.value=k*.5,S.map&&(y.map.value=S.map,i(S.map,y.uvTransform)),S.alphaMap&&(y.alphaMap.value=S.alphaMap,i(S.alphaMap,y.alphaMapTransform)),S.alphaTest>0&&(y.alphaTest.value=S.alphaTest)}function p(y,S){y.diffuse.value.copy(S.color),y.opacity.value=S.opacity,y.rotation.value=S.rotation,S.map&&(y.map.value=S.map,i(S.map,y.mapTransform)),S.alphaMap&&(y.alphaMap.value=S.alphaMap,i(S.alphaMap,y.alphaMapTransform)),S.alphaTest>0&&(y.alphaTest.value=S.alphaTest)}function x(y,S){y.specular.value.copy(S.specular),y.shininess.value=Math.max(S.shininess,1e-4)}function v(y,S){S.gradientMap&&(y.gradientMap.value=S.gradientMap)}function _(y,S){y.metalness.value=S.metalness,S.metalnessMap&&(y.metalnessMap.value=S.metalnessMap,i(S.metalnessMap,y.metalnessMapTransform)),y.roughness.value=S.roughness,S.roughnessMap&&(y.roughnessMap.value=S.roughnessMap,i(S.roughnessMap,y.roughnessMapTransform)),S.envMap&&(y.envMapIntensity.value=S.envMapIntensity)}function M(y,S,O){y.ior.value=S.ior,S.sheen>0&&(y.sheenColor.value.copy(S.sheenColor).multiplyScalar(S.sheen),y.sheenRoughness.value=S.sheenRoughness,S.sheenColorMap&&(y.sheenColorMap.value=S.sheenColorMap,i(S.sheenColorMap,y.sheenColorMapTransform)),S.sheenRoughnessMap&&(y.sheenRoughnessMap.value=S.sheenRoughnessMap,i(S.sheenRoughnessMap,y.sheenRoughnessMapTransform))),S.clearcoat>0&&(y.clearcoat.value=S.clearcoat,y.clearcoatRoughness.value=S.clearcoatRoughness,S.clearcoatMap&&(y.clearcoatMap.value=S.clearcoatMap,i(S.clearcoatMap,y.clearcoatMapTransform)),S.clearcoatRoughnessMap&&(y.clearcoatRoughnessMap.value=S.clearcoatRoughnessMap,i(S.clearcoatRoughnessMap,y.clearcoatRoughnessMapTransform)),S.clearcoatNormalMap&&(y.clearcoatNormalMap.value=S.clearcoatNormalMap,i(S.clearcoatNormalMap,y.clearcoatNormalMapTransform),y.clearcoatNormalScale.value.copy(S.clearcoatNormalScale),S.side===qn&&y.clearcoatNormalScale.value.negate())),S.dispersion>0&&(y.dispersion.value=S.dispersion),S.retroreflectivity>0&&(y.retroreflectivity.value=S.retroreflectivity),S.iridescence>0&&(y.iridescence.value=S.iridescence,y.iridescenceIOR.value=S.iridescenceIOR,y.iridescenceThicknessMinimum.value=S.iridescenceThicknessRange[0],y.iridescenceThicknessMaximum.value=S.iridescenceThicknessRange[1],S.iridescenceMap&&(y.iridescenceMap.value=S.iridescenceMap,i(S.iridescenceMap,y.iridescenceMapTransform)),S.iridescenceThicknessMap&&(y.iridescenceThicknessMap.value=S.iridescenceThicknessMap,i(S.iridescenceThicknessMap,y.iridescenceThicknessMapTransform))),S.transmission>0&&(y.transmission.value=S.transmission,y.transmissionSamplerMap.value=O.texture,y.transmissionSamplerSize.value.set(O.width,O.height),S.transmissionMap&&(y.transmissionMap.value=S.transmissionMap,i(S.transmissionMap,y.transmissionMapTransform)),y.thickness.value=S.thickness,S.thicknessMap&&(y.thicknessMap.value=S.thicknessMap,i(S.thicknessMap,y.thicknessMapTransform)),y.attenuationDistance.value=S.attenuationDistance,y.attenuationColor.value.copy(S.attenuationColor)),S.anisotropy>0&&(y.anisotropyVector.value.set(S.anisotropy*Math.cos(S.anisotropyRotation),S.anisotropy*Math.sin(S.anisotropyRotation)),S.anisotropyMap&&(y.anisotropyMap.value=S.anisotropyMap,i(S.anisotropyMap,y.anisotropyMapTransform))),y.specularIntensity.value=S.specularIntensity,y.specularColor.value.copy(S.specularColor),S.specularColorMap&&(y.specularColorMap.value=S.specularColorMap,i(S.specularColorMap,y.specularColorMapTransform)),S.specularIntensityMap&&(y.specularIntensityMap.value=S.specularIntensityMap,i(S.specularIntensityMap,y.specularIntensityMapTransform))}function R(y,S){S.matcap&&(y.matcap.value=S.matcap)}function N(y,S){const O=e.get(S).light;y.referencePosition.value.setFromMatrixPosition(O.matrixWorld),y.nearDistance.value=O.shadow.camera.near,y.farDistance.value=O.shadow.camera.far}return{refreshFogUniforms:r,refreshMaterialUniforms:c}}function zw(o,e,i,r){let c={},f={},d=[];const h=o.getParameter(o.MAX_UNIFORM_BUFFER_BINDINGS);function m(C,D){const U=D.program;r.uniformBlockBinding(C,U)}function p(C,D){let U=c[C.id];U===void 0&&(y(C),U=x(C),c[C.id]=U,C.addEventListener("dispose",O));const B=D.program;r.updateUBOMapping(C,B);const b=e.render.frame;f[C.id]!==b&&(_(C),f[C.id]=b)}function x(C){const D=v();C.__bindingPointIndex=D;const U=o.createBuffer(),B=C.__size,b=C.usage;return o.bindBuffer(o.UNIFORM_BUFFER,U),o.bufferData(o.UNIFORM_BUFFER,B,b),o.bindBuffer(o.UNIFORM_BUFFER,null),o.bindBufferBase(o.UNIFORM_BUFFER,D,U),U}function v(){for(let C=0;C<h;C++)if(d.indexOf(C)===-1)return d.push(C),C;return ze("WebGLRenderer: Maximum number of simultaneously usable uniforms groups reached."),0}function _(C){const D=c[C.id],U=C.uniforms,B=C.__cache;o.bindBuffer(o.UNIFORM_BUFFER,D);for(let b=0,L=U.length;b<L;b++){const X=U[b];if(Array.isArray(X))for(let j=0,J=X.length;j<J;j++)M(X[j],b,j,B);else M(X,b,0,B)}o.bindBuffer(o.UNIFORM_BUFFER,null)}function M(C,D,U,B){if(N(C,D,U,B)===!0){const b=C.__offset,L=C.value;if(Array.isArray(L)){let X=0;for(let j=0;j<L.length;j++){const J=L[j],P=S(J);R(J,C.__data,X),typeof J!="number"&&typeof J!="boolean"&&!J.isMatrix3&&!ArrayBuffer.isView(J)&&(X+=P.storage/Float32Array.BYTES_PER_ELEMENT)}}else R(L,C.__data,0);o.bufferSubData(o.UNIFORM_BUFFER,b,C.__data)}}function R(C,D,U){typeof C=="number"||typeof C=="boolean"?D[0]=C:C.isMatrix3?(D[0]=C.elements[0],D[1]=C.elements[1],D[2]=C.elements[2],D[3]=0,D[4]=C.elements[3],D[5]=C.elements[4],D[6]=C.elements[5],D[7]=0,D[8]=C.elements[6],D[9]=C.elements[7],D[10]=C.elements[8],D[11]=0):ArrayBuffer.isView(C)?D.set(new C.constructor(C.buffer,C.byteOffset,D.length)):C.toArray(D,U)}function N(C,D,U,B){const b=C.value,L=D+"_"+U;if(B[L]===void 0)return typeof b=="number"||typeof b=="boolean"?B[L]=b:ArrayBuffer.isView(b)?B[L]=b.slice():B[L]=b.clone(),!0;{const X=B[L];if(typeof b=="number"||typeof b=="boolean"){if(X!==b)return B[L]=b,!0}else{if(ArrayBuffer.isView(b))return!0;if(X.equals(b)===!1)return X.copy(b),!0}}return!1}function y(C){const D=C.uniforms;let U=0;const B=16;for(let L=0,X=D.length;L<X;L++){const j=Array.isArray(D[L])?D[L]:[D[L]];for(let J=0,P=j.length;J<P;J++){const q=j[J],$=Array.isArray(q.value)?q.value:[q.value];for(let Y=0,Q=$.length;Y<Q;Y++){const pt=$[Y],ot=S(pt),vt=U%B,Tt=vt%ot.boundary,Zt=vt+Tt;U+=Tt,Zt!==0&&B-Zt<ot.storage&&(U+=B-Zt),q.__data=new Float32Array(ot.storage/Float32Array.BYTES_PER_ELEMENT),q.__offset=U,U+=ot.storage}}}const b=U%B;return b>0&&(U+=B-b),C.__size=U,C.__cache={},this}function S(C){const D={boundary:0,storage:0};return typeof C=="number"||typeof C=="boolean"?(D.boundary=4,D.storage=4):C.isVector2?(D.boundary=8,D.storage=8):C.isVector3||C.isColor?(D.boundary=16,D.storage=12):C.isVector4?(D.boundary=16,D.storage=16):C.isMatrix3?(D.boundary=48,D.storage=48):C.isMatrix4?(D.boundary=64,D.storage=64):C.isTexture?oe("WebGLRenderer: Texture samplers can not be part of an uniforms group."):ArrayBuffer.isView(C)?(D.boundary=16,D.storage=C.byteLength):oe("WebGLRenderer: Unsupported uniform value type.",C),D}function O(C){const D=C.target;D.removeEventListener("dispose",O);const U=d.indexOf(D.__bindingPointIndex);d.splice(U,1),o.deleteBuffer(c[D.id]),delete c[D.id],delete f[D.id]}function k(){for(const C in c)o.deleteBuffer(c[C]);d=[],c={},f={}}return{bind:m,update:p,dispose:k}}const Fw=new Uint16Array([12469,15057,12620,14925,13266,14620,13807,14376,14323,13990,14545,13625,14713,13328,14840,12882,14931,12528,14996,12233,15039,11829,15066,11525,15080,11295,15085,10976,15082,10705,15073,10495,13880,14564,13898,14542,13977,14430,14158,14124,14393,13732,14556,13410,14702,12996,14814,12596,14891,12291,14937,11834,14957,11489,14958,11194,14943,10803,14921,10506,14893,10278,14858,9960,14484,14039,14487,14025,14499,13941,14524,13740,14574,13468,14654,13106,14743,12678,14818,12344,14867,11893,14889,11509,14893,11180,14881,10751,14852,10428,14812,10128,14765,9754,14712,9466,14764,13480,14764,13475,14766,13440,14766,13347,14769,13070,14786,12713,14816,12387,14844,11957,14860,11549,14868,11215,14855,10751,14825,10403,14782,10044,14729,9651,14666,9352,14599,9029,14967,12835,14966,12831,14963,12804,14954,12723,14936,12564,14917,12347,14900,11958,14886,11569,14878,11247,14859,10765,14828,10401,14784,10011,14727,9600,14660,9289,14586,8893,14508,8533,15111,12234,15110,12234,15104,12216,15092,12156,15067,12010,15028,11776,14981,11500,14942,11205,14902,10752,14861,10393,14812,9991,14752,9570,14682,9252,14603,8808,14519,8445,14431,8145,15209,11449,15208,11451,15202,11451,15190,11438,15163,11384,15117,11274,15055,10979,14994,10648,14932,10343,14871,9936,14803,9532,14729,9218,14645,8742,14556,8381,14461,8020,14365,7603,15273,10603,15272,10607,15267,10619,15256,10631,15231,10614,15182,10535,15118,10389,15042,10167,14963,9787,14883,9447,14800,9115,14710,8665,14615,8318,14514,7911,14411,7507,14279,7198,15314,9675,15313,9683,15309,9712,15298,9759,15277,9797,15229,9773,15166,9668,15084,9487,14995,9274,14898,8910,14800,8539,14697,8234,14590,7790,14479,7409,14367,7067,14178,6621,15337,8619,15337,8631,15333,8677,15325,8769,15305,8871,15264,8940,15202,8909,15119,8775,15022,8565,14916,8328,14804,8009,14688,7614,14569,7287,14448,6888,14321,6483,14088,6171,15350,7402,15350,7419,15347,7480,15340,7613,15322,7804,15287,7973,15229,8057,15148,8012,15046,7846,14933,7611,14810,7357,14682,7069,14552,6656,14421,6316,14251,5948,14007,5528,15356,5942,15356,5977,15353,6119,15348,6294,15332,6551,15302,6824,15249,7044,15171,7122,15070,7050,14949,6861,14818,6611,14679,6349,14538,6067,14398,5651,14189,5311,13935,4958,15359,4123,15359,4153,15356,4296,15353,4646,15338,5160,15311,5508,15263,5829,15188,6042,15088,6094,14966,6001,14826,5796,14678,5543,14527,5287,14377,4985,14133,4586,13869,4257,15360,1563,15360,1642,15358,2076,15354,2636,15341,3350,15317,4019,15273,4429,15203,4732,15105,4911,14981,4932,14836,4818,14679,4621,14517,4386,14359,4156,14083,3795,13808,3437,15360,122,15360,137,15358,285,15355,636,15344,1274,15322,2177,15281,2765,15215,3223,15120,3451,14995,3569,14846,3567,14681,3466,14511,3305,14344,3121,14037,2800,13753,2467,15360,0,15360,1,15359,21,15355,89,15346,253,15325,479,15287,796,15225,1148,15133,1492,15008,1749,14856,1882,14685,1886,14506,1783,14324,1608,13996,1398,13702,1183]);let sa=null;function Hw(){return sa===null&&(sa=new uS(Fw,16,16,er,da),sa.name="DFG_LUT",sa.minFilter=zn,sa.magFilter=zn,sa.wrapS=Pa,sa.wrapT=Pa,sa.generateMipmaps=!1,sa.needsUpdate=!0),sa}class bS{constructor(e={}){const{canvas:i=tb(),context:r=null,depth:c=!0,stencil:f=!1,alpha:d=!1,antialias:h=!1,premultipliedAlpha:m=!0,preserveDrawingBuffer:p=!1,powerPreference:x="default",failIfMajorPerformanceCaveat:v=!1,reversedDepthBuffer:_=!1,outputBufferType:M=pi}=e;this.isWebGLRenderer=!0;let R;if(r!==null){if(typeof WebGLRenderingContext<"u"&&r instanceof WebGLRenderingContext)throw new Error("THREE.WebGLRenderer: WebGL 1 is not supported since r163.");R=r.getContextAttributes().alpha}else R=d;const N=M,y=new Set([om,rm,sm]),S=new Set([pi,fa,Ll,Ol,nm,im]),O=new Uint32Array(4),k=new Int32Array(4),C=new dt;let D=null,U=null;const B=[],b=[];let L=null;this.domElement=i,this.debug={checkShaderErrors:!0,diagnostics:{keywords:!1},onShaderError:null},this.autoClear=!0,this.autoClearColor=!0,this.autoClearDepth=!0,this.autoClearStencil=!0,this.sortObjects=!0,this.clippingPlanes=[],this.localClippingEnabled=!1,this.toneMapping=ca,this.toneMappingExposure=1,this.transmissionResolutionScale=1;const X=this;let j=!1,J=null,P=null,q=null,$=null;this._outputColorSpace=ti;let Y=0,Q=0,pt=null,ot=-1,vt=null;const Tt=new on,Zt=new on;let Yt=null;const H=new he(0);let F=0,gt=i.width,G=i.height,st=1,At=null,Dt=null;const xt=new on(0,0,gt,G),Ct=new on(0,0,gt,G);let Fe=!1;const le=new hm;let pe=!1,ve=!1;const te=new nn,ne=new dt,Be=new on,sn={background:null,fog:null,environment:null,overrideMaterial:null,isScene:!0};let Ue=!1;function je(){return pt===null?st:1}let Z=r;function Je(T,V){return i.getContext(T,V)}let Me,I,E,at,lt,et,St,wt,_t,yt,Nt,Jt,Bt,It,kt,ie,ce,K,Ut,bt,Lt,Vt,Rt;try{const T={alpha:!0,depth:c,stencil:f,antialias:h,premultipliedAlpha:m,preserveDrawingBuffer:p,powerPreference:x,failIfMajorPerformanceCaveat:v};if("setAttribute"in i&&i.setAttribute("data-engine",`three.js r${Jp}`),i.addEventListener("webglcontextlost",Le,!1),i.addEventListener("webglcontextrestored",ue,!1),i.addEventListener("webglcontextcreationerror",ni,!1),Z===null){const V="webgl2";if(Z=Je(V,T),Z===null)throw Je(V)?new Error("THREE.WebGLRenderer: Error creating WebGL context with your selected attributes."):new Error("THREE.WebGLRenderer: Error creating WebGL context.")}$t()}catch(T){throw i.removeEventListener("webglcontextlost",Le,!1),i.removeEventListener("webglcontextrestored",ue,!1),i.removeEventListener("webglcontextcreationerror",ni,!1),ze("WebGLRenderer: "+T.message),T}function $t(){Me=new HA(Z),Me.init(),Lt=new Dw(Z,Me),I=new NA(Z,Me,e,Lt),E=new Cw(Z,Me),I.reversedDepthBuffer&&_&&E.buffers.depth.setReversed(!0),P=Z.createFramebuffer(),q=Z.createFramebuffer(),$=Z.createFramebuffer(),at=new kA(Z),lt=new mw,et=new Nw(Z,Me,E,lt,I,Lt,at),St=new FA(X),wt=new Wb(Z),Vt=new wA(Z,wt),_t=new GA(Z,wt,at,Vt),yt=new WA(Z,_t,wt,Vt,at),K=new XA(Z,I,et),kt=new DA(lt),Nt=new pw(X,St,Me,I,Vt,kt),Jt=new Bw(X,lt),Bt=new vw,It=new Ew(Me),ce=new RA(X,St,E,yt,R,m),ie=new ww(X,yt,I),Rt=new zw(Z,at,I,E),Ut=new CA(Z,Me,at),bt=new VA(Z,Me,at),at.programs=Nt.programs,X.capabilities=I,X.extensions=Me,X.properties=lt,X.renderLists=Bt,X.shadowMap=ie,X.state=E,X.info=at}N!==pi&&(L=new YA(N,i.width,i.height,h,c,f));const Gt=new Pw(X,Z);this.xr=Gt,this.getContext=function(){return Z},this.getContextAttributes=function(){return Z.getContextAttributes()},this.forceContextLoss=function(){const T=Me.get("WEBGL_lose_context");T&&T.loseContext()},this.forceContextRestore=function(){const T=Me.get("WEBGL_lose_context");T&&T.restoreContext()},this.getPixelRatio=function(){return st},this.setPixelRatio=function(T){T!==void 0&&(st=T,this.setSize(gt,G,!1))},this.getSize=function(T){return T.set(gt,G)},this.setSize=function(T,V,mt=!0){if(Gt.isPresenting){oe("WebGLRenderer: Can't change size while VR device is presenting.");return}gt=T,G=V,i.width=Math.floor(T*st),i.height=Math.floor(V*st),mt===!0&&(i.style.width=T+"px",i.style.height=V+"px"),L!==null&&L.setSize(i.width,i.height),this.setViewport(0,0,T,V)},this.getDrawingBufferSize=function(T){return T.set(gt*st,G*st).floor()},this.setDrawingBufferSize=function(T,V,mt){gt=T,G=V,st=mt,i.width=Math.floor(T*mt),i.height=Math.floor(V*mt),this.setViewport(0,0,T,V)},this.setEffects=function(T){if(N===pi){ze("WebGLRenderer: setEffects() requires outputBufferType set to HalfFloatType or FloatType.");return}if(T){for(let V=0;V<T.length;V++)if(T[V].isOutputPass===!0){oe("WebGLRenderer: OutputPass is not needed in setEffects(). Tone mapping and color space conversion are applied automatically.");break}}L.setEffects(T||[])},this.getCurrentViewport=function(T){return T.copy(Tt)},this.getViewport=function(T){return T.copy(xt)},this.setViewport=function(T,V,mt,ct){T.isVector4?xt.set(T.x,T.y,T.z,T.w):xt.set(T,V,mt,ct),E.viewport(Tt.copy(xt).multiplyScalar(st).round())},this.getScissor=function(T){return T.copy(Ct)},this.setScissor=function(T,V,mt,ct){T.isVector4?Ct.set(T.x,T.y,T.z,T.w):Ct.set(T,V,mt,ct),E.scissor(Zt.copy(Ct).multiplyScalar(st).round())},this.getScissorTest=function(){return Fe},this.setScissorTest=function(T){E.setScissorTest(Fe=T)},this.setOpaqueSort=function(T){At=T},this.setTransparentSort=function(T){Dt=T},this.getClearColor=function(T){return T.copy(ce.getClearColor())},this.setClearColor=function(){ce.setClearColor(...arguments)},this.getClearAlpha=function(){return ce.getClearAlpha()},this.setClearAlpha=function(){ce.setClearAlpha(...arguments)},this.clear=function(T=!0,V=!0,mt=!0){let ct=0;if(T){let ut=!1;if(pt!==null){const zt=pt.texture.format;ut=y.has(zt)}if(ut){const zt=pt.texture.type,Xt=S.has(zt),Ot=ce.getClearColor(),jt=ce.getClearAlpha(),Kt=Ot.r,se=Ot.g,fe=Ot.b;Xt?(O[0]=Kt,O[1]=se,O[2]=fe,O[3]=jt,Z.clearBufferuiv(Z.COLOR,0,O)):(k[0]=Kt,k[1]=se,k[2]=fe,k[3]=jt,Z.clearBufferiv(Z.COLOR,0,k))}else ct|=Z.COLOR_BUFFER_BIT}V&&(ct|=Z.DEPTH_BUFFER_BIT,this.state.buffers.depth.setMask(!0)),mt&&(ct|=Z.STENCIL_BUFFER_BIT,this.state.buffers.stencil.setMask(4294967295)),ct!==0&&Z.clear(ct)},this.clearColor=function(){this.clear(!0,!1,!1)},this.clearDepth=function(){this.clear(!1,!0,!1)},this.clearStencil=function(){this.clear(!1,!1,!0)},this.setNodesHandler=function(T){T.setRenderer(this),J=T},this.dispose=function(){i.removeEventListener("webglcontextlost",Le,!1),i.removeEventListener("webglcontextrestored",ue,!1),i.removeEventListener("webglcontextcreationerror",ni,!1),ce.dispose(),Bt.dispose(),It.dispose(),lt.dispose(),St.dispose(),yt.dispose(),Vt.dispose(),Rt.dispose(),Nt.dispose(),Gt.dispose(),Gt.removeEventListener("sessionstart",bs),Gt.removeEventListener("sessionend",Ga),Xi.stop()};function Le(T){T.preventDefault(),V_("WebGLRenderer: Context Lost."),j=!0}function ue(){V_("WebGLRenderer: Context Restored."),j=!1;const T=at.autoReset,V=ie.enabled,mt=ie.autoUpdate,ct=ie.needsUpdate,ut=ie.type;$t(),at.autoReset=T,ie.enabled=V,ie.autoUpdate=mt,ie.needsUpdate=ct,ie.type=ut}function ni(T){ze("WebGLRenderer: A WebGL context could not be created. Reason: ",T.statusMessage)}function mi(T){const V=T.target;V.removeEventListener("dispose",mi),Yu(V)}function Yu(T){sr(T),lt.remove(T)}function sr(T){const V=lt.get(T).programs;V!==void 0&&(V.forEach(function(mt){Nt.releaseProgram(mt)}),T.isShaderMaterial&&Nt.releaseShaderCache(T))}this.renderBufferDirect=function(T,V,mt,ct,ut,zt){V===null&&(V=sn);const Xt=ut.isMesh&&ut.matrixWorld.determinantAffine()<0,Ot=To(T,V,mt,ct,ut);E.setMaterial(ct,Xt);let jt=mt.index,Kt=1;if(ct.wireframe===!0){if(jt=_t.getWireframeAttribute(mt),jt===void 0)return;Kt=2}const se=mt.drawRange,fe=mt.attributes.position;let Wt=se.start*Kt,Ee=(se.start+se.count)*Kt;zt!==null&&(Wt=Math.max(Wt,zt.start*Kt),Ee=Math.min(Ee,(zt.start+zt.count)*Kt)),jt!==null?(Wt=Math.max(Wt,0),Ee=Math.min(Ee,jt.count)):fe!=null&&(Wt=Math.max(Wt,0),Ee=Math.min(Ee,fe.count));const xe=Ee-Wt;if(xe<0||xe===1/0)return;Vt.setup(ut,ct,Ot,mt,jt);let Ze,Ve=Ut;if(jt!==null&&(Ze=wt.get(jt),Ve=bt,Ve.setIndex(Ze)),ut.isMesh)ct.wireframe===!0?(E.setLineWidth(ct.wireframeLinewidth*je()),Ve.setMode(Z.LINES)):Ve.setMode(Z.TRIANGLES);else if(ut.isLine){let xn=ct.linewidth;xn===void 0&&(xn=1),E.setLineWidth(xn*je()),ut.isLineSegments?Ve.setMode(Z.LINES):ut.isLineLoop?Ve.setMode(Z.LINE_LOOP):Ve.setMode(Z.LINE_STRIP)}else ut.isPoints?Ve.setMode(Z.POINTS):ut.isSprite&&Ve.setMode(Z.TRIANGLES);if(ut.isBatchedMesh)if(Me.get("WEBGL_multi_draw"))Ve.renderMultiDraw(ut._multiDrawStarts,ut._multiDrawCounts,ut._multiDrawCount);else{const xn=ut._multiDrawStarts,Ft=ut._multiDrawCounts,ln=ut._multiDrawCount,Oe=jt?wt.get(jt).bytesPerElement:1,Hn=lt.get(ct).currentProgram.getUniforms();for(let ii=0;ii<ln;ii++)Hn.setValue(Z,"_gl_DrawID",ii),Ve.render(xn[ii]/Oe,Ft[ii])}else if(ut.isInstancedMesh)Ve.renderInstances(Wt,xe,ut.count);else if(mt.isInstancedBufferGeometry){const xn=mt._maxInstanceCount!==void 0?mt._maxInstanceCount:1/0,Ft=Math.min(mt.instanceCount,xn);Ve.renderInstances(Wt,xe,Ft)}else Ve.render(Wt,xe)};function Es(T,V,mt,ct){J!==null&&T.isNodeMaterial&&J.setObject(ct,T),pe===!0&&kt.setState(T,mt,!1),T.transparent===!0&&T.side===Bi&&T.forceSinglePass===!1?(T.side=qn,T.needsUpdate=!0,Ts(T,V,ct),T.side=$s,T.needsUpdate=!0,Ts(T,V,ct),T.side=Bi):Ts(T,V,ct)}this.compile=function(T,V,mt=null){mt===null&&(mt=T),J!==null&&J.renderStart(T,V,mt),U=It.get(mt),U.init(V),b.push(U),mt.traverseVisible(function(ut){ut.isLight&&ut.layers.test(V.layers)&&(U.pushLight(ut),ut.castShadow&&U.pushShadow(ut))}),T!==mt&&T.traverseVisible(function(ut){ut.isLight&&ut.layers.test(V.layers)&&(U.pushLight(ut),ut.castShadow&&U.pushShadow(ut))}),U.setupLights(),J!==null&&J.updateLights(U.state.lightsArray),ve=this.localClippingEnabled,pe=kt.init(this.clippingPlanes,ve),pe===!0&&kt.setGlobalState(this.clippingPlanes,V),J!==null&&ie.render(U.state.shadowsArray,mt,V);const ct=new Set;return T.traverse(function(ut){if(!(ut.isMesh||ut.isPoints||ut.isLine||ut.isSprite))return;const zt=ut.material;if(zt)if(Array.isArray(zt))for(let Xt=0;Xt<zt.length;Xt++){const Ot=zt[Xt];Es(Ot,mt,V,ut),ct.add(Ot)}else Es(zt,mt,V,ut),ct.add(zt)}),U=b.pop(),J!==null&&J.renderEnd(),ct},this.compileAsync=function(T,V,mt=null){const ct=this.compile(T,V,mt);return new Promise(ut=>{function zt(){if(ct.forEach(function(Xt){const jt=lt.get(Xt).currentProgram;(jt===void 0||jt.isReady())&&ct.delete(Xt)}),ct.size===0){ut(T);return}setTimeout(zt,10)}Me.get("KHR_parallel_shader_compile")!==null?zt():setTimeout(zt,10)})};let Ha=null;function pa(T){Ha&&Ha(T)}function bs(){Xi.stop()}function Ga(){Xi.start()}const Xi=new gS;Xi.setAnimationLoop(pa),typeof self<"u"&&Xi.setContext(self),this.setAnimationLoop=function(T){Ha=T,Gt.setAnimationLoop(T),T===null?Xi.stop():Xi.start()},Gt.addEventListener("sessionstart",bs),Gt.addEventListener("sessionend",Ga),this.render=function(T,V){if(V!==void 0&&V.isCamera!==!0){ze("WebGLRenderer.render: camera is not an instance of THREE.Camera.");return}if(j===!0)return;J!==null&&J.renderStart(T,V);const mt=Gt.enabled===!0&&Gt.isPresenting===!0,ct=L!==null&&(pt===null||mt)&&L.begin(X,pt);if(T.matrixWorldAutoUpdate===!0&&T.updateMatrixWorld(),V.parent===null&&V.matrixWorldAutoUpdate===!0&&V.updateMatrixWorld(),Gt.enabled===!0&&Gt.isPresenting===!0&&(L===null||L.isCompositing()===!1)&&(Gt.cameraAutoUpdate===!0&&Gt.updateCamera(V),V=Gt.getCamera()),T.isScene===!0&&T.onBeforeRender(X,T,V,pt),U=It.get(T,b.length),U.init(V),U.state.textureUnits=et.getTextureUnits(),b.push(U),te.multiplyMatrices(V.projectionMatrix,V.matrixWorldInverse),le.setFromProjectionMatrix(te,la,V.reversedDepth),ve=this.localClippingEnabled,pe=kt.init(this.clippingPlanes,ve),D=Bt.get(T,B.length),D.init(),B.push(D),Gt.enabled===!0&&Gt.isPresenting===!0){const Xt=X.xr.getDepthSensingMesh();Xt!==null&&So(Xt,V,-1/0,X.sortObjects)}So(T,V,0,X.sortObjects),D.finish(),J!==null&&J.updateLights(U.state.lightsArray),X.sortObjects===!0&&D.sort(At,Dt),Ue=Gt.enabled===!1||Gt.isPresenting===!1||Gt.hasDepthSensing()===!1,Ue&&ce.addToRenderList(D,T),this.info.render.frame++,this.info.autoReset===!0&&this.info.reset(),pe===!0&&kt.beginShadows();const ut=U.state.shadowsArray;if(ie.render(ut,T,V),pe===!0&&kt.endShadows(),(ct&&L.hasRenderPass())===!1){const Xt=D.opaque,Ot=D.transmissive;if(U.setupLights(),V.isArrayCamera){const jt=V.cameras;if(Ot.length>0)for(let Kt=0,se=jt.length;Kt<se;Kt++){const fe=jt[Kt];rr(Xt,Ot,T,fe)}Ue&&ce.render(T);for(let Kt=0,se=jt.length;Kt<se;Kt++){const fe=jt[Kt];yo(D,T,fe,fe.viewport)}}else Ot.length>0&&rr(Xt,Ot,T,V),Ue&&ce.render(T),yo(D,T,V)}pt!==null&&Q===0&&(et.updateMultisampleRenderTarget(pt),et.updateRenderTargetMipmap(pt)),ct&&L.end(X),T.isScene===!0&&T.onAfterRender(X,T,V),Vt.resetDefaultState(),ot=-1,vt=null,b.pop(),b.length>0?(U=b[b.length-1],et.setTextureUnits(U.state.textureUnits),pe===!0&&kt.setGlobalState(X.clippingPlanes,U.state.camera)):U=null,B.pop(),B.length>0?D=B[B.length-1]:D=null,J!==null&&J.renderEnd()};function So(T,V,mt,ct){if(T.visible===!1)return;if(T.layers.test(V.layers)){if(T.isGroup)mt=T.renderOrder;else if(T.isLOD)T.autoUpdate===!0&&T.update(V);else if(T.isLightProbeGrid)U.pushLightProbeGrid(T);else if(T.isLight)U.pushLight(T),T.castShadow&&U.pushShadow(T);else if(T.isSprite){if(!T.frustumCulled||T.intersectsFrustum(le)){ct&&Be.setFromMatrixPosition(T.matrixWorld).applyMatrix4(te);const Xt=yt.update(T),Ot=T.material;Ot.visible&&D.push(T,Xt,Ot,mt,Be.z,null,V)}}else if((T.isMesh||T.isLine||T.isPoints)&&(!T.frustumCulled||T.intersectsFrustum(le))){const Xt=yt.update(T),Ot=T.material;if(ct&&(T.boundingSphere!==void 0?(T.boundingSphere===null&&T.computeBoundingSphere(),Be.copy(T.boundingSphere.center)):(Xt.boundingSphere===null&&Xt.computeBoundingSphere(),Be.copy(Xt.boundingSphere.center)),Be.applyMatrix4(T.matrixWorld).applyMatrix4(te)),Array.isArray(Ot)){const jt=Xt.groups;for(let Kt=0,se=jt.length;Kt<se;Kt++){const fe=jt[Kt],Wt=Ot[fe.materialIndex];Wt&&Wt.visible&&D.push(T,Xt,Wt,mt,Be.z,fe,V)}}else Ot.visible&&D.push(T,Xt,Ot,mt,Be.z,null,V)}}const zt=T.children;for(let Xt=0,Ot=zt.length;Xt<Ot;Xt++)So(zt[Xt],V,mt,ct)}function yo(T,V,mt,ct){const{opaque:ut,transmissive:zt,transparent:Xt}=T;U.setupLightsView(mt),pe===!0&&kt.setGlobalState(X.clippingPlanes,mt),ct&&E.viewport(Tt.copy(ct)),ut.length>0&&Wi(ut,V,mt),zt.length>0&&Wi(zt,V,mt),Xt.length>0&&Wi(Xt,V,mt),E.buffers.depth.setTest(!0),E.buffers.depth.setMask(!0),E.buffers.color.setMask(!0),E.setPolygonOffset(!1)}function rr(T,V,mt,ct){if((mt.isScene===!0?mt.overrideMaterial:null)!==null)return;if(U.state.transmissionRenderTarget[ct.id]===void 0){const Wt=Me.has("EXT_color_buffer_half_float")||Me.has("EXT_color_buffer_float");U.state.transmissionRenderTarget[ct.id]=new Vi(1,1,{generateMipmaps:!0,type:Wt?da:pi,minFilter:Qs,samples:Math.max(4,I.samples),stencilBuffer:f,resolveDepthBuffer:!1,resolveStencilBuffer:!1,storeMultisampledDepthBuffer:!1,storeMultisampledStencilBuffer:!1,colorSpace:Pe.workingColorSpace})}const zt=U.state.transmissionRenderTarget[ct.id],Xt=ct.viewport||Tt;zt.setSize(Xt.z*X.transmissionResolutionScale,Xt.w*X.transmissionResolutionScale);const Ot=X.getRenderTarget(),jt=X.getActiveCubeFace(),Kt=X.getActiveMipmapLevel();X.setRenderTarget(zt),X.getClearColor(H),F=X.getClearAlpha(),F<1&&X.setClearColor(16777215,.5),X.clear(),Ue&&ce.render(mt);const se=X.toneMapping;X.toneMapping=ca;const fe=ct.viewport;if(ct.viewport!==void 0&&(ct.viewport=void 0),U.setupLightsView(ct),pe===!0&&kt.setGlobalState(X.clippingPlanes,ct),Wi(T,mt,ct),et.updateMultisampleRenderTarget(zt),et.updateRenderTargetMipmap(zt),Me.has("WEBGL_multisampled_render_to_texture")===!1){let Wt=!1;for(let Ee=0,xe=V.length;Ee<xe;Ee++){const Ze=V[Ee],{object:Ve,geometry:xn,material:Ft,group:ln}=Ze;if(Ft.side===Bi&&Ve.layers.test(ct.layers)){const Oe=Ft.side;Ft.side=qn,Ft.needsUpdate=!0,Fl(Ve,mt,ct,xn,Ft,ln),Ft.side=Oe,Ft.needsUpdate=!0,Wt=!0}}Wt===!0&&(et.updateMultisampleRenderTarget(zt),et.updateRenderTargetMipmap(zt))}X.setRenderTarget(Ot,jt,Kt),X.setClearColor(H,F),fe!==void 0&&(ct.viewport=fe),X.toneMapping=se}function Wi(T,V,mt){const ct=V.isScene===!0?V.overrideMaterial:null;for(let ut=0,zt=T.length;ut<zt;ut++){const Xt=T[ut],{object:Ot,geometry:jt,group:Kt}=Xt;let se=Xt.material;se.allowOverride===!0&&ct!==null&&(se=ct),Ot.layers.test(mt.layers)&&Fl(Ot,V,mt,jt,se,Kt)}}function Fl(T,V,mt,ct,ut,zt){J!==null&&ut.isNodeMaterial&&J.setObject(T,ut),T.onBeforeRender(X,V,mt,ct,ut,zt),T.modelViewMatrix.multiplyMatrices(mt.matrixWorldInverse,T.matrixWorld),T.normalMatrix.getNormalMatrix(T.modelViewMatrix),ut.onBeforeRender(X,V,mt,ct,T,zt),ut.transparent===!0&&ut.side===Bi&&ut.forceSinglePass===!1?(ut.side=qn,ut.needsUpdate=!0,X.renderBufferDirect(mt,V,ct,ut,T,zt),ut.side=$s,ut.needsUpdate=!0,X.renderBufferDirect(mt,V,ct,ut,T,zt),ut.side=Bi):X.renderBufferDirect(mt,V,ct,ut,T,zt),T.onAfterRender(X,V,mt,ct,ut,zt)}function Ts(T,V,mt){V.isScene!==!0&&(V=sn);const ct=lt.get(T),ut=U.state.lights,zt=U.state.shadowsArray,Xt=ut.state.version,Ot=Nt.getParameters(T,ut.state,zt,V,mt,U.state.lightProbeGridArray),jt=Nt.getProgramCacheKey(Ot);let Kt=ct.programs;ct.environment=T.isMeshStandardMaterial||T.isMeshLambertMaterial||T.isMeshPhongMaterial?V.environment:null,ct.fog=V.fog;const se=T.isMeshStandardMaterial||T.isMeshLambertMaterial&&!T.envMap||T.isMeshPhongMaterial&&!T.envMap;ct.envMap=St.get(T.envMap||ct.environment,se),ct.envMapRotation=ct.environment!==null&&T.envMap===null?V.environmentRotation:T.envMapRotation,Kt===void 0&&(T.addEventListener("dispose",mi),Kt=new Map,ct.programs=Kt);let fe=Kt.get(jt);if(fe!==void 0){if(ct.currentProgram===fe&&ct.lightsStateVersion===Xt)return Eo(T,Ot),fe}else Ot.uniforms=Nt.getUniforms(T),J!==null&&T.isNodeMaterial&&J.build(T,mt,Ot),T.onBeforeCompile(Ot,X),fe=Nt.acquireProgram(Ot,jt),Kt.set(jt,fe),ct.uniforms=Ot.uniforms;const Wt=ct.uniforms;return(!T.isShaderMaterial&&!T.isRawShaderMaterial||T.clipping===!0)&&(Wt.clippingPlanes=kt.uniform),Eo(T,Ot),ct.needsLights=Gl(T),ct.lightsStateVersion=Xt,ct.needsLights&&(Wt.ambientLightColor.value=ut.state.ambient,Wt.lightProbe.value=ut.state.probe,Wt.sunLights.value=ut.state.sun,Wt.sunLightShadows.value=ut.state.sunShadow,Wt.directionalLights.value=ut.state.directional,Wt.directionalLightShadows.value=ut.state.directionalShadow,Wt.spotLights.value=ut.state.spot,Wt.spotLightShadows.value=ut.state.spotShadow,Wt.rectAreaLights.value=ut.state.rectArea,Wt.ltc_1.value=ut.state.rectAreaLTC1,Wt.ltc_2.value=ut.state.rectAreaLTC2,Wt.pointLights.value=ut.state.point,Wt.pointLightShadows.value=ut.state.pointShadow,Wt.hemisphereLights.value=ut.state.hemi,Wt.sunShadowMatrix.value=ut.state.sunShadowMatrix,Wt.sunShadowCascade.value=ut.state.sunShadowCascade,Wt.directionalShadowMatrix.value=ut.state.directionalShadowMatrix,Wt.spotLightMatrix.value=ut.state.spotLightMatrix,Wt.spotLightMap.value=ut.state.spotLightMap,Wt.pointShadowMatrix.value=ut.state.pointShadowMatrix),ct.lightProbeGrid=U.state.lightProbeGridArray.length>0,ct.currentProgram=fe,ct.uniformsList=null,fe}function Mo(T){if(T.uniformsList===null){const V=T.currentProgram.getUniforms();T.uniformsList=Pu.seqWithValue(V.seq,T.uniforms)}return T.uniformsList}function Eo(T,V){const mt=lt.get(T);mt.outputColorSpace=V.outputColorSpace,mt.batching=V.batching,mt.batchingColor=V.batchingColor,mt.instancing=V.instancing,mt.instancingColor=V.instancingColor,mt.instancingMorph=V.instancingMorph,mt.skinning=V.skinning,mt.morphTargets=V.morphTargets,mt.morphNormals=V.morphNormals,mt.morphColors=V.morphColors,mt.morphTargetsCount=V.morphTargetsCount,mt.numClippingPlanes=V.numClippingPlanes,mt.numIntersection=V.numClipIntersection,mt.vertexAlphas=V.vertexAlphas,mt.vertexTangents=V.vertexTangents,mt.toneMapping=V.toneMapping}function bo(T,V){if(T.length===0)return null;if(T.length===1)return T[0].texture!==null?T[0]:null;C.setFromMatrixPosition(V.matrixWorld);for(let mt=0,ct=T.length;mt<ct;mt++){const ut=T[mt];if(ut.texture!==null&&ut.boundingBox.containsPoint(C))return ut}return null}function To(T,V,mt,ct,ut){V.isScene!==!0&&(V=sn),et.resetTextureUnits();const zt=V.fog,Xt=ct.isMeshStandardMaterial||ct.isMeshLambertMaterial||ct.isMeshPhongMaterial?V.environment:null,Ot=pt===null?X.outputColorSpace:pt.isXRRenderTarget===!0?pt.texture.colorSpace:Pe.workingColorSpace,jt=ct.isMeshStandardMaterial||ct.isMeshLambertMaterial&&!ct.envMap||ct.isMeshPhongMaterial&&!ct.envMap,Kt=St.get(ct.envMap||Xt,jt),se=ct.vertexColors===!0&&!!mt.attributes.color&&mt.attributes.color.itemSize===4,fe=!!mt.attributes.tangent&&(!!ct.normalMap||ct.anisotropy>0),Wt=!!mt.morphAttributes.position,Ee=!!mt.morphAttributes.normal,xe=!!mt.morphAttributes.color;let Ze=ca;ct.toneMapped&&(pt===null||pt.isXRRenderTarget===!0)&&(Ze=X.toneMapping);const Ve=mt.morphAttributes.position||mt.morphAttributes.normal||mt.morphAttributes.color,xn=Ve!==void 0?Ve.length:0,Ft=lt.get(ct),ln=U.state.lights;if(pe===!0&&(ve===!0||T!==vt)){const we=T===vt&&ct.id===ot;kt.setState(ct,T,we)}let Oe=!1;ct.version===Ft.__version?(Ft.needsLights&&Ft.lightsStateVersion!==ln.state.version||Ft.outputColorSpace!==Ot||ut.isBatchedMesh&&Ft.batching===!1||!ut.isBatchedMesh&&Ft.batching===!0||ut.isBatchedMesh&&Ft.batchingColor===!0&&ut._colorsTexture===null||ut.isBatchedMesh&&Ft.batchingColor===!1&&ut._colorsTexture!==null||ut.isInstancedMesh&&Ft.instancing===!1||!ut.isInstancedMesh&&Ft.instancing===!0||ut.isSkinnedMesh&&Ft.skinning===!1||!ut.isSkinnedMesh&&Ft.skinning===!0||ut.isInstancedMesh&&Ft.instancingColor===!0&&ut.instanceColor===null||ut.isInstancedMesh&&Ft.instancingColor===!1&&ut.instanceColor!==null||ut.isInstancedMesh&&Ft.instancingMorph===!0&&ut.morphTexture===null||ut.isInstancedMesh&&Ft.instancingMorph===!1&&ut.morphTexture!==null||Ft.envMap!==Kt||ct.fog===!0&&Ft.fog!==zt||Ft.numClippingPlanes!==void 0&&(Ft.numClippingPlanes!==kt.numPlanes||Ft.numIntersection!==kt.numIntersection)||Ft.vertexAlphas!==se||Ft.vertexTangents!==fe||Ft.morphTargets!==Wt||Ft.morphNormals!==Ee||Ft.morphColors!==xe||Ft.toneMapping!==Ze||Ft.morphTargetsCount!==xn||!!Ft.lightProbeGrid!=U.state.lightProbeGridArray.length>0)&&(Oe=!0):(Oe=!0,Ft.__version=ct.version);let Hn=Ft.currentProgram;Oe===!0&&(Hn=Ts(ct,V,ut),J&&ct.isNodeMaterial&&J.onUpdateProgram(ct,Hn,Ft));let ii=!1,qi=!1,Se=!1;const He=Hn.getUniforms(),$e=Ft.uniforms;if(E.useProgram(Hn.program)&&(ii=!0,qi=!0,Se=!0),ct.id!==ot&&(ot=ct.id,qi=!0),Ft.needsLights){const we=bo(U.state.lightProbeGridArray,ut);Ft.lightProbeGrid!==we&&(Ft.lightProbeGrid=we,qi=!0)}if(ii||vt!==T){E.buffers.depth.getReversed()&&T.reversedDepth!==!0&&(T._reversedDepth=!0,T.updateProjectionMatrix()),He.setValue(Z,"projectionMatrix",T.projectionMatrix),He.setValue(Z,"viewMatrix",T.matrixWorldInverse);const cn=He.map.cameraPosition;cn!==void 0&&cn.setValue(Z,ne.setFromMatrixPosition(T.matrixWorld)),I.logarithmicDepthBuffer&&He.setValue(Z,"logDepthBufFC",2/(Math.log(T.far+1)/Math.LN2)),(ct.isMeshPhongMaterial||ct.isMeshToonMaterial||ct.isMeshLambertMaterial||ct.isMeshBasicMaterial||ct.isMeshStandardMaterial||ct.isShaderMaterial)&&He.setValue(Z,"isOrthographic",T.isOrthographicCamera===!0),vt!==T&&(vt=T,qi=!0,Se=!0)}if(Ft.needsLights&&(ln.state.sunShadowMap.length>0&&He.setValue(Z,"sunShadowMap",ln.state.sunShadowMap,et),ln.state.directionalShadowMap.length>0&&He.setValue(Z,"directionalShadowMap",ln.state.directionalShadowMap,et),ln.state.spotShadowMap.length>0&&He.setValue(Z,"spotShadowMap",ln.state.spotShadowMap,et),ln.state.pointShadowMap.length>0&&He.setValue(Z,"pointShadowMap",ln.state.pointShadowMap,et)),ut.isSkinnedMesh){He.setOptional(Z,ut,"bindMatrix"),He.setOptional(Z,ut,"bindMatrixInverse");const we=ut.skeleton;we&&(we.boneTexture===null&&we.computeBoneTexture(),He.setValue(Z,"boneTexture",we.boneTexture,et))}ut.isBatchedMesh&&(He.setOptional(Z,ut,"batchingTexture"),He.setValue(Z,"batchingTexture",ut._matricesTexture,et),He.setOptional(Z,ut,"batchingIdTexture"),He.setValue(Z,"batchingIdTexture",ut._indirectTexture,et),He.setOptional(Z,ut,"batchingColorTexture"),ut._colorsTexture!==null&&He.setValue(Z,"batchingColorTexture",ut._colorsTexture,et));const ai=mt.morphAttributes;if((ai.position!==void 0||ai.normal!==void 0||ai.color!==void 0)&&K.update(ut,mt,Hn),(qi||Ft.receiveShadow!==ut.receiveShadow)&&(Ft.receiveShadow=ut.receiveShadow,He.setValue(Z,"receiveShadow",ut.receiveShadow)),(ct.isMeshStandardMaterial||ct.isMeshLambertMaterial||ct.isMeshPhongMaterial)&&ct.envMap===null&&V.environment!==null&&($e.envMapIntensity.value=V.environmentIntensity),$e.dfgLUT!==void 0&&($e.dfgLUT.value=Hw()),qi){if(He.setValue(Z,"toneMappingExposure",X.toneMappingExposure),Ft.needsLights&&Hl($e,Se),zt&&ct.fog===!0&&Jt.refreshFogUniforms($e,zt),Jt.refreshMaterialUniforms($e,ct,st,G,U.state.transmissionRenderTarget[T.id]),Ft.needsLights&&Ft.lightProbeGrid){const we=Ft.lightProbeGrid;$e.probesSH.value=we.texture,$e.probesMin.value.copy(we.boundingBox.min),$e.probesMax.value.copy(we.boundingBox.max),$e.probesResolution.value.copy(we.resolution)}Pu.upload(Z,Mo(Ft),$e,et)}if(ct.isShaderMaterial&&ct.uniformsNeedUpdate===!0&&(Pu.upload(Z,Mo(Ft),$e,et),ct.uniformsNeedUpdate=!1),ct.isSpriteMaterial&&He.setValue(Z,"center",ut.center),He.setValue(Z,"modelViewMatrix",ut.modelViewMatrix),He.setValue(Z,"normalMatrix",ut.normalMatrix),He.setValue(Z,"modelMatrix",ut.matrixWorld),ct.uniformsGroups!==void 0){const we=ct.uniformsGroups;for(let cn=0,ma=we.length;cn<ma;cn++){const Vl=we[cn];Rt.update(Vl,Hn),Rt.bind(Vl,Hn)}}return Hn}function Hl(T,V){T.ambientLightColor.needsUpdate=V,T.lightProbe.needsUpdate=V,T.sunLights.needsUpdate=V,T.sunLightShadows.needsUpdate=V,T.directionalLights.needsUpdate=V,T.directionalLightShadows.needsUpdate=V,T.pointLights.needsUpdate=V,T.pointLightShadows.needsUpdate=V,T.spotLights.needsUpdate=V,T.spotLightShadows.needsUpdate=V,T.rectAreaLights.needsUpdate=V,T.hemisphereLights.needsUpdate=V}function Gl(T){return T.isMeshLambertMaterial||T.isMeshToonMaterial||T.isMeshPhongMaterial||T.isMeshStandardMaterial||T.isShadowMaterial||T.isShaderMaterial&&T.lights===!0}this.getActiveCubeFace=function(){return Y},this.getActiveMipmapLevel=function(){return Q},this.getRenderTarget=function(){return pt},this.setRenderTargetTextures=function(T,V,mt){const ct=lt.get(T);ct.__autoAllocateDepthBuffer=T.resolveDepthBuffer===!1,ct.__autoAllocateDepthBuffer===!1&&(ct.__useRenderToTexture=!1),lt.get(T.texture).__webglTexture=V,lt.get(T.depthTexture).__webglTexture=ct.__autoAllocateDepthBuffer?void 0:mt,ct.__hasExternalTextures=!0},this.setRenderTargetFramebuffer=function(T,V){const mt=lt.get(T);mt.__webglFramebuffer=V,mt.__useDefaultFramebuffer=V===void 0},this.setRenderTarget=function(T,V=0,mt=0){pt=T,Y=V,Q=mt;let ct=null,ut=!1,zt=!1;if(T){const Ot=lt.get(T);if(Ot.__useDefaultFramebuffer!==void 0){E.bindFramebuffer(Z.FRAMEBUFFER,Ot.__webglFramebuffer),Tt.copy(T.viewport),Zt.copy(T.scissor),Yt=T.scissorTest,E.viewport(Tt),E.scissor(Zt),E.setScissorTest(Yt),ot=-1;return}else if(Ot.__webglFramebuffer===void 0)et.setupRenderTarget(T);else if(Ot.__hasExternalTextures)et.rebindTextures(T,lt.get(T.texture).__webglTexture,lt.get(T.depthTexture).__webglTexture);else if(T.depthBuffer){const se=T.depthTexture;if(Ot.__boundDepthTexture!==se){if(se!==null&&lt.has(se)&&(T.width!==se.image.width||T.height!==se.image.height))throw new Error("THREE.WebGLRenderer: Attached DepthTexture is initialized to the incorrect size.");et.setupDepthRenderbuffer(T)}}const jt=T.texture;(jt.isData3DTexture||jt.isDataArrayTexture||jt.isCompressedArrayTexture)&&(zt=!0);const Kt=lt.get(T).__webglFramebuffer;T.isWebGLCubeRenderTarget?(Array.isArray(Kt[V])?ct=Kt[V][mt]:ct=Kt[V],ut=!0):T.samples>0&&et.useMultisampledRTT(T)===!1?ct=lt.get(T).__webglMultisampledFramebuffer:Array.isArray(Kt)?ct=Kt[mt]:ct=Kt,Tt.copy(T.viewport),Zt.copy(T.scissor),Yt=T.scissorTest}else Tt.copy(xt).multiplyScalar(st).floor(),Zt.copy(Ct).multiplyScalar(st).floor(),Yt=Fe;if(mt!==0&&(ct=P),E.bindFramebuffer(Z.FRAMEBUFFER,ct)&&E.drawBuffers(T,ct),E.viewport(Tt),E.scissor(Zt),E.setScissorTest(Yt),ut){const Ot=lt.get(T.texture);Z.framebufferTexture2D(Z.FRAMEBUFFER,Z.COLOR_ATTACHMENT0,Z.TEXTURE_CUBE_MAP_POSITIVE_X+V,Ot.__webglTexture,mt)}else if(zt){const Ot=V;for(let jt=0;jt<T.textures.length;jt++){const Kt=lt.get(T.textures[jt]);Z.framebufferTextureLayer(Z.FRAMEBUFFER,Z.COLOR_ATTACHMENT0+jt,Kt.__webglTexture,mt,Ot)}}else if(T!==null&&mt!==0){const Ot=lt.get(T.texture);Z.framebufferTexture2D(Z.FRAMEBUFFER,Z.COLOR_ATTACHMENT0,Z.TEXTURE_2D,Ot.__webglTexture,mt)}ot=-1};function gi(T){const V=lt.get(T);return(V.__readFormat!==T.format||V.__readType!==T.type)&&(V.__readFormat=T.format,V.__readType=T.type,V.__formatReadable=I.textureFormatReadable(T.format),V.__typeReadable=I.textureTypeReadable(T.type)),V}this.readRenderTargetPixels=function(T,V,mt,ct,ut,zt,Xt,Ot=0){if(!(T&&T.isWebGLRenderTarget)){ze("WebGLRenderer.readRenderTargetPixels: renderTarget is not THREE.WebGLRenderTarget.");return}let jt=lt.get(T).__webglFramebuffer;if(T.isWebGLCubeRenderTarget&&Xt!==void 0&&(jt=jt[Xt]),jt){E.bindFramebuffer(Z.FRAMEBUFFER,jt);try{const Kt=T.textures[Ot],se=Kt.format,fe=Kt.type;T.textures.length>1&&Z.readBuffer(Z.COLOR_ATTACHMENT0+Ot);const Wt=gi(Kt);if(Wt.__formatReadable===!1){ze("WebGLRenderer.readRenderTargetPixels: renderTarget is not in RGBA or implementation defined format.");return}if(Wt.__typeReadable===!1){ze("WebGLRenderer.readRenderTargetPixels: renderTarget is not in UnsignedByteType or implementation defined type.");return}V>=0&&V<=T.width-ct&&mt>=0&&mt<=T.height-ut&&Z.readPixels(V,mt,ct,ut,Lt.convert(se),Lt.convert(fe),zt)}finally{const Kt=pt!==null?lt.get(pt).__webglFramebuffer:null;E.bindFramebuffer(Z.FRAMEBUFFER,Kt)}}},this.readRenderTargetPixelsAsync=async function(T,V,mt,ct,ut,zt,Xt,Ot=0){if(!(T&&T.isWebGLRenderTarget))throw new Error("THREE.WebGLRenderer.readRenderTargetPixels: renderTarget is not THREE.WebGLRenderTarget.");let jt=lt.get(T).__webglFramebuffer;if(T.isWebGLCubeRenderTarget&&Xt!==void 0&&(jt=jt[Xt]),jt)if(V>=0&&V<=T.width-ct&&mt>=0&&mt<=T.height-ut){E.bindFramebuffer(Z.FRAMEBUFFER,jt);const Kt=T.textures[Ot],se=Kt.format,fe=Kt.type;T.textures.length>1&&Z.readBuffer(Z.COLOR_ATTACHMENT0+Ot);const Wt=gi(Kt);if(Wt.__formatReadable===!1)throw new Error("THREE.WebGLRenderer.readRenderTargetPixelsAsync: renderTarget is not in RGBA or implementation defined format.");if(Wt.__typeReadable===!1)throw new Error("THREE.WebGLRenderer.readRenderTargetPixelsAsync: renderTarget is not in UnsignedByteType or implementation defined type.");const Ee=Z.createBuffer();Z.bindBuffer(Z.PIXEL_PACK_BUFFER,Ee),Z.bufferData(Z.PIXEL_PACK_BUFFER,zt.byteLength,Z.STREAM_READ),Z.readPixels(V,mt,ct,ut,Lt.convert(se),Lt.convert(fe),0),Z.bindBuffer(Z.PIXEL_PACK_BUFFER,null);const xe=pt!==null?lt.get(pt).__webglFramebuffer:null;E.bindFramebuffer(Z.FRAMEBUFFER,xe);const Ze=Z.fenceSync(Z.SYNC_GPU_COMMANDS_COMPLETE,0);return Z.flush(),await eb(Z,Ze,4),Z.bindBuffer(Z.PIXEL_PACK_BUFFER,Ee),Z.getBufferSubData(Z.PIXEL_PACK_BUFFER,0,zt),Z.bindBuffer(Z.PIXEL_PACK_BUFFER,null),Z.deleteBuffer(Ee),Z.deleteSync(Ze),zt}else throw new Error("THREE.WebGLRenderer.readRenderTargetPixelsAsync: requested read bounds are out of range.")},this.copyFramebufferToTexture=function(T,V=null,mt=0){const ct=Math.pow(2,-mt),ut=Math.floor(T.image.width*ct),zt=Math.floor(T.image.height*ct),Xt=V!==null?V.x:0,Ot=V!==null?V.y:0;et.setTexture2D(T,0),Z.copyTexSubImage2D(Z.TEXTURE_2D,mt,0,0,Xt,Ot,ut,zt),E.unbindTexture()},this.copyTextureToTexture=function(T,V,mt=null,ct=null,ut=0,zt=0){let Xt,Ot,jt,Kt,se,fe,Wt,Ee,xe;const Ze=T.isCompressedTexture?T.mipmaps[zt]:T.image;if(mt!==null)Xt=mt.max.x-mt.min.x,Ot=mt.max.y-mt.min.y,jt=mt.isBox3?mt.max.z-mt.min.z:1,Kt=mt.min.x,se=mt.min.y,fe=mt.isBox3?mt.min.z:0;else{const $e=Math.pow(2,-ut);Xt=Math.floor(Ze.width*$e),Ot=Math.floor(Ze.height*$e),T.isDataArrayTexture?jt=Ze.depth:T.isData3DTexture?jt=Math.floor(Ze.depth*$e):jt=1,Kt=0,se=0,fe=0}ct!==null?(Wt=ct.x,Ee=ct.y,xe=ct.z):(Wt=0,Ee=0,xe=0);const Ve=Lt.convert(V.format),xn=Lt.convert(V.type);let Ft;V.isData3DTexture?(et.setTexture3D(V,0),Ft=Z.TEXTURE_3D):V.isDataArrayTexture||V.isCompressedArrayTexture?(et.setTexture2DArray(V,0),Ft=Z.TEXTURE_2D_ARRAY):(et.setTexture2D(V,0),Ft=Z.TEXTURE_2D),E.activeTexture(Z.TEXTURE0),E.pixelStorei(Z.UNPACK_FLIP_Y_WEBGL,V.flipY),E.pixelStorei(Z.UNPACK_PREMULTIPLY_ALPHA_WEBGL,V.premultiplyAlpha),E.pixelStorei(Z.UNPACK_ALIGNMENT,V.unpackAlignment);const ln=E.getParameter(Z.UNPACK_ROW_LENGTH),Oe=E.getParameter(Z.UNPACK_IMAGE_HEIGHT),Hn=E.getParameter(Z.UNPACK_SKIP_PIXELS),ii=E.getParameter(Z.UNPACK_SKIP_ROWS),qi=E.getParameter(Z.UNPACK_SKIP_IMAGES);E.pixelStorei(Z.UNPACK_ROW_LENGTH,Ze.width),E.pixelStorei(Z.UNPACK_IMAGE_HEIGHT,Ze.height),E.pixelStorei(Z.UNPACK_SKIP_PIXELS,Kt),E.pixelStorei(Z.UNPACK_SKIP_ROWS,se),E.pixelStorei(Z.UNPACK_SKIP_IMAGES,fe);const Se=T.isDataArrayTexture||T.isData3DTexture,He=V.isDataArrayTexture||V.isData3DTexture;if(T.isDepthTexture){const $e=lt.get(T),ai=lt.get(V),we=lt.get($e.__renderTarget),cn=lt.get(ai.__renderTarget);E.bindFramebuffer(Z.READ_FRAMEBUFFER,we.__webglFramebuffer),E.bindFramebuffer(Z.DRAW_FRAMEBUFFER,cn.__webglFramebuffer);for(let ma=0;ma<jt;ma++)Se&&(Z.framebufferTextureLayer(Z.READ_FRAMEBUFFER,Z.COLOR_ATTACHMENT0,lt.get(T).__webglTexture,ut,fe+ma),Z.framebufferTextureLayer(Z.DRAW_FRAMEBUFFER,Z.COLOR_ATTACHMENT0,lt.get(V).__webglTexture,zt,xe+ma)),Z.blitFramebuffer(Kt,se,Xt,Ot,Wt,Ee,Xt,Ot,Z.DEPTH_BUFFER_BIT,Z.NEAREST);E.bindFramebuffer(Z.READ_FRAMEBUFFER,null),E.bindFramebuffer(Z.DRAW_FRAMEBUFFER,null)}else if(ut!==0||T.isRenderTargetTexture||lt.has(T)){const $e=lt.get(T),ai=lt.get(V);E.bindFramebuffer(Z.READ_FRAMEBUFFER,q),E.bindFramebuffer(Z.DRAW_FRAMEBUFFER,$);for(let we=0;we<jt;we++)Se?Z.framebufferTextureLayer(Z.READ_FRAMEBUFFER,Z.COLOR_ATTACHMENT0,$e.__webglTexture,ut,fe+we):Z.framebufferTexture2D(Z.READ_FRAMEBUFFER,Z.COLOR_ATTACHMENT0,Z.TEXTURE_2D,$e.__webglTexture,ut),He?Z.framebufferTextureLayer(Z.DRAW_FRAMEBUFFER,Z.COLOR_ATTACHMENT0,ai.__webglTexture,zt,xe+we):Z.framebufferTexture2D(Z.DRAW_FRAMEBUFFER,Z.COLOR_ATTACHMENT0,Z.TEXTURE_2D,ai.__webglTexture,zt),ut!==0?Z.blitFramebuffer(Kt,se,Xt,Ot,Wt,Ee,Xt,Ot,Z.COLOR_BUFFER_BIT,Z.NEAREST):He?Z.copyTexSubImage3D(Ft,zt,Wt,Ee,xe+we,Kt,se,Xt,Ot):Z.copyTexSubImage2D(Ft,zt,Wt,Ee,Kt,se,Xt,Ot);E.bindFramebuffer(Z.READ_FRAMEBUFFER,null),E.bindFramebuffer(Z.DRAW_FRAMEBUFFER,null)}else He?T.isDataTexture||T.isData3DTexture?Z.texSubImage3D(Ft,zt,Wt,Ee,xe,Xt,Ot,jt,Ve,xn,Ze.data):V.isCompressedArrayTexture?Z.compressedTexSubImage3D(Ft,zt,Wt,Ee,xe,Xt,Ot,jt,Ve,Ze.data):Z.texSubImage3D(Ft,zt,Wt,Ee,xe,Xt,Ot,jt,Ve,xn,Ze):T.isDataTexture?Z.texSubImage2D(Z.TEXTURE_2D,zt,Wt,Ee,Xt,Ot,Ve,xn,Ze.data):T.isCompressedTexture?Z.compressedTexSubImage2D(Z.TEXTURE_2D,zt,Wt,Ee,Ze.width,Ze.height,Ve,Ze.data):Z.texSubImage2D(Z.TEXTURE_2D,zt,Wt,Ee,Xt,Ot,Ve,xn,Ze);E.pixelStorei(Z.UNPACK_ROW_LENGTH,ln),E.pixelStorei(Z.UNPACK_IMAGE_HEIGHT,Oe),E.pixelStorei(Z.UNPACK_SKIP_PIXELS,Hn),E.pixelStorei(Z.UNPACK_SKIP_ROWS,ii),E.pixelStorei(Z.UNPACK_SKIP_IMAGES,qi),zt===0&&V.generateMipmaps&&Z.generateMipmap(Ft),E.unbindTexture()},this.initRenderTarget=function(T){lt.get(T).__webglFramebuffer===void 0&&et.setupRenderTarget(T)},this.initTexture=function(T){T.isCubeTexture?et.setTextureCube(T,0):T.isData3DTexture?et.setTexture3D(T,0):T.isDataArrayTexture||T.isCompressedArrayTexture?et.setTexture2DArray(T,0):et.setTexture2D(T,0),E.unbindTexture()},this.resetState=function(){Y=0,Q=0,pt=null,E.reset(),Vt.reset()},typeof __THREE_DEVTOOLS__<"u"&&__THREE_DEVTOOLS__.dispatchEvent(new CustomEvent("observe",{detail:this}))}get coordinateSystem(){return la}get outputColorSpace(){return this._outputColorSpace}set outputColorSpace(e){this._outputColorSpace=e;const i=this.getContext();i.drawingBufferColorSpace=Pe._getDrawingBufferColorSpace(e),i.unpackColorSpace=Pe._getUnpackColorSpace()}}const Rl=new dt;function Ri(o,e,i,r,c,f){const d=2*Math.PI*c/4,h=Math.max(f-2*c,0),m=Math.PI/4;Rl.copy(e),Rl[r]=0,Rl.normalize();const p=.5*d/(d+h),x=1-Rl.angleTo(o)/m;return Math.sign(Rl[i])===1?x*p:h/(d+h)+p+p*(1-x)}class uo extends Ms{constructor(e=1,i=1,r=1,c=2,f=.1){const d=c*2+1;if(f=Math.min(e/2,i/2,r/2,f),super(1,1,1,d,d,d),this.type="RoundedBoxGeometry",this.parameters={width:e,height:i,depth:r,segments:c,radius:f},d===1)return;const h=this.toNonIndexed();this.index=null,this.attributes.position=h.attributes.position,this.attributes.normal=h.attributes.normal,this.attributes.uv=h.attributes.uv;const m=new dt,p=new dt,x=new dt(e,i,r).divideScalar(2).subScalar(f),v=this.attributes.position.array,_=this.attributes.normal.array,M=this.attributes.uv.array,R=v.length/6,N=new dt,y=.5/d;for(let S=0,O=0;S<v.length;S+=3,O+=2)switch(m.fromArray(v,S),p.copy(m),p.x-=Math.sign(p.x)*y,p.y-=Math.sign(p.y)*y,p.z-=Math.sign(p.z)*y,p.normalize(),v[S+0]=x.x*Math.sign(m.x)+p.x*f,v[S+1]=x.y*Math.sign(m.y)+p.y*f,v[S+2]=x.z*Math.sign(m.z)+p.z*f,_[S+0]=p.x,_[S+1]=p.y,_[S+2]=p.z,Math.floor(S/R)){case 0:N.set(1,0,0),M[O+0]=Ri(N,p,"z","y",f,r),M[O+1]=1-Ri(N,p,"y","z",f,i);break;case 1:N.set(-1,0,0),M[O+0]=1-Ri(N,p,"z","y",f,r),M[O+1]=1-Ri(N,p,"y","z",f,i);break;case 2:N.set(0,1,0),M[O+0]=1-Ri(N,p,"x","z",f,e),M[O+1]=Ri(N,p,"z","x",f,r);break;case 3:N.set(0,-1,0),M[O+0]=1-Ri(N,p,"x","z",f,e),M[O+1]=1-Ri(N,p,"z","x",f,r);break;case 4:N.set(0,0,1),M[O+0]=1-Ri(N,p,"x","y",f,e),M[O+1]=1-Ri(N,p,"y","x",f,i);break;case 5:N.set(0,0,-1),M[O+0]=Ri(N,p,"x","y",f,e),M[O+1]=1-Ri(N,p,"y","x",f,i);break}}static fromJSON(e){return new uo(e.width,e.height,e.depth,e.segments,e.radius)}}class Gw extends fm{constructor(){super(),this.name="RoomEnvironment",this.position.y=-3.5;const e=new Ms;e.deleteAttribute("uv");const i=new Fi({side:qn}),r=new Fi,c=new mm(16777215,900,28,2);c.position.set(.418,16.199,.3),this.add(c);const f=new _n(e,i);f.position.set(-.757,13.219,.717),f.scale.set(31.713,28.305,28.591),this.add(f);const d=new Ab(e,r,6),h=new Rn;h.position.set(-10.906,2.009,1.846),h.rotation.set(0,-.195,0),h.scale.set(2.328,7.905,4.651),h.updateMatrix(),d.setMatrixAt(0,h.matrix),h.position.set(-5.607,-.754,-.758),h.rotation.set(0,.994,0),h.scale.set(1.97,1.534,3.955),h.updateMatrix(),d.setMatrixAt(1,h.matrix),h.position.set(6.167,.857,7.803),h.rotation.set(0,.561,0),h.scale.set(3.927,6.285,3.687),h.updateMatrix(),d.setMatrixAt(2,h.matrix),h.position.set(-2.017,.018,6.124),h.rotation.set(0,.333,0),h.scale.set(2.002,4.566,2.064),h.updateMatrix(),d.setMatrixAt(3,h.matrix),h.position.set(2.291,-.756,-2.621),h.rotation.set(0,-.286,0),h.scale.set(1.546,1.552,1.496),h.updateMatrix(),d.setMatrixAt(4,h.matrix),h.position.set(-2.193,-.369,-5.547),h.rotation.set(0,.516,0),h.scale.set(3.875,3.487,2.986),h.updateMatrix(),d.setMatrixAt(5,h.matrix),this.add(d);const m=new _n(e,ro(50));m.position.set(-16.116,14.37,8.208),m.scale.set(.1,2.428,2.739),this.add(m);const p=new _n(e,ro(50));p.position.set(-16.109,18.021,-8.207),p.scale.set(.1,2.425,2.751),this.add(p);const x=new _n(e,ro(17));x.position.set(14.904,12.198,-1.832),x.scale.set(.15,4.265,6.331),this.add(x);const v=new _n(e,ro(43));v.position.set(-.462,8.89,14.52),v.scale.set(4.38,5.441,.088),this.add(v);const _=new _n(e,ro(20));_.position.set(3.235,11.486,-12.541),_.scale.set(2.5,2,.1),this.add(_);const M=new _n(e,ro(100));M.position.set(0,20,0),M.scale.set(1,.1,1),this.add(M)}dispose(){const e=new Set;this.traverse(i=>{i.isMesh&&(e.add(i.geometry),e.add(i.material))});for(const i of e)i.dispose()}}function ro(o){return new Pb({color:0,emissive:16777215,emissiveIntensity:o})}function Vw({theme:o}){const e=Ae.useRef(null),i=Ae.useRef(null),r=Ae.useRef(o),c=Ae.useRef(()=>{});return r.current=o,Ae.useEffect(()=>{const f=e.current,d=i.current;if(!f||!d)return;let h;try{h=new bS({canvas:d,antialias:!0,alpha:!0,powerPreference:"high-performance",premultipliedAlpha:!1})}catch{return}h.setClearColor(0,0),h.outputColorSpace=ti,h.toneMapping=tm,h.shadowMap.enabled=!0,h.shadowMap.type=Vx;const m=new fm,p=new ei(30,1,.1,40),x=new lo;m.add(x);const v=[],_=[],M=[],R=et=>(v.push(et),et),N=et=>(_.push(et),et),y=document.createElement("canvas");y.width=512,y.height=680;const S=y.getContext("2d");if(S){const et=S.createLinearGradient(0,0,0,680);et.addColorStop(0,"#3f3c38"),et.addColorStop(.45,"#1a1917"),et.addColorStop(1,"#0e0d0c"),S.fillStyle=et,S.fillRect(0,0,512,680);const St=S.createLinearGradient(0,0,280,160);St.addColorStop(0,"rgba(255,244,224,0.28)"),St.addColorStop(.35,"rgba(255,244,224,0)"),S.fillStyle=St,S.fillRect(0,0,512,680)}const O=new rx(y);O.colorSpace=ti,M.push(O);const k=document.createElement("canvas");k.width=256,k.height=256;const C=k.getContext("2d");if(C){const et=C.createRadialGradient(128,128,16,128,128,122);et.addColorStop(0,"rgba(70,48,28,0.55)"),et.addColorStop(.55,"rgba(70,48,28,0.16)"),et.addColorStop(1,"rgba(70,48,28,0)"),C.fillStyle=et,C.fillRect(0,0,256,256)}const D=new rx(k);M.push(D);const U=N(new Fi({color:"#d9cbb6",roughness:.86,metalness:.02})),B=N(new Fi({color:"#fffaf3",roughness:.94,metalness:0})),b=N(new lx({color:"#d7cec2",roughness:.42,metalness:.02,clearcoat:.28,clearcoatRoughness:.4})),L=N(new lx({color:"#2a2826",roughness:.46,metalness:.18,clearcoat:.22,clearcoatRoughness:.4})),X=N(new Fi({map:O,roughness:.46,metalness:.08})),j=N(new Fi({color:"#a68455",roughness:.45,metalness:.35})),J=N(new Fi({color:"#f0d3a8",roughness:.42,metalness:.05,emissive:new he("#ffd7a4"),emissiveIntensity:.55,side:Bi})),P=N(new dm({map:D,transparent:!0,depthWrite:!1,opacity:.85})),q=N(new Cb({color:"#6a5038",opacity:.28})),$=(et,St,wt=x)=>{const _t=new _n(et,St);return _t.castShadow=!0,_t.receiveShadow=!0,wt.add(_t),_t},Y=$(R(new go(1.45,.85)),P);Y.rotation.x=-Math.PI/2,Y.position.set(.1,.004,0),Y.castShadow=!1,Y.receiveShadow=!1;const Q=$(R(new go(2.4,1.5)),q);Q.rotation.x=-Math.PI/2,Q.position.y=.001,Q.castShadow=!1,Q.receiveShadow=!0,$(R(new uo(1.48,.08,.92,4,.04)),U).position.set(.1,.04,0);const ot=new lo;ot.position.set(.12,.52,-.04),ot.rotation.set(.04,-.32,0),x.add(ot),$(R(new uo(.68,.9,.04,3,.018)),L,ot);const vt=$(R(new uo(.58,.78,.008,2,.006)),X,ot);vt.position.z=.018,vt.castShadow=!1;const Tt=$(R(new uo(.42,.3,.012,2,.006)),B);Tt.position.set(-.12,.26,.16),Tt.rotation.set(.1,.42,-.03),$(R(new pm(.07,48,32)),b).position.set(.34,.11,.16),$(R(new mo(.01,.012,.28,16)),j).position.set(.46,.22,-.16),$(R(new mo(.07,.11,.09,28,1,!0)),J).position.set(.46,.4,-.16);const F=new zb("#fff8ee","#d9cbb6",.7),gt=new $h("#fff4e6",2.7);gt.position.set(-2.6,4.1,2.6),gt.castShadow=!0,gt.shadow.mapSize.set(1024,1024),gt.shadow.radius=5,gt.shadow.bias=-4e-4,gt.shadow.normalBias=.025;const G=gt.shadow.camera;G.near=.4,G.far=12,G.left=-2.4,G.right=2.4,G.top=2.4,G.bottom=-2.4;const st=new $h("#d5e3f2",.55);st.position.set(2.8,1.6,1.4);const At=new $h("#fffaf4",.7);At.position.set(-.6,2.2,-2.8);const Dt=new mm("#ffd2a2",2.4,1.8,2);Dt.position.set(.46,.36,-.16),m.add(F,gt,st,At,Dt);let xt=null;try{const et=new Yp(h),St=new Gw;xt=et.fromScene(St,.04).texture,m.environment=xt,St.traverse(wt=>{var Nt;const _t=wt;(Nt=_t.geometry)==null||Nt.dispose();const yt=_t.material;Array.isArray(yt)?yt.forEach(Jt=>Jt.dispose()):yt==null||yt.dispose()}),et.dispose()}catch{xt=null}let Ct="";const Fe=et=>{if(Ct===et)return;Ct=et;const St=et==="dark";U.color.set(St?"#1c1b19":"#d9cbb6"),B.color.set(St?"#3a3631":"#fffdf8"),b.color.set(St?"#c4bbae":"#f4efe6"),L.color.set(St?"#141312":"#2c2a27"),j.color.set(St?"#8d734c":"#a68455"),J.color.set(St?"#e7c79a":"#f0d3a8"),J.emissiveIntensity=St?.9:.5,P.opacity=St?.7:.85,q.opacity=St?.48:.26,q.color.set(St?"#000000":"#6a5038"),gt.intensity=St?1.15:1.7,st.intensity=St?.22:.48,F.intensity=St?.22:.62,F.color.set(St?"#3a3834":"#fff8ee"),F.groundColor.set(St?"#141311":"#d9cbb6"),Dt.intensity=St?4.5:2.8,m.environmentIntensity=St?.4:.72,h.toneMappingExposure=St?.86:1.02},le=window.matchMedia("(prefers-reduced-motion: reduce)").matches;let pe=0,ve=0,te=!1,ne=!0,Be=0,sn=0,Ue=0;const je=()=>{const et=d.clientWidth,St=d.clientHeight;if(et<2||St<2)return!1;const wt=Math.min(window.devicePixelRatio||1,et<700?1.5:1.75);if(et===Be&&St===sn&&wt===Ue)return!0;Be=et,sn=St,Ue=wt,h.setPixelRatio(wt),h.setSize(et,St,!1),p.aspect=et/St,p.fov=p.aspect<1?30:24;const _t=p.aspect<.9?4.15:p.aspect<1.2?3.55:3.15;return p.position.set(.42,.72,_t),p.lookAt(.14,.34,0),p.updateProjectionMatrix(),!0},Z=()=>{if(Fe(r.current),!!je()){if(!le&&ne){const et=performance.now()*.001,St=pe+Math.sin(et*.32)*.045,wt=ve+Math.sin(et*.24)*.012;x.rotation.y+=(St-x.rotation.y)*.055,x.rotation.x+=(wt-x.rotation.x)*.055,x.position.y=Math.sin(et*.42)*.012}h.render(m,p),f.classList.add("is-live")}};c.current=Z;const Je=()=>{te&&(Z(),window.requestAnimationFrame(Je))},Me=()=>{te||le||(te=!0,window.requestAnimationFrame(Je))},I=()=>{te=!1},E=et=>{const St=f.getBoundingClientRect();if(St.width<2)return;const wt=(et.clientX-St.left)/St.width-.5,_t=(et.clientY-St.top)/St.height-.5,yt=Math.abs(wt)<.85&&_t>-.9&&_t<1.15;pe=yt?wt*.32:0,ve=yt?_t*-.08:0},at=new IntersectionObserver(([et])=>{ne=!!(et!=null&&et.isIntersecting),ne?Me():I()});at.observe(f);const lt=()=>{document.hidden?I():ne&&Me()};return je(),Z(),le||(window.addEventListener("pointermove",E,{passive:!0}),document.addEventListener("visibilitychange",lt),Me()),()=>{I(),at.disconnect(),window.removeEventListener("pointermove",E),document.removeEventListener("visibilitychange",lt),c.current=()=>{},v.forEach(et=>et.dispose()),_.forEach(et=>et.dispose()),M.forEach(et=>et.dispose()),xt==null||xt.dispose(),h.dispose(),f.classList.remove("is-live")}},[]),Ae.useEffect(()=>{c.current()},[o]),w.jsxs("div",{className:"space-still",ref:e,children:[w.jsxs("div",{className:"space-fallback","aria-hidden":"true",children:[w.jsx("span",{className:"fb-lamp"}),w.jsx("span",{className:"fb-screen"}),w.jsx("span",{className:"fb-paper"}),w.jsx("span",{className:"fb-sphere"}),w.jsx("span",{className:"fb-plinth"})]}),w.jsx("canvas",{ref:i,className:"space-canvas"})]})}function Ix({lit:o}){const e=Ae.useRef(null),i=Ae.useRef(o);return i.current=o,Ae.useEffect(()=>{const r=e.current;if(!r)return;let c;try{c=new bS({canvas:r,antialias:!0,alpha:!1,powerPreference:"low-power"})}catch{return}c.setClearColor(1315086,1);const f=new fm;f.background=new he("#14110e");const d=new ei(32,1,.1,20);d.position.set(.15,.85,2.35),d.lookAt(0,.15,0);const h=[],m=[],p=(k,C,D)=>{h.push(k),m.push(C);const U=new _n(k,C);return U.position.set(...D),f.add(U),U};p(new Ms(2.4,.06,1.15),new Fi({color:3812386,roughness:.92}),[0,0,0]),p(new mo(.012,.014,.42,12),new Fi({color:6048829,roughness:.8}),[.28,.24,.05]);const v=p(new mo(.09,.14,.1,16,1,!0),new Fi({color:9272164,roughness:.7,side:Bi,emissive:new he("#f3e2bc"),emissiveIntensity:0}),[.28,.48,.05]).material,_=new Gb(16052454,.18),M=new mm(15980452,0,3.2);M.position.set(.28,.42,.05),f.add(_,M);let R=0,N=i.current?1:0;const y=window.matchMedia("(prefers-reduced-motion: reduce)").matches,S=()=>{const k=r.clientWidth,C=r.clientHeight;if(k<2||C<2)return;const D=Math.min(window.devicePixelRatio||1,1.5);(r.width!==Math.floor(k*D)||r.height!==Math.floor(C*D))&&(c.setPixelRatio(D),c.setSize(k,C,!1),d.aspect=k/C,d.updateProjectionMatrix());const U=i.current?1:0;N+=(U-N)*(y?1:.08),M.intensity=.15+N*7.5,_.intensity=.16+N*.22,v.emissiveIntensity=N*.55,v.color.set(N>.5?"#f4e2bc":"#8d7b64"),c.render(f,d)},O=()=>{R=window.requestAnimationFrame(O),S(),y&&window.cancelAnimationFrame(R)};return O(),()=>{window.cancelAnimationFrame(R),h.forEach(k=>k.dispose()),m.forEach(k=>k.dispose()),c.dispose()}},[]),w.jsx("canvas",{ref:e,className:"shot-light"})}const kw=["once","session","always","deny"];function Bx(o){const{t:e,scene:i,onScene:r}=o,c=3,f=i==="diff"?e.demo.review:i==="drama"?e.demo.video:i==="trust"?e.demo.harbor:e.demo.agent;return w.jsxs("div",{className:o.embed?"window is-embed":"window",children:[w.jsxs("div",{className:"w-title",children:[w.jsx("span",{className:"w-name",children:"yoyo"}),w.jsx("span",{className:"w-module",children:f}),w.jsx("span",{className:"w-model",children:e.model.modelName})]}),w.jsxs("div",{className:"w-body",children:[w.jsxs("aside",{className:"rail",children:[w.jsx("p",{className:"rail-label",children:e.demo.module}),w.jsx("button",{type:"button",className:i==="ask"?"is-on":void 0,onClick:()=>r("ask"),children:e.demo.agent}),w.jsx("button",{type:"button",className:i==="diff"?"is-on":void 0,onClick:()=>r("diff"),children:e.demo.review}),w.jsx("button",{type:"button",className:i==="drama"?"is-on":void 0,onClick:()=>r("drama"),children:e.demo.video}),w.jsx("button",{type:"button",className:i==="trust"?"is-on":void 0,onClick:()=>r("trust"),children:e.demo.harbor}),w.jsx("p",{className:"rail-label",children:e.demo.chats}),w.jsxs("button",{type:"button",className:i==="ask"?"thread is-live is-on":"thread is-live",onClick:()=>r("ask"),children:[e.demo.threadLive,w.jsx("small",{children:e.demo.threadLiveSub})]}),w.jsx("button",{type:"button",className:i==="trust"?"thread is-on":"thread",onClick:()=>r("trust"),children:e.demo.threadIdle}),w.jsx("p",{className:"thread is-quiet",children:e.demo.newChat})]}),w.jsxs("div",{className:"mid",children:[i==="ask"?w.jsx(Xw,{t:e,phase:c,gate:o.gate,onGate:o.onGate}):null,i==="diff"?w.jsx(Ww,{t:e,hunks:o.hunks,onHunk:o.onHunk}):null,i==="drama"?w.jsx(qw,{t:e,episode:o.episode,onEpisode:o.onEpisode,tab:o.videoTab,onTab:o.onVideoTab}):null,i==="trust"?w.jsx(Yw,{t:e,promoted:o.promoted,onCheckout:o.onCheckout,refKey:o.refKey,onRef:o.onRef}):null]}),i==="ask"||i==="diff"?w.jsxs("aside",{className:"queue",children:[w.jsx("p",{className:"rail-label",children:e.demo.queue}),w.jsxs("button",{type:"button",onClick:()=>r("diff"),children:[w.jsx("b",{children:"loop.go"}),w.jsx("span",{children:"+24 −6"})]})]}):null]}),w.jsxs("div",{className:"w-status",children:[w.jsx("span",{children:e.demo.statusFolder}),w.jsx("span",{children:e.model.modelName}),w.jsx("span",{children:e.demo.local})]})]})}function Xw({t:o,phase:e,gate:i,onGate:r}){const c={once:o.gate.onceHint,session:o.gate.sessionHint,always:o.gate.alwaysHint,deny:o.gate.denyHint}[i],f={once:o.demo.allowOnce,session:o.demo.allowSession,always:o.demo.allowAlways,deny:o.demo.reject},d={once:"1",session:"2",always:"3",deny:"Esc"},h=e>0&&e<3;return w.jsxs("div",{className:"pane",children:[w.jsxs("div",{className:"pane-scroll",children:[w.jsx("p",{className:"hello",children:o.demo.hello}),w.jsx("p",{className:"bubble",children:o.demo.bubble}),e>=1?w.jsxs("p",{className:"proc",role:"status",children:[w.jsx("i",{className:h&&e===1?"dot is-live":"dot","aria-hidden":"true"}),w.jsx("span",{children:e===1?o.demo.reading:o.demo.read}),w.jsx("code",{children:"loop.go"})]}):null,e>=2?w.jsxs("p",{className:"proc",children:[w.jsx("i",{className:e===2?"dot is-live":"dot is-done","aria-hidden":"true"}),w.jsx("span",{children:e===2?o.demo.patching:o.demo.patch})]}):null,e>=3?w.jsxs("div",{className:"gate-card",role:"status",children:[w.jsx("p",{className:"gate-kicker",children:o.demo.needsApproval}),w.jsxs("p",{className:"gate-action",children:[w.jsx("strong",{children:o.demo.action}),w.jsx("code",{children:o.demo.path})]}),w.jsx("div",{className:"chips",children:kw.map(m=>w.jsxs("button",{type:"button",className:m===i?"chip is-hot":"chip","data-kind":m,"aria-pressed":m===i,onClick:()=>r(m),children:[f[m],w.jsx("kbd",{children:d[m]})]},m))}),w.jsx("p",{className:"verdict",children:c})]}):null]}),w.jsxs("div",{className:"composer","aria-hidden":"true",children:[w.jsx("span",{className:"plan",children:o.demo.plan}),w.jsx("span",{className:"composer-text",children:o.demo.composer}),w.jsx("span",{className:h?"send is-stop":"send"})]})]})}function Ww({t:o,hunks:e,onHunk:i}){const[r,c]=Ae.useState(!1),f=e.filter(Boolean).length;return Ae.useEffect(()=>{c(!1)},[e]),w.jsxs("div",{className:"pane diff-pane",children:[w.jsxs("div",{className:"diffhead",children:[w.jsx("b",{children:"loop.go"}),w.jsxs("span",{children:[f," ",o.demo.hunks]}),w.jsx("button",{type:"button",className:"apply",disabled:f===0||r,onClick:()=>c(!0),children:r?o.demo.kept:`${o.demo.apply}${f?` ${f}`:""}`})]}),w.jsxs("div",{className:"diff-file",children:[w.jsxs("p",{className:"ctx",children:["func parse(src string) error ","{"]}),w.jsx(zx,{on:e[0],onToggle:()=>i(0),del:"− if err != nil { return }",add:"+ if err != nil { return err }"}),w.jsx(zx,{on:e[1],onToggle:()=>i(1),del:"− log.Print(got)",add:'+ return fmt.Errorf("parse: %w", err)'}),w.jsx("p",{className:"ctx",children:"  return nil"}),w.jsx("p",{className:"ctx",children:"}"}),r?w.jsx("p",{className:"kept-note",children:o.demo.keptNote}):null]})]})}function zx({on:o,onToggle:e,del:i,add:r}){return w.jsxs("button",{type:"button",className:o?"hunk is-on":"hunk","aria-pressed":o,onClick:e,children:[w.jsx("i",{"aria-hidden":"true"}),w.jsx("span",{className:"del",children:i}),w.jsx("span",{className:"add",children:r})]})}function qw({t:o,episode:e,onEpisode:i,tab:r,onTab:c}){const[f,d]=Ae.useState("video"),h=r??f,m=x=>{c==null||c(x),r===void 0&&d(x)},p=[["01",o.demo.ep1],["02",o.demo.ep2],["03",o.demo.ep3]];return w.jsxs("div",{className:"pane drama-pane",children:[w.jsx("div",{className:"vseg",role:"tablist","aria-label":o.demo.video,children:[["create",o.demo.createTab],["video",o.demo.video],["canvas",o.demo.canvasTab]].map(([x,v])=>w.jsx("button",{type:"button",role:"tab","aria-selected":h===x,onClick:()=>m(x),children:v},x))}),h==="create"?w.jsxs("div",{className:"vbody",children:[w.jsx("p",{className:"bubble",children:o.demo.createHint}),w.jsxs("figure",{className:"shot",children:[w.jsx(Ix,{lit:!0}),w.jsx("figcaption",{children:o.demo.createHint})]}),w.jsx("p",{className:"vcopy",children:o.demo.createBody})]}):null,h==="video"?w.jsxs("div",{className:"vbody",children:[w.jsx("p",{className:"bubble",children:o.demo.dramaPrompt}),w.jsx("div",{className:"eps",children:p.map(([x,v])=>w.jsxs("button",{type:"button",className:e===x?"is-on":void 0,"aria-pressed":e===x,onClick:()=>i(x),children:[w.jsx("b",{children:x}),w.jsx("span",{children:v})]},x))}),w.jsxs("figure",{className:"shot",children:[w.jsx(Ix,{lit:e==="02"}),w.jsx("figcaption",{children:e==="02"?o.demo.shotCap:o.demo.createHint})]})]}):null,h==="canvas"?w.jsxs("div",{className:"vbody",children:[w.jsxs("div",{className:"board",children:[p.map(([x,v])=>w.jsxs("button",{type:"button",className:e===x?"is-on":void 0,"aria-pressed":e===x,onClick:()=>i(x),children:[w.jsx("i",{}),w.jsxs("span",{children:[x," ",v]})]},x)),w.jsx("div",{className:"board-empty","aria-hidden":"true"})]}),w.jsx("p",{className:"vcopy",children:o.demo.canvasHint})]}):null]})}function Yw({t:o,promoted:e,onCheckout:i,refKey:r,onRef:c}){const f=e&&r==="active"?o.trust.done:r==="staging"?o.trust.stagingHint:r==="canary"?o.trust.canaryHint:o.trust.activeHint,d=[["staging",o.trust.staging],["canary",o.trust.canary],["active",o.trust.active]];return w.jsxs("div",{className:"pane trust-pane",children:[w.jsx("div",{className:"w-refs",role:"radiogroup","aria-label":o.nav.trust,children:d.map(([h,m])=>w.jsxs("button",{type:"button",role:"radio","data-ref":h,"aria-checked":r===h,onClick:()=>c(h),children:[w.jsx("b",{children:m}),w.jsx("i",{children:h})]},h))}),w.jsx("p",{className:"w-ref-note","aria-live":"polite",children:f}),w.jsx("button",{type:"button",className:"apply",disabled:e||r!=="canary",onClick:i,children:o.trust.checkout})]})}const ra=["ask","diff","drama","trust"],xs="https://github.com/Shenchangxin/yoyo";function jw(){const o=new URLSearchParams(window.location.search).get("theme");return o==="dark"||o==="light"?o:document.documentElement.dataset.theme==="dark"?"dark":"light"}function Zw(){const[o,e]=Ae.useState(()=>mE()),[i,r]=Ae.useState(()=>jw()),[c,f]=Ae.useState(!1),[d,h]=Ae.useState("workstation"),[m,p]=Ae.useState(()=>{const F=new URLSearchParams(window.location.search).get("scene");return F==="ask"||F==="diff"||F==="drama"||F==="trust"?F:"ask"}),[x,v]=Ae.useState("once"),[_,M]=Ae.useState(!1),[R,N]=Ae.useState("canary"),[y,S]=Ae.useState([!0,!0]),[O,k]=Ae.useState("02"),[C,D]=Ae.useState(1),U=C===0?"create":C===2?"canvas":"video",[B,b]=Ae.useState(Nl),[L,X]=Ae.useState(()=>rp(Nl)),[j,J]=Ae.useState("windows"),P=pE[o],q=Ae.useMemo(()=>L.find(F=>F.os===j)||L[0],[L,j]);Ae.useEffect(()=>{var F,gt,G;document.documentElement.dataset.lang=o,document.documentElement.lang=gE(o),document.title=`${P.meta.title} · ${B}`,(F=document.querySelector('meta[name="description"]'))==null||F.setAttribute("content",P.meta.description),(gt=document.querySelector('meta[property="og:title"]'))==null||gt.setAttribute("content",P.meta.title),(G=document.querySelector('meta[property="og:description"]'))==null||G.setAttribute("content",P.meta.description)},[o,P.meta.description,P.meta.title,B]),Ae.useEffect(()=>{var F;document.documentElement.dataset.theme=i,document.documentElement.style.colorScheme=i,(F=document.querySelector('meta[name="theme-color"]'))==null||F.setAttribute("content",i==="dark"?"#121211":"#f7f4ee")},[i]),Ae.useEffect(()=>{const F=window.matchMedia("(prefers-color-scheme: dark)"),gt=()=>{try{if(localStorage.getItem("yoyo-site-theme")||new URLSearchParams(window.location.search).get("theme"))return}catch{return}r(F.matches?"dark":"light")};return F.addEventListener("change",gt),()=>F.removeEventListener("change",gt)},[]),Ae.useEffect(()=>{J(_E()),xE().then(F=>{b(F.version),X(F.assets)})},[]),Ae.useEffect(()=>{const F=window.matchMedia("(prefers-reduced-motion: reduce)").matches,gt=[...document.querySelectorAll("main > section:not(.hero), .foot")];if(F){gt.forEach(st=>st.classList.add("is-in"));return}const G=new IntersectionObserver(st=>{for(const At of st)!At.isIntersecting||!(At.target instanceof HTMLElement)||(At.target.classList.add("is-in"),G.unobserve(At.target))},{threshold:.14,rootMargin:"0px 0px -6% 0px"});return gt.forEach(st=>G.observe(st)),()=>G.disconnect()},[]),Ae.useEffect(()=>(document.body.classList.toggle("menu-open",c),()=>document.body.classList.remove("menu-open")),[c]),Ae.useEffect(()=>{const F=()=>{var G;const gt=window.location.hash.replace(/^#/,"");gt&&((G=document.getElementById(gt))==null||G.scrollIntoView())};return F(),window.addEventListener("hashchange",F),()=>window.removeEventListener("hashchange",F)},[]),Ae.useEffect(()=>{const F=[...document.querySelectorAll("[data-zone]")],gt=new IntersectionObserver(G=>{const st=G.filter(At=>At.isIntersecting).sort((At,Dt)=>Dt.intersectionRatio-At.intersectionRatio)[0];(st==null?void 0:st.target)instanceof HTMLElement&&st.target.dataset.zone&&h(st.target.dataset.zone)},{rootMargin:"-30% 0px -45% 0px",threshold:[.2,.45,.7]});return F.forEach(G=>gt.observe(G)),()=>gt.disconnect()},[]),Ae.useEffect(()=>{const F=gt=>{if(gt.metaKey||gt.ctrlKey||gt.altKey)return;if(c&&gt.key==="Escape"){f(!1);return}const G=gt.target;if(G!=null&&G.closest("input, textarea, select")||m!=="ask")return;const st={1:"once",2:"session",3:"always",Escape:"deny"}[gt.key];st&&v(st)};return document.addEventListener("keydown",F),()=>document.removeEventListener("keydown",F)},[c,m]);const $=F=>{e(F);try{localStorage.setItem("yoyo-site-lang",F)}catch{}},Y=()=>{const F=i==="dark"?"light":"dark";r(F);try{localStorage.setItem("yoyo-site-theme",F)}catch{}},Q=F=>{const gt=ra.indexOf(m),G=F.key==="ArrowRight"?ra[(gt+1)%ra.length]:F.key==="ArrowLeft"?ra[(gt-1+ra.length)%ra.length]:F.key==="Home"?ra[0]:F.key==="End"?ra[ra.length-1]:null;G&&(F.preventDefault(),p(G),window.requestAnimationFrame(()=>{var st;return(st=document.getElementById(`scene-${G}`))==null?void 0:st.focus()}))},pt=()=>{M(!0),N("active")},ot=P.stage[m],vt={ask:P.demo.agent,diff:P.demo.review,drama:P.demo.video,trust:P.nav.trust},Tt=_&&R==="active"?P.trust.done:R==="staging"?P.trust.stagingHint:R==="canary"?P.trust.canaryHint:P.trust.activeHint,Zt=[["l1",P.model.l1,P.model.l1Body],["l2",P.model.l2,P.model.l2Body],["l3",P.model.l3,P.model.l3Body],["l4",P.model.l4,P.model.l4Body]],Yt=[["staging",P.trust.staging,P.trust.stagingHint],["canary",P.trust.canary,P.trust.canaryHint],["active",P.trust.active,P.trust.activeHint]],H=()=>f(!1);return w.jsxs(w.Fragment,{children:[w.jsx("a",{className:"skip",href:"#download",children:P.nav.download}),w.jsxs("header",{className:"nav",children:[w.jsxs("a",{className:"brand",href:"#top",onClick:H,children:[w.jsx(Hx,{}),w.jsx("span",{children:"yoyo"})]}),w.jsxs("nav",{className:"nav-links","aria-label":"On this page",children:[w.jsx("a",{href:"#top","aria-current":d==="workstation"?"true":void 0,children:P.nav.workstation}),w.jsx("a",{href:"#video","aria-current":d==="video"?"true":void 0,children:P.nav.video}),w.jsx("a",{href:"#model","aria-current":d==="model"?"true":void 0,children:P.nav.model}),w.jsx("a",{href:"#trust","aria-current":d==="trust"?"true":void 0,children:P.nav.trust}),w.jsx("a",{href:"#doors","aria-current":d==="doors"?"true":void 0,children:P.nav.doors}),w.jsx("a",{href:"#download","aria-current":d==="download"?"true":void 0,children:P.nav.download})]}),w.jsxs("div",{className:"nav-end",children:[w.jsx(Fx,{lang:o,onPick:$}),w.jsx("button",{type:"button",className:"icon-btn theme-btn","aria-pressed":i==="dark","aria-label":i==="dark"?P.nav.themeToLight:P.nav.themeToDark,onClick:Y,children:i==="dark"?w.jsx(Qw,{}):w.jsx(Jw,{})}),w.jsx("a",{className:"nav-github",href:xs,children:"GitHub"}),w.jsxs("a",{className:"btn btn-ink nav-cta",href:(q==null?void 0:q.href)||Th,"aria-label":P.nav.download,children:[w.jsx("span",{className:"nav-cta-label",children:P.nav.download}),w.jsx("span",{className:"btn-orb","aria-hidden":"true",children:w.jsx(Gx,{})})]}),w.jsx("button",{type:"button",className:"icon-btn menu-btn","aria-expanded":c,"aria-controls":"nav-panel","aria-label":c?P.nav.close:P.nav.menu,onClick:()=>f(F=>!F),children:w.jsx($w,{open:c})})]}),c?w.jsx("button",{type:"button",className:"nav-backdrop","aria-label":P.nav.close,onClick:H}):null,w.jsxs("div",{className:"nav-panel",id:"nav-panel",hidden:!c,children:[w.jsxs("nav",{"aria-label":"On this page",children:[w.jsx("a",{href:"#top",onClick:H,children:P.nav.workstation}),w.jsx("a",{href:"#video",onClick:H,children:P.nav.video}),w.jsx("a",{href:"#model",onClick:H,children:P.nav.model}),w.jsx("a",{href:"#trust",onClick:H,children:P.nav.trust}),w.jsx("a",{href:"#doors",onClick:H,children:P.nav.doors}),w.jsx("a",{href:"#download",onClick:H,children:P.nav.download}),w.jsx("a",{href:xs,children:P.hero.github})]}),w.jsxs("div",{className:"nav-panel-tools",children:[w.jsx(Fx,{lang:o,onPick:$}),w.jsx("button",{type:"button",className:"text-link",onClick:Y,children:i==="dark"?P.nav.themeToLight:P.nav.themeToDark})]})]})]}),w.jsxs("main",{children:[w.jsxs("section",{className:"hero",id:"top","data-zone":"workstation",children:[w.jsx("div",{className:"wrap hero-grid",children:w.jsxs("div",{className:"hero-copy",children:[w.jsxs("p",{className:"kicker",children:["yoyo · ",B]}),w.jsx(Oa,{as:"h1",text:P.hero.title}),w.jsxs("div",{className:"hero-lower",children:[w.jsxs("div",{className:"hero-lead",children:[w.jsx("p",{className:"dek",children:P.hero.dek}),w.jsxs("div",{className:"hero-actions",children:[w.jsxs("a",{className:"btn btn-ink btn-hero",href:(q==null?void 0:q.href)||Th,children:[w.jsxs("span",{children:[P.hero.download," ",B,q?w.jsxs("span",{className:"btn-os",children:[" · ",q.label]}):null]}),w.jsx("span",{className:"btn-orb","aria-hidden":"true",children:w.jsx(Gx,{})})]}),w.jsx("a",{className:"text-link",href:"#download",children:P.hero.builds})]}),w.jsx("p",{className:"premise",children:P.get.premise}),w.jsx("ul",{className:"ledger",children:P.hero.chips.map(F=>w.jsx("li",{children:F},F))})]}),w.jsx("div",{className:"hero-sculpture","aria-hidden":"true",children:w.jsx(Vw,{theme:i})})]})]})}),w.jsxs("div",{className:"wrap hero-stage",children:[w.jsx("div",{id:"stage",role:"tabpanel","aria-labelledby":`scene-${m}`,"aria-describedby":"stage-note",children:w.jsx(Bx,{t:P,scene:m,onScene:p,gate:x,onGate:v,promoted:_,onCheckout:pt,refKey:R,onRef:N,hunks:y,onHunk:F=>S(gt=>F===0?[!gt[0],gt[1]]:[gt[0],!gt[1]]),episode:O,onEpisode:k})}),w.jsx("div",{className:"scene-tabs",role:"tablist","aria-label":P.nav.workstation,onKeyDown:Q,children:ra.map(F=>w.jsx("button",{id:`scene-${F}`,type:"button",role:"tab","aria-selected":m===F,tabIndex:m===F?0:-1,onClick:()=>p(F),children:vt[F]},F))}),w.jsxs("p",{className:"stage-note",id:"stage-note",children:[ot,m==="drama"?w.jsx("a",{className:"text-link",href:"#video",children:P.nav.video}):null]})]})]}),w.jsxs("section",{className:"section section-stack wrap",id:"work","data-zone":"workstation",children:[w.jsxs("header",{className:"section-head",children:[w.jsx("p",{className:"kicker",children:"01"}),w.jsx(Oa,{text:P.turn.title}),w.jsx("p",{className:"dek",children:P.turn.dek})]}),w.jsx("ol",{className:"steps",children:P.turn.steps.map((F,gt)=>w.jsxs("li",{children:[w.jsx("span",{className:"step-n",children:String(gt+1).padStart(2,"0")}),w.jsx("h3",{children:F.title}),w.jsx("p",{children:F.body})]},F.title))})]}),w.jsxs("section",{className:"section section-stack wrap",id:"ability","data-zone":"workstation",children:[w.jsxs("header",{className:"section-head",children:[w.jsx("p",{className:"kicker",children:"02"}),w.jsx(Oa,{text:P.pitch.title}),w.jsx("p",{className:"dek",children:P.pitch.dek})]}),w.jsx("div",{className:"ability-grid",children:P.pitch.cards.map((F,gt)=>w.jsxs("article",{children:[w.jsx("span",{className:"kicker",children:String(gt+1).padStart(2,"0")}),w.jsx("h3",{children:F.title}),w.jsx("p",{children:F.body})]},F.title))})]}),w.jsxs("section",{className:"section section-stack wrap",id:"video","data-zone":"video",children:[w.jsxs("header",{className:"section-head",children:[w.jsx("p",{className:"kicker",children:"03"}),w.jsx(Oa,{text:P.film.title}),w.jsx("p",{className:"dek",children:P.film.dek})]}),w.jsxs("div",{className:"film-grid",children:[w.jsx(Bx,{t:P,scene:"drama",embed:!0,onScene:()=>{},videoTab:U,onVideoTab:F=>D(F==="create"?0:F==="canvas"?2:1),gate:x,onGate:v,promoted:_,onCheckout:pt,refKey:R,onRef:N,hunks:y,onHunk:F=>S(gt=>F===0?[!gt[0],gt[1]]:[gt[0],!gt[1]]),episode:O,onEpisode:k}),w.jsx("div",{className:"film-modes",role:"tablist","aria-label":P.nav.video,children:P.film.modes.map((F,gt)=>w.jsxs("button",{type:"button",role:"tab","aria-selected":C===gt,onClick:()=>D(gt),children:[w.jsx("b",{children:F.title}),w.jsx("span",{children:F.body})]},F.title))})]}),w.jsx("h3",{className:"phases-title",children:P.film.phasesTitle}),w.jsx("ol",{className:"phases",children:P.film.phases.map((F,gt)=>w.jsxs("li",{children:[w.jsx("span",{className:"step-n",children:String(gt+1).padStart(2,"0")}),w.jsx("h3",{children:F.title}),w.jsx("p",{children:F.body})]},F.title))}),w.jsx("p",{className:"apart",children:P.film.apart})]}),w.jsxs("section",{className:"section split wrap",id:"model","data-zone":"model",children:[w.jsxs("div",{className:"section-copy",children:[w.jsx("p",{className:"kicker",children:"04"}),w.jsx(Oa,{text:P.model.title}),w.jsx("p",{className:"dek",children:P.model.dek})]}),w.jsxs("div",{className:"section-body",children:[w.jsxs("p",{className:"weights",children:[w.jsx("b",{children:P.model.modelName}),w.jsx("span",{children:P.model.weightsStill})]}),w.jsx("ul",{className:"layers",children:Zt.map(([F,gt,G])=>w.jsxs("li",{className:"layer","data-layer":F,children:[w.jsx("b",{children:gt}),w.jsx("p",{children:G})]},F))}),w.jsxs("p",{className:"fp",children:[w.jsx("span",{children:P.model.fp}),w.jsx("code",{children:"refs/models/<fingerprint>/active"})]}),w.jsx("p",{className:"fp-body",children:P.model.fpBody}),w.jsx("p",{className:"more",children:w.jsx("a",{href:`${xs}/blob/master/docs/architecture/invariants.md`,children:P.foot.invariants})})]})]}),w.jsxs("section",{className:"section split wrap",id:"trust","data-zone":"trust",children:[w.jsxs("div",{className:"section-copy",children:[w.jsx("p",{className:"kicker",children:"05"}),w.jsx(Oa,{id:"trust-title",text:P.trust.title}),w.jsx("p",{className:"dek",children:P.trust.dek})]}),w.jsxs("div",{className:"section-body",children:[w.jsx("div",{className:"track",role:"radiogroup","aria-labelledby":"trust-title",children:Yt.map(([F,gt])=>w.jsxs("button",{type:"button",role:"radio","data-ref":F,"aria-checked":R===F,onClick:()=>N(F),children:[w.jsx("b",{children:gt}),w.jsx("i",{children:F})]},F))}),w.jsx("p",{className:"ref-note","aria-live":"polite",children:Tt}),w.jsx("button",{type:"button",className:"btn btn-ink",disabled:_||R!=="canary",onClick:pt,children:P.trust.checkout})]})]}),w.jsxs("section",{className:"section section-stack wrap",id:"doors","data-zone":"doors",children:[w.jsxs("header",{className:"section-head",children:[w.jsx("p",{className:"kicker",children:"06"}),w.jsx(Oa,{text:P.doors.title}),w.jsx("p",{className:"dek",children:P.doors.dek})]}),w.jsxs("ol",{className:"door-grid",children:[w.jsxs("li",{children:[w.jsx("h3",{children:P.doors.desktop}),w.jsx("p",{children:P.doors.desktopBody})]}),w.jsxs("li",{children:[w.jsx("h3",{children:P.doors.browser}),w.jsx("p",{children:P.doors.browserBody}),w.jsx("code",{children:"yoyo serve"})]}),w.jsxs("li",{children:[w.jsx("h3",{children:P.doors.cli}),w.jsx("p",{children:P.doors.cliBody}),w.jsx("code",{children:"yoyo run"})]})]}),w.jsxs("div",{className:"aside",children:[w.jsxs("article",{children:[w.jsx("h3",{children:P.doors.skills}),w.jsx("p",{children:w.jsx(Kw,{text:P.doors.skillsBody})})]}),w.jsxs("article",{children:[w.jsx("h3",{children:P.doors.budget}),w.jsx("p",{children:P.doors.budgetBody})]})]})]}),w.jsxs("section",{className:"section split wrap",id:"limits","data-zone":"doors",children:[w.jsxs("div",{className:"section-copy",children:[w.jsx("p",{className:"kicker",children:"07"}),w.jsx(Oa,{text:P.limits.title}),w.jsx("p",{className:"dek",children:P.limits.dek})]}),w.jsx("ul",{className:"limits",children:P.limits.items.map(F=>w.jsx("li",{children:F},F))})]}),w.jsxs("section",{className:"section split wrap",id:"download","data-zone":"download",children:[w.jsxs("div",{className:"section-copy",children:[w.jsxs("p",{className:"kicker",children:[P.get.version," ",B]}),w.jsx(Oa,{text:P.get.title}),w.jsx("p",{className:"dek",children:P.get.dek}),w.jsx("ol",{className:"start",children:P.get.steps.map(F=>w.jsx("li",{children:F},F))})]}),w.jsxs("div",{className:"section-body",children:[w.jsx("div",{className:"downloads",children:L.map(F=>{const gt=F.os===j;return w.jsxs("a",{className:gt?"asset is-you":"asset",href:F.href,children:[w.jsxs("span",{className:"asset-copy",children:[w.jsx("b",{children:F.label}),w.jsx("span",{className:"asset-name",children:F.name})]}),w.jsxs("span",{className:"asset-meta",children:[gt?w.jsx("em",{children:P.get.thisMachine}):null,F.size?w.jsx("span",{children:F.size}):null]})]},F.os)})}),w.jsx("p",{className:"note",children:P.get.note})]})]})]}),w.jsx("footer",{className:"foot",children:w.jsxs("div",{className:"wrap foot-row",children:[w.jsxs("a",{className:"brand",href:"#top",children:[w.jsx(Hx,{}),w.jsx("span",{children:"yoyo"})]}),w.jsxs("nav",{children:[w.jsx("a",{href:xs,children:"GitHub"}),w.jsx("a",{href:Th,children:"Releases"}),w.jsx("a",{href:`${xs}/blob/master/docs/PRODUCT.md`,children:P.foot.product}),w.jsx("a",{href:`${xs}/blob/master/docs/architecture/invariants.md`,children:P.foot.invariants}),w.jsx("a",{href:`${xs}/issues`,children:P.foot.issues}),w.jsx("a",{href:`${xs}/blob/master/LICENSE`,children:P.foot.license})]}),w.jsxs("p",{children:[w.jsx("span",{children:P.foot.stack}),w.jsx("span",{children:"© 2026"})]})]})})]})}function Oa({text:o,as:e="h2",id:i}){return w.jsx(e,{id:i,children:o.split(`
`).map((r,c)=>w.jsx("span",{className:"ln",children:r},c))})}function Kw({text:o}){return w.jsx(w.Fragment,{children:o.split("`").map((e,i)=>i%2?w.jsx("code",{children:e},i):w.jsx("span",{children:e},i))})}function Fx({lang:o,onPick:e}){return w.jsx("div",{className:"lang",role:"group","aria-label":"Language",children:[["en","EN","English"],["zh","中","简体中文"],["ja","日","日本語"]].map(([i,r,c])=>w.jsx("button",{type:"button","aria-label":c,"aria-pressed":o===i,onClick:()=>e(i),children:r},i))})}function Hx(){return w.jsxs("svg",{className:"brand-mark",viewBox:"0 0 32 32",width:"18",height:"18","aria-hidden":"true",children:[w.jsx("rect",{width:"32",height:"32",rx:"8",fill:"currentColor"}),w.jsxs("g",{fill:"none",stroke:"var(--mark-cut)",strokeLinecap:"round",children:[w.jsx("circle",{cx:"16",cy:"10.2",r:"4",strokeWidth:"2.2"}),w.jsx("path",{d:"M8.2 16.4 16 21.2 23.8 16.4",strokeWidth:"2.4"}),w.jsx("path",{d:"M16 21.2v5.2",strokeWidth:"2.4"})]})]})}function Gx(){return w.jsx("svg",{viewBox:"0 0 16 16",width:"14",height:"14","aria-hidden":"true",children:w.jsx("path",{d:"M8 2.2v7.2M5.2 7.2 8 10l2.8-2.8M3.2 13.2h9.6",fill:"none",stroke:"currentColor",strokeWidth:"1.6",strokeLinecap:"round",strokeLinejoin:"round"})})}function Qw(){return w.jsxs("svg",{viewBox:"0 0 24 24",width:"18",height:"18","aria-hidden":"true",children:[w.jsx("circle",{cx:"12",cy:"12",r:"3.5",fill:"none",stroke:"currentColor",strokeWidth:"1.6"}),w.jsx("path",{d:"M12 3v2.2M12 18.8V21M3 12h2.2M18.8 12H21M5.4 5.4l1.6 1.6M17 17l1.6 1.6M18.6 5.4 17 7M7 17l-1.6 1.6",fill:"none",stroke:"currentColor",strokeWidth:"1.6",strokeLinecap:"round"})]})}function Jw(){return w.jsx("svg",{viewBox:"0 0 24 24",width:"18",height:"18","aria-hidden":"true",children:w.jsx("path",{d:"M15.2 3.4A7.4 7.4 0 1 0 20.6 14 5.8 5.8 0 0 1 15.2 3.4Z",fill:"none",stroke:"currentColor",strokeWidth:"1.6",strokeLinejoin:"round"})})}function $w({open:o}){return w.jsx("svg",{viewBox:"0 0 24 24",width:"18",height:"18","aria-hidden":"true",children:o?w.jsx("path",{d:"M6 6l12 12M18 6 6 18",fill:"none",stroke:"currentColor",strokeWidth:"1.6",strokeLinecap:"round"}):w.jsx("path",{d:"M4 8h16M4 16h16",fill:"none",stroke:"currentColor",strokeWidth:"1.6",strokeLinecap:"round"})})}hE.createRoot(document.getElementById("root")).render(w.jsx(Ae.StrictMode,{children:w.jsx(Zw,{})}));
