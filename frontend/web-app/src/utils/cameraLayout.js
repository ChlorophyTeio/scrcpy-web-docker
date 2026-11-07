// Size the unrotated frame so its rotated bounds fit (or cover) the viewport.
export function cameraFrameLayout(frame, viewport, rotation = 0, fit = 'contain') {
  const width = frame.width > 0 ? frame.width : 1920
  const height = frame.height > 0 ? frame.height : 1080
  const availableWidth = Math.max(1, viewport.width)
  const availableHeight = Math.max(1, viewport.height)
  const quarterTurn = Math.abs(rotation % 180) === 90
  const ratios = [availableWidth / (quarterTurn ? height : width), availableHeight / (quarterTurn ? width : height)]
  const scale = fit === 'cover' ? Math.max(...ratios) : Math.min(...ratios)
  return { width: width * scale, height: height * scale }
}
