// Android browser chrome and the keyboard can resize only the visual viewport.
// Leave pinch zoom to the browser instead of resizing the app while zoomed in.
export function trackMobileViewport(win=window,doc=document){
 let frame=0
 const apply=()=>{frame=0;const vv=win.visualViewport;if(vv&&Math.abs(vv.scale-1)>0.01)return;const height=vv?.height||win.innerHeight;if(!height)return;doc.documentElement.style.setProperty('--app-visible-height',`${Math.round(height)}px`);doc.documentElement.style.setProperty('--app-visible-top',`${Math.max(0,Math.round(vv?.offsetTop||0))}px`)}
 const schedule=()=>{if(!frame)frame=win.requestAnimationFrame(apply)}
 const events=[[win,'resize'],[win,'orientationchange'],[win,'pageshow'],[doc,'visibilitychange']]
 if(win.visualViewport)events.push([win.visualViewport,'resize'],[win.visualViewport,'scroll'])
 for(const [target,event] of events)target.addEventListener(event,schedule)
 apply()
 return()=>{for(const [target,event] of events)target.removeEventListener(event,schedule);if(frame)win.cancelAnimationFrame(frame)}
}
