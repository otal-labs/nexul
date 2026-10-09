import { TextClassContext } from '@/components/ui/text';
import { cn } from '@/lib/utils';
import { cva, type VariantProps } from 'class-variance-authority';
import { Platform, Pressable } from 'react-native';
import Animated, { useAnimatedStyle, useSharedValue, withTiming } from 'react-native-reanimated';

import { ease, useReducedMotion } from '@/lib/motion';

const buttonVariants = cva(
  cn(
    'group shrink-0 flex-row items-center justify-center gap-2 rounded-md shadow-none',
    Platform.select({
      web: "focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive whitespace-nowrap outline-none transition-all focus-visible:ring-[3px] disabled:pointer-events-none [&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0",
    })
  ),
  {
    variants: {
      variant: {
        default: cn(
          'bg-brand active:bg-brand/85',
          Platform.select({ web: 'hover:bg-brand/90' })
        ),
        destructive: cn(
          'bg-destructive-fill active:bg-destructive-fill/85',
          Platform.select({
            web: 'hover:bg-destructive/90 focus-visible:ring-destructive/20 dark:focus-visible:ring-destructive/40',
          })
        ),
        outline: cn(
          'border-input bg-card/60 active:bg-accent border',
          Platform.select({
            web: 'hover:bg-accent dark:hover:bg-input/50',
          })
        ),
        secondary: cn(
          'bg-secondary active:bg-secondary/80',
          Platform.select({ web: 'hover:bg-secondary/80' })
        ),
        ghost: cn(
          'active:bg-accent',
          Platform.select({ web: 'hover:bg-accent dark:hover:bg-accent/50' })
        ),
        link: '',
      },
      size: {
        default: cn('h-12 px-5', Platform.select({ web: 'has-[>svg]:px-3' })),
        sm: cn('h-11 gap-1.5 rounded-md px-3', Platform.select({ web: 'has-[>svg]:px-2.5' })),
        lg: cn('h-12 rounded-md px-6', Platform.select({ web: 'has-[>svg]:px-4' })),
        icon: 'h-11 w-11',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  }
);

const buttonTextVariants = cva(
  cn(
    'text-foreground text-[15px] font-medium',
    Platform.select({ web: 'pointer-events-none transition-colors' })
  ),
  {
    variants: {
      variant: {
        default: 'text-brand-foreground font-semibold',
        destructive: 'text-white font-semibold',
        outline: cn(
          'group-active:text-accent-foreground',
          Platform.select({ web: 'group-hover:text-accent-foreground' })
        ),
        secondary: 'text-secondary-foreground',
        ghost: 'group-active:text-accent-foreground',
        link: cn(
          'text-primary group-active:underline',
          Platform.select({ web: 'underline-offset-4 hover:underline group-hover:underline' })
        ),
      },
      size: {
        default: '',
        sm: '',
        lg: '',
        icon: '',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  }
);

type ButtonProps = Omit<React.ComponentProps<typeof Pressable>, 'ref'> & VariantProps<typeof buttonVariants>;

const AnimatedPressable = Animated.createAnimatedComponent(Pressable);

// A labelled button presses to 0.97 on the finger's touch and lets go over 150ms, as on the web; never under reduced motion.
function Button({ className, variant, size, onPressIn, onPressOut, ...props }: ButtonProps) {
  const reduced = useReducedMotion();
  const pressed = useSharedValue(1);
  const style = useAnimatedStyle(() => ({ transform: [{ scale: pressed.get() }] }));
  return (
    <TextClassContext.Provider value={buttonTextVariants({ variant, size })}>
      <AnimatedPressable
        className={cn(props.disabled && 'opacity-50', buttonVariants({ variant, size }), className)}
        role="button"
        style={style}
        onPressIn={(e) => {
          if (!reduced) pressed.set(withTiming(0.97, { duration: 100, easing: ease.out }));
          onPressIn?.(e);
        }}
        onPressOut={(e) => {
          pressed.set(withTiming(1, { duration: 150, easing: ease.out }));
          onPressOut?.(e);
        }}
        {...props}
      />
    </TextClassContext.Provider>
  );
}

export { Button, buttonTextVariants, buttonVariants };
export type { ButtonProps };
